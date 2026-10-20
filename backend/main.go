package main

import (
	"context"
	"crypto/subtle"
	"database/sql"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"time"
	"unicode/utf16"

	_ "github.com/jackc/pgx/v5/stdlib"
)

//go:embed data/schema.sql
var schema string

type app struct {
	db        *sql.DB
	token     string
	topicURL  string
	topicBusy atomic.Bool
}
type question struct {
	ID          int64  `json:"id,omitempty"`
	Question    string `json:"question"`
	Topic       string `json:"topic"`
	TopicSource string `json:"topicSource,omitempty"`
}
type interview struct {
	ID         int64      `json:"id,omitempty"`
	Company    string     `json:"company"`
	Role       string     `json:"role"`
	Field      string     `json:"field"`
	Experience string     `json:"experience"`
	Type       string     `json:"interviewType"`
	Date       string     `json:"interviewDate"`
	Location   string     `json:"location"`
	Questions  []question `json:"questions"`
}

func env(k, fallback string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return fallback
}
func main() {
	db, err := sql.Open("pgx", os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal(err)
	}
	db.SetMaxOpenConns(4)
	db.SetMaxIdleConns(1)
	db.SetConnMaxIdleTime(5 * time.Minute)
	defer db.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	if err = db.PingContext(ctx); err == nil {
		_, err = db.ExecContext(ctx, schema)
	}
	cancel()
	if err != nil {
		log.Fatal(err)
	}
	a := &app{db: db, token: os.Getenv("ADMIN_TOKEN"), topicURL: os.Getenv("TOPIC_SERVICE_URL")}
	s := &http.Server{Addr: env("LISTEN_ADDR", "127.0.0.1:8091"), Handler: a.routes(env("WEB_DIR", "../web")), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 20 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 16 << 10}
	quit, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	go func() {
		<-quit.Done()
		c, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		s.Shutdown(c)
	}()
	log.Printf("Interview Tracker listening on %s", s.Addr)
	if err = s.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}
func output(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if v != nil {
		json.NewEncoder(w).Encode(v)
	}
}
func fail(w http.ResponseWriter, status int, message string) {
	output(w, status, map[string]string{"message": message})
}
func (a *app) internal(w http.ResponseWriter, err error) {
	log.Printf("Database request failed: %v", err)
	fail(w, 500, "Request failed. Please try again.")
}
func (a *app) routes(dir string) http.Handler {
	m := http.NewServeMux()
	m.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		if err := a.db.PingContext(r.Context()); err != nil {
			fail(w, 503, "Database unavailable")
			return
		}
		output(w, 200, map[string]string{"status": "ok"})
	})
	m.HandleFunc("GET /api/interviews", a.browse)
	m.HandleFunc("POST /api/interviews", a.submit)
	m.HandleFunc("POST /api/topics/suggest", a.suggestTopics)
	m.HandleFunc("GET /api/filters", a.filters)
	m.HandleFunc("GET /api/insights", a.insights)
	m.HandleFunc("POST /api/visits", a.visit)
	m.HandleFunc("GET /api/admin/visitors", a.visitors)
	m.HandleFunc("GET /api/admin/session", func(w http.ResponseWriter, r *http.Request) {
		if a.authorize(w, r) {
			output(w, 200, map[string]bool{"authenticated": true})
		}
	})
	m.HandleFunc("DELETE /api/admin/interviews/{id}", a.delete)
	m.HandleFunc("PUT /api/admin/interviews/{id}", a.submit)
	m.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) { fail(w, 404, "Not found") })
	files := http.FileServer(http.Dir(dir))
	m.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		switch r.URL.Path {
		case "/", "/index.html", "/admin.html", "/insights.html", "/insights.js", "/app.js", "/style.css":
			files.ServeHTTP(w, r)
		default:
			http.NotFound(w, r)
		}
	})
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "same-origin")
		w.Header().Set("X-Frame-Options", "DENY")
		if strings.HasPrefix(r.URL.Path, "/api/") {
			w.Header().Set("Cache-Control", "no-store")
		}
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		m.ServeHTTP(w, r.WithContext(ctx))
	})
}
func length(s string) int { return len(utf16.Encode([]rune(s))) }
func text(s string, max int, required bool) bool {
	return length(s) <= max && (!required || strings.TrimSpace(s) != "")
}
func valid(i interview) bool {
	d, err := time.Parse("2006-01-02", i.Date)
	return (i.Date == "" || (err == nil && d.Format("2006-01-02") <= time.Now().UTC().Format("2006-01-02"))) && text(i.Company, 120, true) && text(i.Role, 160, false) && text(i.Field, 80, false) && text(i.Location, 120, false) && slicesContains([]string{"", "0–2", "3–5", "6–10", "10+"}, i.Experience) && slicesContains([]string{"", "Online", "On-site", "Walk-in", "Phone"}, i.Type) && len(i.Questions) <= 50
}
func slicesContains(items []string, s string) bool {
	for _, v := range items {
		if v == s {
			return true
		}
	}
	return false
}
func (a *app) submit(w http.ResponseWriter, r *http.Request) {
	editing := r.Method == http.MethodPut
	var id int64
	if editing {
		if !a.authorize(w, r) {
			return
		}
		var err error
		id, err = strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil || id <= 0 {
			fail(w, 400, "Invalid interview id")
			return
		}
	}
	var i interview
	d := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	d.DisallowUnknownFields()
	if err := d.Decode(&i); err != nil {
		fail(w, 400, "Invalid submission")
		return
	}
	if d.Decode(new(any)) != io.EOF || !valid(i) {
		fail(w, 400, "Invalid submission")
		return
	}
	for _, q := range i.Questions {
		if !text(q.Question, 2000, false) || !text(q.Topic, 80, false) || (q.TopicSource != "" && q.TopicSource != "manual" && q.TopicSource != "auto") {
			fail(w, 400, "Question and topic must be within length limits")
			return
		}
	}
	tx, err := a.db.BeginTx(r.Context(), nil)
	if err != nil {
		a.internal(w, err)
		return
	}
	defer tx.Rollback()
	if editing {
		err = tx.QueryRowContext(r.Context(), `UPDATE interview SET company=$1,role=$2,experience=$3,interview_type=$4,interview_date=NULLIF($5,'')::date,location=$6,field=$7 WHERE id=$8 RETURNING id`, strings.TrimSpace(i.Company), strings.TrimSpace(i.Role), i.Experience, i.Type, i.Date, strings.TrimSpace(i.Location), strings.TrimSpace(i.Field), id).Scan(&i.ID)
		if err == sql.ErrNoRows {
			fail(w, 404, "Interview not found")
			return
		}
		if err == nil {
			_, err = tx.ExecContext(r.Context(), "DELETE FROM question WHERE interview_id=$1", i.ID)
		}
	} else {
		err = tx.QueryRowContext(r.Context(), `INSERT INTO interview(company,role,experience,interview_type,interview_date,location,field) VALUES($1,$2,$3,$4,NULLIF($5,'')::date,$6,$7) RETURNING id`, strings.TrimSpace(i.Company), strings.TrimSpace(i.Role), i.Experience, i.Type, i.Date, strings.TrimSpace(i.Location), strings.TrimSpace(i.Field)).Scan(&i.ID)
	}
	if err != nil {
		a.internal(w, err)
		return
	}
	for n, q := range i.Questions {
		if q.TopicSource == "" {
			q.TopicSource = "manual"
		}
		if _, err = tx.ExecContext(r.Context(), `INSERT INTO question(interview_id,position,question,topic,topic_source) VALUES($1,$2,$3,$4,$5)`, i.ID, n, strings.TrimSpace(q.Question), strings.TrimSpace(q.Topic), q.TopicSource); err != nil {
			a.internal(w, err)
			return
		}
	}
	if err = tx.Commit(); err != nil {
		a.internal(w, err)
		return
	}
	status := http.StatusCreated
	if editing {
		status = http.StatusOK
	}
	output(w, status, map[string]int64{"id": i.ID})
}
func parseInt(s string, fallback, min, max int) (int, error) {
	if s == "" {
		return fallback, nil
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < min || n > max {
		return 0, fmt.Errorf("invalid number")
	}
	return n, nil
}
func (a *app) browse(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	search, company, experience := q.Get("search"), q.Get("company"), q.Get("experience")
	topics := []string{}
	for _, topic := range q["topic"] {
		if !text(topic, 80, false) {
			fail(w, 400, "Invalid topic")
			return
		}
		if topic = strings.TrimSpace(topic); topic != "" {
			topics = append(topics, strings.ToLower(topic))
		}
	}
	topicJSON, _ := json.Marshal(topics)
	field, role := q.Get("field"), q.Get("role")
	page, e1 := parseInt(q.Get("page"), 0, 0, 100000)
	year, e2 := parseInt(q.Get("year"), 0, 1900, 2200)
	if e1 != nil || e2 != nil || !text(search, 200, false) || !text(company, 120, false) || !text(experience, 20, false) || !text(field, 80, false) || !text(role, 160, false) {
		fail(w, 400, "Invalid filter or page value")
		return
	}
	search = strings.TrimSpace(search)
	pattern := "%" + strings.NewReplacer("!", "!!", "%", "!%", "_", "!_").Replace(strings.ToLower(search)) + "%"
	args := []any{strings.TrimSpace(company), experience, year, string(topicJSON), search, pattern, strings.TrimSpace(field), strings.TrimSpace(role)}
	where := ` WHERE ($1='' OR lower(i.company)=lower($1)) AND ($2='' OR i.experience=$2) AND ($3=0 OR extract(year from i.interview_date)=$3)
 AND ($4='[]' OR EXISTS(SELECT 1 FROM question t WHERE t.interview_id=i.id AND lower(t.topic) IN (SELECT jsonb_array_elements_text($4::jsonb))))
 AND ($7='' OR lower(i.field)=lower($7)) AND ($8='' OR lower(i.role)=lower($8))
 AND ($5='' OR lower(concat_ws(' ',i.field,i.company,i.role)) LIKE $6 ESCAPE '!' OR interview_search_matches($5,concat_ws(' ',i.field,i.company,i.role)) OR EXISTS(
 SELECT 1 FROM question q WHERE q.interview_id=i.id
 AND ($4='[]' OR lower(q.topic) IN (SELECT jsonb_array_elements_text($4::jsonb)))
 AND interview_search_matches($5,concat_ws(' ',i.field,i.company,i.role,q.question,q.topic))))`
	var total int64
	if err := a.db.QueryRowContext(r.Context(), "SELECT count(*) FROM interview i"+where, args...).Scan(&total); err != nil {
		a.internal(w, err)
		return
	}
	args = append(args, page*12)
	rows, err := a.db.QueryContext(r.Context(), `SELECT i.id,i.company,i.role,i.experience,i.interview_type,COALESCE(to_char(i.interview_date,'YYYY-MM-DD'),''),i.location,i.field,
 COALESCE((SELECT json_agg(json_build_object('id',q.id,'question',q.question,'topic',q.topic,'topicSource',q.topic_source) ORDER BY q.position,q.id) FROM question q WHERE q.interview_id=i.id AND ($4='[]' OR lower(q.topic) IN (SELECT jsonb_array_elements_text($4::jsonb))) AND ($5='' OR interview_search_matches($5,concat_ws(' ',i.field,i.company,i.role,q.question,q.topic)))),'[]'::json)
 FROM interview i`+where+` ORDER BY i.interview_date DESC NULLS LAST,i.id DESC LIMIT 12 OFFSET $9`, args...)
	if err != nil {
		a.internal(w, err)
		return
	}
	defer rows.Close()
	items := []interview{}
	for rows.Next() {
		var i interview
		var qs []byte
		if err = rows.Scan(&i.ID, &i.Company, &i.Role, &i.Experience, &i.Type, &i.Date, &i.Location, &i.Field, &qs); err == nil {
			err = json.Unmarshal(qs, &i.Questions)
		}
		if err != nil {
			a.internal(w, err)
			return
		}
		items = append(items, i)
	}
	if err = rows.Err(); err != nil {
		a.internal(w, err)
		return
	}
	output(w, 200, map[string]any{"items": items, "total": total, "page": page, "pageSize": 12})
}
func (a *app) filters(w http.ResponseWriter, r *http.Request) {
	var raw []byte
	err := a.db.QueryRowContext(r.Context(), `SELECT json_build_object(
 'companies',COALESCE((SELECT json_agg(company ORDER BY company) FROM (SELECT DISTINCT company FROM interview) c),'[]'::json),
 'fields',COALESCE((SELECT json_agg(field ORDER BY field) FROM (SELECT DISTINCT field FROM interview WHERE field<>'') f),'[]'::json),
 'roles',COALESCE((SELECT json_agg(role ORDER BY role) FROM (SELECT DISTINCT role FROM interview WHERE role<>'') r),'[]'::json),
 'topics',COALESCE((SELECT json_agg(topic ORDER BY topic) FROM (SELECT DISTINCT topic FROM question WHERE topic<>'') t),'[]'::json),
 'years',COALESCE((SELECT json_agg(year ORDER BY year DESC) FROM (SELECT DISTINCT extract(year from interview_date)::int AS year FROM interview WHERE interview_date IS NOT NULL) y),'[]'::json),
 'interviews',(SELECT count(*) FROM interview),'questions',(SELECT count(*) FROM question))`).Scan(&raw)
	if err != nil {
		a.internal(w, err)
		return
	}
	output(w, 200, json.RawMessage(raw))
}
func (a *app) authorize(w http.ResponseWriter, r *http.Request) bool {
	if strings.TrimSpace(a.token) == "" {
		fail(w, 503, "Moderation is not configured")
		return false
	}
	if subtle.ConstantTimeCompare([]byte(r.Header.Get("Authorization")), []byte("Bearer "+a.token)) != 1 {
		fail(w, 401, "Invalid admin token")
		return false
	}
	return true
}
func (a *app) delete(w http.ResponseWriter, r *http.Request) {
	if !a.authorize(w, r) {
		return
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		fail(w, 400, "Invalid interview id")
		return
	}
	res, err := a.db.ExecContext(r.Context(), "DELETE FROM interview WHERE id=$1", id)
	if err != nil {
		a.internal(w, err)
		return
	}
	n, err := res.RowsAffected()
	if err != nil {
		a.internal(w, err)
		return
	}
	if n == 0 {
		fail(w, 404, "Interview not found")
		return
	}
	output(w, 204, nil)
}

package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestValidation(t *testing.T) {
	if !valid(interview{Company: "Company only"}) {
		t.Fatal("company-only submission rejected")
	}
	if valid(interview{}) {
		t.Fatal("missing company accepted")
	}
	i := interview{Company: "Example", Role: "Java Developer", Experience: "6–10", Type: "Online", Date: "2026-01-15", Questions: []question{{Question: "Java?", Topic: "Java"}}}
	if !valid(i) {
		t.Fatal("valid submission rejected")
	}
	i.Date = "2099-01-15"
	if valid(i) {
		t.Fatal("future date accepted")
	}
	i.Date = "2026-01-15"
	i.Company = "   "
	if valid(i) {
		t.Fatal("blank company accepted")
	}
}
func TestPostgresParity(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set TEST_DATABASE_URL for PostgreSQL integration checks")
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	name := fmt.Sprintf("interview_test_%d", time.Now().UnixNano())
	if _, err = db.Exec("CREATE SCHEMA " + name); err != nil {
		t.Fatal(err)
	}
	defer db.Exec("DROP SCHEMA " + name + " CASCADE")
	u, err := url.Parse(dsn)
	if err != nil {
		t.Fatal(err)
	}
	q := u.Query()
	q.Set("search_path", name)
	u.RawQuery = q.Encode()
	isolated, err := sql.Open("pgx", u.String())
	if err != nil {
		t.Fatal(err)
	}
	defer isolated.Close()
	isolated.SetMaxOpenConns(4)
	if _, err = isolated.Exec(schema); err != nil {
		t.Fatal(err)
	}
	a := &app{db: isolated, token: "test-secret"}
	h := a.routes("../web")
	request := func(method, path, body, token string, status int) *httptest.ResponseRecorder {
		t.Helper()
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		if token != "" {
			r.Header.Set("Authorization", "Bearer "+token)
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != status {
			t.Fatalf("%s %s: status %d, body %s", method, path, w.Code, w.Body.String())
		}
		return w
	}
	minimal := request("POST", "/api/interviews", `{"company":"Company only"}`, "", 201)
	var minimalID struct{ ID int64 }
	json.Unmarshal(minimal.Body.Bytes(), &minimalID)
	minimal = request("GET", "/api/interviews?company=Company%20only", "", "", 200)
	var minimalList struct{ Items []interview }
	if err := json.Unmarshal(minimal.Body.Bytes(), &minimalList); err != nil || len(minimalList.Items) != 1 || minimalList.Items[0].Date != "" || len(minimalList.Items[0].Questions) != 0 {
		t.Fatalf("company-only report did not round-trip: %s", minimal.Body.String())
	}
	request("DELETE", fmt.Sprintf("/api/admin/interviews/%d", minimalID.ID), "", "test-secret", 204)
	body := `{"company":"Capgemini","role":"Java Developer","experience":"6–10","interviewType":"Walk-in","interviewDate":"2026-01-15","location":"Bengaluru","questions":[{"question":"How does flatMap work?","topic":"Java"},{"question":"What is @CrossOrigin?","topic":"Spring Boot"}]}`
	w := request("POST", "/api/interviews", body, "", 201)
	var id struct{ ID int64 }
	json.Unmarshal(w.Body.Bytes(), &id)
	t.Run("insights", func(t *testing.T) {
		// A second, undated report verifies that question joins do not inflate totals.
		extra := request("POST", "/api/interviews", `{"company":"CAPGEMINI","questions":[{"question":"Another Java question","topic":"java"},{"question":"No topic","topic":""}]}`, "", 201)
		var extraID struct{ ID int64 }
		json.Unmarshal(extra.Body.Bytes(), &extraID)
		defer request("DELETE", fmt.Sprintf("/api/admin/interviews/%d", extraID.ID), "", "test-secret", 204)
		type insightStats struct {
			Reports, Questions, Companies, Undated, Untagged int
			Topics                                           []struct {
				Label          string
				Count, Reports int
			}
			CompanyBreakdown []struct {
				Label string
				Count int
			}
			Months []struct {
				Label string
				Count int
			}
		}
		var stats insightStats
		read := func(path string) {
			t.Helper()
			stats = insightStats{}
			result := request("GET", path, "", "", 200)
			if err := json.Unmarshal(result.Body.Bytes(), &stats); err != nil {
				t.Fatal(err)
			}
		}
		read("/api/insights")
		if stats.Reports != 2 || stats.Questions != 4 || stats.Companies != 1 || stats.Undated != 1 || stats.Untagged != 1 || len(stats.Topics) != 2 || stats.Topics[0].Count != 2 || stats.Topics[0].Reports != 2 || len(stats.CompanyBreakdown) != 1 || stats.CompanyBreakdown[0].Count != 2 || len(stats.Months) != 1 || stats.Months[0].Label != "2026-01" || stats.Months[0].Count != 1 {
			t.Fatalf("incorrect insights: %+v", stats)
		}
		read("/api/insights?company=capgemini&role=Java%20Developer&experience=6%E2%80%9310&year=2026")
		if stats.Reports != 1 || stats.Questions != 2 || stats.Undated != 0 {
			t.Fatalf("filters ignored: %+v", stats)
		}
		read("/api/insights?field=absent")
		if stats.Reports != 0 || len(stats.Topics) != 0 || len(stats.Months) != 0 {
			t.Fatalf("empty selection: %+v", stats)
		}
		request("GET", "/api/insights?year=invalid", "", "", 400)
		request("GET", "/api/insights?experience=invalid", "", "", 400)
		request("GET", "/insights.html", "", "", 200)
		request("GET", "/insights.js", "", "", 200)
	})
	// Simulate a pre-field database and rerun initialization without losing reports.
	if _, err = isolated.Exec("ALTER TABLE interview DROP COLUMN field"); err != nil {
		t.Fatal(err)
	}
	if _, err = isolated.Exec(schema); err != nil {
		t.Fatal(err)
	}
	w = request("GET", "/api/interviews?search=FLATmap&company=capgemini&topic=java&experience=6%E2%80%9310&year=2026", "", "", 200)
	var page struct {
		Total int
		Items []interview
	}
	if err = json.Unmarshal(w.Body.Bytes(), &page); err != nil || page.Total != 1 || len(page.Items) != 1 || len(page.Items[0].Questions) != 1 || page.Items[0].Questions[0].Topic != "Java" {
		t.Fatalf("combined filtering: %s", w.Body.String())
	}
	w = request("GET", "/api/interviews?topic=java&topic=Spring%20Boot", "", "", 200)
	json.Unmarshal(w.Body.Bytes(), &page)
	if page.Total != 1 || len(page.Items[0].Questions) != 2 {
		t.Fatalf("multiple topics: %s", w.Body.String())
	}
	w = request("GET", "/api/interviews", "", "", 200)
	json.Unmarshal(w.Body.Bytes(), &page)
	if page.Total != 1 || len(page.Items[0].Questions) != 2 {
		t.Fatal("unfiltered questions missing")
	}
	for _, term := range []string{"flatmp", "FLATMAP", "capgemni flatmp", "flatmp java", "java flatmp"} {
		w = request("GET", "/api/interviews?search="+url.QueryEscape(term), "", "", 200)
		json.Unmarshal(w.Body.Bytes(), &page)
		if page.Total != 1 || len(page.Items[0].Questions) != 1 || page.Items[0].Questions[0].Topic != "Java" {
			t.Fatalf("fuzzy search %q: %s", term, w.Body.String())
		}
	}
	w = request("GET", "/api/interviews?search=flatmp&topic=Spring%20Boot", "", "", 200)
	json.Unmarshal(w.Body.Bytes(), &page)
	if page.Total != 0 {
		t.Fatal("search ignored topic filter")
	}
	for _, term := range []string{"%25", "_", "!", "absent"} {
		w = request("GET", "/api/interviews?search="+term, "", "", 200)
		json.Unmarshal(w.Body.Bytes(), &page)
		if page.Total != 0 {
			t.Fatalf("literal search matched: %s", term)
		}
	}
	request("GET", "/api/interviews?page=-1", "", "", 400)
	request("GET", "/api/interviews?year=2201", "", "", 400)
	request("POST", "/api/interviews", strings.Replace(body, "How does flatMap work?", strings.Repeat("x", 2001), 1), "", 400)
	request("POST", "/api/interviews", strings.Replace(body, "2026-01-15", "2099-01-15", 1), "", 400)
	request("POST", "/api/interviews", strings.Replace(body, "6–10", "invalid", 1), "", 400)
	request("POST", "/api/interviews", body+body, "", 400)
	path := fmt.Sprintf("/api/admin/interviews/%d", id.ID)
	request("DELETE", path, "", "", 401)
	request("DELETE", path, "", "wrong", 401)
	request("GET", "/api/admin/session", "", "test-secret", 200)
	request("GET", "/api/filters", "", "", 200)
	for n := 0; n < 12; n++ {
		request("POST", "/api/interviews", body, "", 201)
	}
	w = request("GET", "/api/interviews?page=1", "", "", 200)
	json.Unmarshal(w.Body.Bytes(), &page)
	if page.Total != 13 || len(page.Items) != 1 {
		t.Fatal("pagination failed")
	}
	general := strings.Replace(body, `"role":"Java Developer"`, `"role":"Accountant","field":"Finance"`, 1)
	request("POST", "/api/interviews", general, "", 201)
	w = request("GET", "/api/interviews?field=finance&role=accountant&company=capgemini&topic=java&year=2026", "", "", 200)
	json.Unmarshal(w.Body.Bytes(), &page)
	if page.Total != 1 || page.Items[0].Field != "Finance" || page.Items[0].Role != "Accountant" {
		t.Fatalf("field and role filtering failed: %s", w.Body.String())
	}
	w = request("GET", "/api/interviews?field=Finance&role=Developer", "", "", 200)
	json.Unmarshal(w.Body.Bytes(), &page)
	if page.Total != 0 {
		t.Fatal("mismatched role returned a report")
	}
	w = request("GET", "/api/interviews?search=finance", "", "", 200)
	json.Unmarshal(w.Body.Bytes(), &page)
	if page.Total != 1 {
		t.Fatal("field search failed")
	}
	w = request("GET", "/api/filters", "", "", 200)
	var filters struct {
		Fields []string
		Roles  []string
	}
	json.Unmarshal(w.Body.Bytes(), &filters)
	if !slicesContains(filters.Fields, "Finance") || !slicesContains(filters.Roles, "Accountant") {
		t.Fatal("filter options missing")
	}
	request("GET", "/api/interviews?field="+strings.Repeat("a", 81), "", "", 400)
	request("GET", "/api/interviews?role="+strings.Repeat("a", 161), "", "", 400)
	request("POST", "/api/interviews", strings.Replace(general, `"field":"Finance"`, `"field":"`+strings.Repeat("a", 81)+`"`, 1), "", 400)
	request("PUT", path, general, "", 401)
	request("PUT", path, general, "wrong", 401)
	request("PUT", "/api/admin/interviews/invalid", general, "test-secret", 400)
	request("PUT", "/api/admin/interviews/9223372036854775807", general, "test-secret", 404)
	request("PUT", path, strings.Replace(general, "2026-01-15", "2099-01-15", 1), "test-secret", 400)
	edited := strings.Replace(general, "Capgemini", "Edited company", 1)
	edited = strings.Replace(edited, "How does flatMap work?", "Updated question?", 1)
	edited = strings.Replace(edited, `"topic":"Java"`, `"topic":"Updated topic"`, 1)
	request("PUT", path, edited, "test-secret", 200)
	w = request("GET", "/api/interviews?company=Edited%20company", "", "", 200)
	json.Unmarshal(w.Body.Bytes(), &page)
	if page.Total != 1 || page.Items[0].ID != id.ID || page.Items[0].Role != "Accountant" || page.Items[0].Field != "Finance" || len(page.Items[0].Questions) != 2 || page.Items[0].Questions[0].Question != "Updated question?" {
		t.Fatalf("edit did not preserve report identity and replace contents: %s", w.Body.String())
	}
	w = request("GET", "/api/filters", "", "", 200)
	var topicFilters struct{ Topics []string }
	json.Unmarshal(w.Body.Bytes(), &topicFilters)
	if !slicesContains(topicFilters.Topics, "Updated topic") {
		t.Fatal("edited topic absent from filters")
	}
	w = request("GET", "/api/interviews?company=Edited%20company&topic=Updated%20topic", "", "", 200)
	json.Unmarshal(w.Body.Bytes(), &page)
	if page.Total != 1 || len(page.Items[0].Questions) != 1 || page.Items[0].Questions[0].Topic != "Updated topic" {
		t.Fatal("edited topic filtering failed")
	}
	w = request("GET", "/api/interviews?company=Edited%20company&topic=Java", "", "", 200)
	json.Unmarshal(w.Body.Bytes(), &page)
	if page.Total != 0 {
		t.Fatal("old topic still matched edited report")
	}
	request("PUT", path, strings.Replace(edited, "Updated question?", strings.Repeat("x", 2001), 1), "test-secret", 400)
	w = request("GET", "/api/interviews?company=Edited%20company", "", "", 200)
	json.Unmarshal(w.Body.Bytes(), &page)
	if page.Items[0].Questions[0].Question != "Updated question?" {
		t.Fatal("invalid edit changed stored questions")
	}
	// Suggestions learn only manually assigned topics, never their own outputs.
	autoBody := strings.Replace(body, "Capgemini", "Auto topic report", 1)
	autoBody = strings.Replace(autoBody, "How does flatMap work?", "Automatically classified question?", 1)
	autoBody = strings.Replace(autoBody, `"topic":"Java"`, `"topic":"Java","topicSource":"auto"`, 1)
	request("POST", "/api/interviews", autoBody, "", 201)
	var modelUnavailable atomic.Bool
	model := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if modelUnavailable.Load() {
			w.WriteHeader(503)
			return
		}
		var payload struct {
			Questions []string
			Examples  []question
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Error(err)
		}
		if len(payload.Questions) != 1 {
			t.Error("missing requested questions")
		}
		learned := false
		for _, example := range payload.Examples {
			if example.Question == "Automatically classified question?" {
				t.Error("automatic topic used as training example")
			}
			if example.Question == "Updated question?" && example.Topic == "Updated topic" {
				learned = true
			}
		}
		if !learned {
			t.Error("manual correction was not supplied to model")
		}
		output(w, 200, map[string]any{"suggestions": []topicSuggestion{{Topic: "SQL", Method: "rules"}}})
	}))
	defer model.Close()
	a.topicURL = model.URL
	w = request("POST", "/api/topics/suggest", `{"questions":["Find the second highest salary"]}`, "", 200)
	if !strings.Contains(w.Body.String(), `"topic":"SQL"`) {
		t.Fatal("missing model suggestion")
	}
	request("POST", "/api/topics/suggest", `{"questions":[]}`, "", 400)
	request("POST", "/api/topics/suggest", `{"questions":[""]}`, "", 400)
	request("POST", "/api/topics/suggest", `{"questions":["Question"],"unknown":true}`, "", 400)
	a.topicBusy.Store(true)
	request("POST", "/api/topics/suggest", `{"questions":["Question"]}`, "", 429)
	a.topicBusy.Store(false)
	modelUnavailable.Store(true)
	request("POST", "/api/topics/suggest", `{"questions":["Question"]}`, "", 503)
	request("DELETE", path, "", "test-secret", 204)
	request("DELETE", path, "", "test-secret", 404)
	var count int
	if err = isolated.QueryRowContext(context.Background(), "SELECT count(*) FROM question WHERE interview_id=$1", id.ID).Scan(&count); err != nil || count != 0 {
		t.Fatal("cascade failed")
	}
	request("GET", "/", "", "", 200)
	request("GET", "/admin.html", "", "", 200)
	request("GET", "/api/health", "", "", 200)
	request(http.MethodGet, "/main.go", "", "", 404)
	request("GET", "/api/admin/visitors", "", "", 401)
	request("GET", "/api/admin/visitors", "", "wrong", 401)
	var stats struct{ Online, Today int64 }
	readStats := func(online, today int64) {
		t.Helper()
		w := request("GET", "/api/admin/visitors", "", "test-secret", 200)
		if err := json.Unmarshal(w.Body.Bytes(), &stats); err != nil || stats.Online != online || stats.Today != today {
			t.Fatalf("visitor counts: %s", w.Body.String())
		}
	}
	readStats(0, 0)
	w = request("POST", "/api/visits", "", "", 204)
	cookies := w.Result().Cookies()
	if len(cookies) != 1 || !cookies[0].HttpOnly || cookies[0].SameSite != http.SameSiteStrictMode || cookies[0].Expires.Hour() != 0 {
		t.Fatalf("visitor cookie: %v", cookies)
	}
	readStats(1, 1)
	r := httptest.NewRequest("POST", "/api/visits", nil)
	r.AddCookie(cookies[0])
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 204 {
		t.Fatalf("repeat visit: %d", w.Code)
	}
	readStats(1, 1)
	if _, err := isolated.Exec("UPDATE visitor_presence SET last_seen=CURRENT_TIMESTAMP - INTERVAL '6 minutes'"); err != nil {
		t.Fatal(err)
	}
	readStats(0, 1)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	readStats(1, 1)
	request("POST", "/api/visits", "", "", 204)
	readStats(2, 2)
	if _, err := isolated.Exec("INSERT INTO visitor_presence VALUES ((CURRENT_TIMESTAMP AT TIME ZONE 'UTC')::date - 1, 'expired', CURRENT_TIMESTAMP - INTERVAL '1 day')"); err != nil {
		t.Fatal(err)
	}
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	var visitors int
	if err := isolated.QueryRow("SELECT count(*) FROM visitor_presence").Scan(&visitors); err != nil || visitors != 2 {
		t.Fatalf("expired visitors not removed: %d, %v", visitors, err)
	}
	r.Header.Set("Sec-Fetch-Site", "cross-site")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 403 {
		t.Fatalf("cross-site heartbeat: %d", w.Code)
	}
}

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
	"testing"
	"time"
)

func TestValidation(t *testing.T) {
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
	body := `{"company":"Capgemini","role":"Java Developer","experience":"6–10","interviewType":"Walk-in","interviewDate":"2026-01-15","location":"Bengaluru","questions":[{"question":"How does flatMap work?","topic":"Java"},{"question":"What is @CrossOrigin?","topic":"Spring Boot"}]}`
	w := request("POST", "/api/interviews", body, "", 201)
	var id struct{ ID int64 }
	json.Unmarshal(w.Body.Bytes(), &id)
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
	if err = json.Unmarshal(w.Body.Bytes(), &page); err != nil || page.Total != 1 || len(page.Items) != 1 || len(page.Items[0].Questions) != 2 {
		t.Fatalf("combined filtering: %s", w.Body.String())
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
	request("POST", "/api/interviews", strings.Replace(body, "How does flatMap work?", "   ", 1), "", 400)
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
}

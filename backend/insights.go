package main

import (
	"encoding/json"
	"net/http"
	"strings"
)

func (a *app) insights(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	year, err := parseInt(q.Get("year"), 0, 1900, 2200)
	if err != nil || !text(q.Get("company"), 120, false) || !text(q.Get("field"), 80, false) || !text(q.Get("role"), 160, false) || !slicesContains([]string{"", "0–2", "3–5", "6–10", "10+"}, q.Get("experience")) {
		fail(w, 400, "Invalid insight filter")
		return
	}
	var raw []byte
	err = a.db.QueryRowContext(r.Context(), `WITH reports AS (
 SELECT * FROM interview WHERE ($1='' OR lower(company)=lower($1))
 AND ($2='' OR lower(field)=lower($2)) AND ($3='' OR lower(role)=lower($3))
 AND ($4='' OR experience=$4) AND ($5=0 OR extract(year from interview_date)=$5)
 ), questions AS (SELECT q.* FROM question q JOIN reports r ON r.id=q.interview_id WHERE trim(q.question)<>'')
 SELECT json_build_object(
 'reports',(SELECT count(*) FROM reports),
 'questions',(SELECT count(*) FROM questions),
 'companies',(SELECT count(DISTINCT lower(company)) FROM reports),
 'undated',(SELECT count(*) FROM reports WHERE interview_date IS NULL),
 'untagged',(SELECT count(*) FROM questions WHERE trim(topic)=''),
 'topics',COALESCE((SELECT json_agg(t) FROM (SELECT min(topic) AS label,count(*) AS count,count(DISTINCT interview_id) AS reports FROM questions WHERE trim(topic)<>'' GROUP BY lower(topic) ORDER BY count(*) DESC,lower(topic) LIMIT 10) t),'[]'::json),
 'companyBreakdown',COALESCE((SELECT json_agg(c) FROM (SELECT min(company) AS label,count(*) AS count FROM reports GROUP BY lower(company) ORDER BY count(*) DESC,lower(company) LIMIT 10) c),'[]'::json),
 'months',COALESCE((SELECT json_agg(m ORDER BY label) FROM (SELECT to_char(interview_date,'YYYY-MM') AS label,count(*) AS count FROM reports WHERE interview_date IS NOT NULL GROUP BY to_char(interview_date,'YYYY-MM')) m),'[]'::json)
 )`, strings.TrimSpace(q.Get("company")), strings.TrimSpace(q.Get("field")), strings.TrimSpace(q.Get("role")), q.Get("experience"), year).Scan(&raw)
	if err != nil {
		a.internal(w, err)
		return
	}
	output(w, 200, json.RawMessage(raw))
}

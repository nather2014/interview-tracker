package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strconv"
	"time"
)

// IDs expire at midnight UTC; only a daily hash is stored, never an IP address.
func (a *app) visit(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Sec-Fetch-Site") == "cross-site" {
		fail(w, 403, "Same-site requests only")
		return
	}
	now := time.Now().UTC()
	day := now.Format("2006-01-02")
	value := ""
	if c, err := r.Cookie("interview_visitor"); err == nil {
		value = c.Value
	}
	if decoded, err := hex.DecodeString(value); err != nil || len(decoded) != 16 {
		var random [16]byte
		if _, err := rand.Read(random[:]); err != nil {
			fail(w, 500, "Could not create visitor ID")
			return
		}
		value = hex.EncodeToString(random[:])
	}
	expires := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, time.UTC)
	id := sha256.Sum256([]byte(day + ":" + value))
	_, err := a.db.ExecContext(r.Context(), `WITH cleanup AS (DELETE FROM visitor_presence WHERE visit_day < $1::date),
 inserted AS (
 INSERT INTO visitor_presence (visit_day, visitor_id, last_seen) VALUES ($1::date,$2,CURRENT_TIMESTAMP)
 ON CONFLICT (visit_day,visitor_id) DO NOTHING RETURNING visit_day
 ), refreshed AS (
 UPDATE visitor_presence SET last_seen=CURRENT_TIMESTAMP WHERE visit_day=$1::date AND visitor_id=$2
 )
 INSERT INTO visitor_daily (visit_day, visitors) SELECT visit_day, 1 FROM inserted
 ON CONFLICT (visit_day) DO UPDATE SET visitors=visitor_daily.visitors+1`, day, hex.EncodeToString(id[:]))
	if err != nil {
		a.internal(w, err)
		return
	}
	http.SetCookie(w, &http.Cookie{Name: "interview_visitor", Value: value, Path: "/api/visits", HttpOnly: true, Secure: true, SameSite: http.SameSiteStrictMode, Expires: expires, MaxAge: int(expires.Sub(now).Seconds())})
	w.WriteHeader(http.StatusNoContent)
}

func (a *app) visitors(w http.ResponseWriter, r *http.Request) {
	if !a.authorize(w, r) {
		return
	}
	page := 0
	if raw := r.URL.Query().Get("historyPage"); raw != "" {
		var err error
		page, err = strconv.Atoi(raw)
		if err != nil || page < 0 || page > 12000 {
			fail(w, 400, "Invalid history page")
			return
		}
	}
	var online, today int64
	err := a.db.QueryRowContext(r.Context(), `SELECT count(*) FILTER (WHERE last_seen > CURRENT_TIMESTAMP - INTERVAL '5 minutes'), COALESCE((SELECT visitors FROM visitor_daily WHERE visit_day = (CURRENT_TIMESTAMP AT TIME ZONE 'UTC')::date), 0) FROM visitor_presence`).Scan(&online, &today)
	if err != nil {
		a.internal(w, err)
		return
	}
	type dailyVisitors struct {
		Day      string `json:"day"`
		Visitors int64  `json:"visitors"`
	}
	history := []dailyVisitors{}
	rows, err := a.db.QueryContext(r.Context(), `SELECT days.day::date::text, COALESCE(d.visitors, 0)
 FROM generate_series((SELECT min(visit_day)::timestamp FROM visitor_daily),
 (CURRENT_TIMESTAMP AT TIME ZONE 'UTC')::date::timestamp, INTERVAL '1 day') AS days(day)
 LEFT JOIN visitor_daily d ON d.visit_day=days.day::date
 ORDER BY days.day DESC LIMIT 31 OFFSET $1`, page*30)
	if err != nil {
		a.internal(w, err)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var day dailyVisitors
		if err := rows.Scan(&day.Day, &day.Visitors); err != nil {
			a.internal(w, err)
			return
		}
		history = append(history, day)
	}
	if err := rows.Err(); err != nil {
		a.internal(w, err)
		return
	}
	hasMore := len(history) > 30
	if hasMore {
		history = history[:30]
	}
	output(w, 200, map[string]any{"online": online, "today": today, "dayTimezone": "UTC", "history": history, "historyPage": page, "hasMoreHistory": hasMore})
}

package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
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
	_, err := a.db.ExecContext(r.Context(), `WITH cleanup AS (DELETE FROM visitor_presence WHERE visit_day < $1::date)
 INSERT INTO visitor_presence (visit_day, visitor_id, last_seen) VALUES ($1::date,$2,CURRENT_TIMESTAMP)
 ON CONFLICT (visit_day,visitor_id) DO UPDATE SET last_seen=CURRENT_TIMESTAMP`, day, hex.EncodeToString(id[:]))
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
	var online, today int64
	err := a.db.QueryRowContext(r.Context(), `SELECT count(*) FILTER (WHERE last_seen > CURRENT_TIMESTAMP - INTERVAL '5 minutes'), count(*) FILTER (WHERE visit_day = (CURRENT_TIMESTAMP AT TIME ZONE 'UTC')::date) FROM visitor_presence`).Scan(&online, &today)
	if err != nil {
		a.internal(w, err)
		return
	}
	output(w, 200, map[string]any{"online": online, "today": today, "dayTimezone": "UTC"})
}

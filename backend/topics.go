package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"
)

type topicSuggestion struct {
	Topic       string `json:"topic"`
	NeedsReview bool   `json:"needsReview"`
	Method      string `json:"method"`
}

// Only the Go API reaches the single-worker model service on loopback.
func (a *app) suggestTopics(w http.ResponseWriter, r *http.Request) {
	if a.topicURL == "" {
		fail(w, 503, "Automatic topics are unavailable. Enter topics manually.")
		return
	}
	var body struct {
		Questions []string `json:"questions"`
	}
	d := json.NewDecoder(http.MaxBytesReader(w, r.Body, 110000))
	d.DisallowUnknownFields()
	if d.Decode(&body) != nil || d.Decode(new(any)) != io.EOF || len(body.Questions) < 1 || len(body.Questions) > 50 {
		fail(w, 400, "Provide between 1 and 50 questions")
		return
	}
	for _, question := range body.Questions {
		if !text(question, 2000, true) {
			fail(w, 400, "Invalid question")
			return
		}
	}
	if !a.topicBusy.CompareAndSwap(false, true) {
		fail(w, 429, "Topic suggestions are busy. Try again shortly.")
		return
	}
	defer a.topicBusy.Store(false)
	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()
	rows, err := a.db.QueryContext(ctx, "SELECT question,topic FROM question WHERE topic_source='manual' ORDER BY id DESC LIMIT 100")
	if err != nil {
		a.internal(w, err)
		return
	}
	examples := []question{}
	for rows.Next() {
		var example question
		if err = rows.Scan(&example.Question, &example.Topic); err != nil {
			break
		}
		examples = append(examples, example)
	}
	if err == nil {
		err = rows.Err()
	}
	rows.Close()
	if err != nil {
		a.internal(w, err)
		return
	}
	payload, _ := json.Marshal(map[string]any{"questions": body.Questions, "examples": examples})
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, a.topicURL+"/suggest", bytes.NewReader(payload))
	if err != nil {
		fail(w, 503, "Automatic topics are unavailable. Enter topics manually.")
		return
	}
	request.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 8 * time.Second}
	response, err := client.Do(request)
	if err != nil {
		fail(w, 503, "Automatic topics are unavailable. Enter topics manually.")
		return
	}
	defer response.Body.Close()
	var result struct {
		Suggestions []topicSuggestion `json:"suggestions"`
	}
	if response.StatusCode != 200 || json.NewDecoder(io.LimitReader(response.Body, 20000)).Decode(&result) != nil || len(result.Suggestions) != len(body.Questions) {
		fail(w, 503, "Automatic topics are unavailable. Enter topics manually.")
		return
	}
	for _, s := range result.Suggestions {
		if !text(s.Topic, 80, true) {
			fail(w, 503, "Invalid topic suggestion")
			return
		}
	}
	output(w, 200, result)
}

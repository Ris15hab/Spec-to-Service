package main

import (
	"encoding/json"
	"net/http"
	"sync"
)

type logStore struct {
	mu     sync.Mutex
	counts map[string]int
}

func newLogStore() *logStore {
	return &logStore{
		counts: map[string]int{
			"info":  0,
			"warn":  0,
			"error": 0,
		},
	}
}

func (s *logStore) increment(level string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.counts[level]++
}

func (s *logStore) getCounts() map[string]int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return map[string]int{
		"info":  s.counts["info"],
		"warn":  s.counts["warn"],
		"error": s.counts["error"],
	}
}

type logRequest struct {
	Level   string `json:"level"`
	Message string `json:"message"`
}

type statsResponse struct {
	Counts map[string]int `json:"counts"`
}

func registerRoutes(mux *http.ServeMux) {
	store := newLogStore()

	mux.HandleFunc("POST /logs", func(w http.ResponseWriter, r *http.Request) {
		var req logRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "malformed JSON", http.StatusBadRequest)
			return
		}

		if req.Message == "" {
			http.Error(w, "message cannot be empty", http.StatusBadRequest)
			return
		}

		switch req.Level {
		case "info":
			store.increment("info")
			logLinesInfoTotal.Inc()
		case "warn":
			store.increment("warn")
			logLinesWarnTotal.Inc()
		case "error":
			store.increment("error")
			logLinesErrorTotal.Inc()
		default:
			http.Error(w, "invalid level", http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusAccepted)
	})

	mux.HandleFunc("GET /logs/stats", func(w http.ResponseWriter, r *http.Request) {
		counts := store.getCounts()
		resp := statsResponse{Counts: counts}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})
}

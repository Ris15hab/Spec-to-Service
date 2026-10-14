package main

import (
	"encoding/json"
	"net/http"
	"sync"
)

type logStore struct {
	mu       sync.Mutex
	info     int
	warn     int
	errCount int
}

func (s *logStore) increment(level string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	switch level {
	case "info":
		s.info++
		logLinesInfoTotal.Inc()
	case "warn":
		s.warn++
		logLinesWarnTotal.Inc()
	case "error":
		s.errCount++
		logLinesErrorTotal.Inc()
	}
}

func (s *logStore) stats() (info, warn, err int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.info, s.warn, s.errCount
}

type logRequest struct {
	Level   string `json:"level"`
	Message string `json:"message"`
}

type statsResponse struct {
	Counts map[string]int `json:"counts"`
}

func registerRoutes(mux *http.ServeMux) {
	store := &logStore{}

	mux.HandleFunc("POST /logs", func(w http.ResponseWriter, r *http.Request) {
		var req logRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "malformed JSON", http.StatusBadRequest)
			return
		}

		if req.Message == "" {
			http.Error(w, "message must not be empty", http.StatusBadRequest)
			return
		}

		switch req.Level {
		case "info", "warn", "error":
			store.increment(req.Level)
			w.WriteHeader(http.StatusAccepted)
		default:
			http.Error(w, "invalid level", http.StatusBadRequest)
		}
	})

	mux.HandleFunc("GET /logs/stats", func(w http.ResponseWriter, r *http.Request) {
		info, warn, err := store.stats()
		resp := statsResponse{
			Counts: map[string]int{
				"info":  info,
				"warn":  warn,
				"error": err,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})
}

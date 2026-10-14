package main

import (
	"encoding/json"
	"net/http"
)

type logRequest struct {
	Level   string `json:"level"`
	Message string `json:"message"`
}

type statsResponse struct {
	Counts map[string]int64 `json:"counts"`
}

var store = NewStore()

func registerRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /logs", handlePostLogs)
	mux.HandleFunc("GET /logs/stats", handleGetStats)
}

func handlePostLogs(w http.ResponseWriter, r *http.Request) {
	var req logRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "malformed JSON", http.StatusBadRequest)
		return
	}

	// Validate level
	switch req.Level {
	case "info", "warn", "error":
	default:
		http.Error(w, "bad level", http.StatusBadRequest)
		return
	}

	// Validate message
	if req.Message == "" {
		http.Error(w, "empty message", http.StatusBadRequest)
		return
	}

	// Increment store and Prometheus counter
	store.Increment(req.Level)
	switch req.Level {
	case "info":
		logLinesInfoTotal.Inc()
	case "warn":
		logLinesWarnTotal.Inc()
	case "error":
		logLinesErrorTotal.Inc()
	}

	w.WriteHeader(http.StatusAccepted)
}

func handleGetStats(w http.ResponseWriter, r *http.Request) {
	counts := store.Counts()
	resp := statsResponse{Counts: counts}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

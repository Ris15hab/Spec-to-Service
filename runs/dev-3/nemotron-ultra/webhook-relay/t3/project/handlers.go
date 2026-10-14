package main

import (
	"encoding/json"
	"net/http"
	"net/url"
)

func registerRoutes(mux *http.ServeMux) {
	store := NewStore()

	mux.HandleFunc("POST /webhooks", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			URL     string          `json:"url"`
			Payload json.RawMessage `json:"payload"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "malformed JSON", http.StatusBadRequest)
			return
		}

		if req.URL == "" {
			http.Error(w, "missing url", http.StatusBadRequest)
			return
		}

		parsedURL, err := url.ParseRequestURI(req.URL)
		if err != nil || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") || parsedURL.Host == "" {
			http.Error(w, "invalid url", http.StatusBadRequest)
			return
		}

		if req.Payload == nil {
			req.Payload = json.RawMessage("{}")
		}

		job := &Job{
			URL:     req.URL,
			Payload: req.Payload,
		}
		store.Enqueue(job)

		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(map[string]string{"status": "queued"})
	})

	mux.HandleFunc("GET /webhooks/stats", func(w http.ResponseWriter, r *http.Request) {
		delivered, failed, pending := store.Stats()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]int64{
			"delivered": delivered,
			"failed":    failed,
			"pending":   pending,
		})
	})

	// Store shutdown on server close would be ideal but not required for tests
	_ = store
}

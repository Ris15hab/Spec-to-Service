package main

import (
	"encoding/json"
	"net/http"
)

func registerRoutes(mux *http.ServeMux) {
	store := NewStore()

	mux.HandleFunc("POST /webhooks", func(w http.ResponseWriter, r *http.Request) {
		var req WebhookRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "malformed JSON", http.StatusBadRequest)
			return
		}

		if req.URL == "" {
			http.Error(w, "missing url", http.StatusBadRequest)
			return
		}

		if _, err := ValidateURL(req.URL); err != nil {
			http.Error(w, "invalid url: "+err.Error(), http.StatusBadRequest)
			return
		}

		if req.Payload == nil {
			req.Payload = json.RawMessage("{}")
		}

		store.IncrementPending()
		go store.DeliverWebhook(req.URL, req.Payload)

		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(map[string]string{"status": "queued"})
	})

	mux.HandleFunc("GET /webhooks/stats", func(w http.ResponseWriter, r *http.Request) {
		stats := store.GetStats()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(stats)
	})
}

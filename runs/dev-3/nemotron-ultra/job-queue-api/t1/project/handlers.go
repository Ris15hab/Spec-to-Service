package main

import (
	"encoding/json"
	"net/http"
)

func registerRoutes(mux *http.ServeMux) {
	store := NewStore()

	mux.HandleFunc("POST /jobs", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID      string `json:"id"`
			Payload any    `json:"payload"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		if req.ID == "" {
			http.Error(w, "id is required", http.StatusBadRequest)
			return
		}

		_, ok := store.Enqueue(Job{ID: req.ID, Payload: req.Payload})
		if !ok {
			http.Error(w, "job id already exists", http.StatusConflict)
			return
		}

		jobsEnqueuedTotal.Inc()
		queueDepth.Set(float64(store.QueueDepth()))

		w.WriteHeader(http.StatusCreated)
	})

	mux.HandleFunc("POST /jobs/dequeue", func(w http.ResponseWriter, r *http.Request) {
		job, ok := store.Dequeue()
		if !ok {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		queueDepth.Set(float64(store.QueueDepth()))

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(job)
	})

	mux.HandleFunc("POST /jobs/{id}/ack", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		status, ok := store.Ack(id)
		if !ok {
			http.Error(w, http.StatusText(status), status)
			return
		}

		jobsAckedTotal.Inc()
		w.WriteHeader(http.StatusOK)
	})
}

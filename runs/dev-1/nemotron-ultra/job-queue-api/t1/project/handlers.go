package main

import (
	"encoding/json"
	"net/http"
)

type enqueueRequest struct {
	ID      string `json:"id"`
	Payload any    `json:"payload"`
}

type jobResponse struct {
	ID      string `json:"id"`
	Payload any    `json:"payload"`
}

var store = NewStore()

func registerRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /jobs", handleEnqueue)
	mux.HandleFunc("POST /jobs/dequeue", handleDequeue)
	mux.HandleFunc("POST /jobs/{id}/ack", handleAck)
}

func handleEnqueue(w http.ResponseWriter, r *http.Request) {
	handleEnqueueWithStore(store, w, r)
}

func handleDequeue(w http.ResponseWriter, r *http.Request) {
	handleDequeueWithStore(store, w, r)
}

func handleAck(w http.ResponseWriter, r *http.Request) {
	handleAckWithStore(store, w, r)
}

func handleEnqueueWithStore(s *Store, w http.ResponseWriter, r *http.Request) {
	var req enqueueRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	if req.ID == "" {
		http.Error(w, "id is required", http.StatusBadRequest)
		return
	}

	success, err := s.Enqueue(req.ID, req.Payload)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if !success {
		http.Error(w, "job id already exists", http.StatusConflict)
		return
	}

	jobsEnqueuedTotal.Inc()
	queueDepth.Set(float64(s.QueuedCount()))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(jobResponse{ID: req.ID, Payload: req.Payload})
}

func handleDequeueWithStore(s *Store, w http.ResponseWriter, r *http.Request) {
	job := s.Dequeue()
	if job == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	queueDepth.Set(float64(s.QueuedCount()))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(jobResponse{ID: job.ID, Payload: job.Payload})
}

func handleAckWithStore(s *Store, w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "id is required", http.StatusBadRequest)
		return
	}

	found, wasInFlight := s.Ack(id)
	if !found {
		http.Error(w, "job not found", http.StatusNotFound)
		return
	}

	if !wasInFlight {
		http.Error(w, "job not in-flight", http.StatusConflict)
		return
	}

	jobsAckedTotal.Inc()
	queueDepth.Set(float64(s.QueuedCount()))

	w.WriteHeader(http.StatusOK)
}

// NewHandler returns an http.Handler with its own isolated Store for testing.
func NewHandler(s *Store) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /jobs", func(w http.ResponseWriter, r *http.Request) {
		handleEnqueueWithStore(s, w, r)
	})
	mux.HandleFunc("POST /jobs/dequeue", func(w http.ResponseWriter, r *http.Request) {
		handleDequeueWithStore(s, w, r)
	})
	mux.HandleFunc("POST /jobs/{id}/ack", func(w http.ResponseWriter, r *http.Request) {
		handleAckWithStore(s, w, r)
	})
	return mux
}

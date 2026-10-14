package main

import (
	"encoding/json"
	"net/http"
	"sync"
)

type jobStatus int

const (
	statusQueued jobStatus = iota
	statusInFlight
	statusDone
)

type job struct {
	ID      string          `json:"id"`
	Payload json.RawMessage `json:"payload"`
	status  jobStatus
}

type store struct {
	mu    sync.Mutex
	jobs  map[string]*job
	queue []string
}

func newStore() *store {
	return &store{
		jobs:  make(map[string]*job),
		queue: make([]string, 0),
	}
}

func (s *store) enqueue(id string, payload json.RawMessage) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.jobs[id]; exists {
		return http.StatusConflict, nil
	}

	j := &job{
		ID:      id,
		Payload: payload,
		status:  statusQueued,
	}
	s.jobs[id] = j
	s.queue = append(s.queue, id)

	queueDepth.Set(float64(len(s.queue)))
	jobsEnqueuedTotal.Inc()

	return http.StatusCreated, nil
}

func (s *store) dequeue() (int, *job, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.queue) == 0 {
		return http.StatusNoContent, nil, nil
	}

	id := s.queue[0]
	s.queue = s.queue[1:]

	j := s.jobs[id]
	j.status = statusInFlight

	queueDepth.Set(float64(len(s.queue)))

	return http.StatusOK, j, nil
}

func (s *store) ack(id string) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	j, exists := s.jobs[id]
	if !exists {
		return http.StatusNotFound
	}

	if j.status != statusInFlight {
		return http.StatusConflict
	}

	j.status = statusDone
	jobsAckedTotal.Inc()

	return http.StatusOK
}

func registerRoutes(mux *http.ServeMux) {
	store := newStore()

	mux.HandleFunc("POST /jobs", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID      string          `json:"id"`
			Payload json.RawMessage `json:"payload"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}

		if req.ID == "" {
			http.Error(w, "id is required", http.StatusBadRequest)
			return
		}

		status, err := store.enqueue(req.ID, req.Payload)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(status)
	})

	mux.HandleFunc("POST /jobs/dequeue", func(w http.ResponseWriter, r *http.Request) {
		status, j, err := store.dequeue()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(status)
		if j != nil {
			json.NewEncoder(w).Encode(j)
		}
	})

	mux.HandleFunc("POST /jobs/{id}/ack", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if id == "" {
			http.Error(w, "id is required", http.StatusBadRequest)
			return
		}

		status := store.ack(id)
		w.WriteHeader(status)
	})
}

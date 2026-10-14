package main

import (
	"encoding/json"
	"net/http"
	"sync"
)

type job struct {
	ID      string          `json:"id"`
	Payload json.RawMessage `json:"payload"`
}

type store struct {
	mu       sync.Mutex
	queue    []job
	inFlight map[string]job
	exists   map[string]bool
}

func newStore() *store {
	return &store{
		queue:    make([]job, 0),
		inFlight: make(map[string]job),
		exists:   make(map[string]bool),
	}
}

func (s *store) enqueue(j job) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.exists[j.ID] {
		return http.StatusConflict, nil
	}

	s.queue = append(s.queue, j)
	s.exists[j.ID] = true
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

	j := s.queue[0]
	s.queue = s.queue[1:]
	s.inFlight[j.ID] = j
	queueDepth.Set(float64(len(s.queue)))
	return http.StatusOK, &j, nil
}

func (s *store) ack(id string) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.exists[id] {
		return http.StatusNotFound
	}

	if _, ok := s.inFlight[id]; !ok {
		return http.StatusConflict
	}

	delete(s.inFlight, id)
	delete(s.exists, id)
	jobsAckedTotal.Inc()
	return http.StatusOK
}

func registerRoutes(mux *http.ServeMux) {
	s := newStore()

	mux.HandleFunc("POST /jobs", func(w http.ResponseWriter, r *http.Request) {
		var j job
		if err := json.NewDecoder(r.Body).Decode(&j); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		if j.ID == "" {
			http.Error(w, "id required", http.StatusBadRequest)
			return
		}

		status, err := s.enqueue(j)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(status)
	})

	mux.HandleFunc("POST /jobs/dequeue", func(w http.ResponseWriter, r *http.Request) {
		status, j, err := s.dequeue()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if status == http.StatusNoContent {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(j)
	})

	mux.HandleFunc("POST /jobs/{id}/ack", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		status := s.ack(id)
		w.WriteHeader(status)
	})
}

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
	return http.StatusCreated, nil
}

func (s *store) dequeue() (job, int, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.queue) == 0 {
		return job{}, http.StatusNoContent, false
	}

	j := s.queue[0]
	s.queue = s.queue[1:]
	s.inFlight[j.ID] = j
	delete(s.exists, j.ID)
	queueDepth.Set(float64(len(s.queue)))
	return j, http.StatusOK, true
}

func (s *store) ack(id string) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.inFlight[id]; ok {
		delete(s.inFlight, id)
		jobsAckedTotal.Inc()
		return http.StatusOK
	}

	if _, ok := s.exists[id]; ok {
		return http.StatusConflict
	}

	return http.StatusNotFound
}

func registerRoutes(mux *http.ServeMux) {
	st := newStore()

	mux.HandleFunc("POST /jobs", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID      string          `json:"id"`
			Payload json.RawMessage `json:"payload"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		if req.ID == "" {
			http.Error(w, "id is required", http.StatusBadRequest)
			return
		}

		j := job{ID: req.ID, Payload: req.Payload}
		status, err := st.enqueue(j)
		if err != nil {
			http.Error(w, err.Error(), status)
			return
		}
		if status == http.StatusConflict {
			http.Error(w, "job id already exists", http.StatusConflict)
			return
		}

		jobsEnqueuedTotal.Inc()
		w.WriteHeader(http.StatusCreated)
	})

	mux.HandleFunc("POST /jobs/dequeue", func(w http.ResponseWriter, r *http.Request) {
		j, status, ok := st.dequeue()
		if !ok {
			w.WriteHeader(status)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		json.NewEncoder(w).Encode(j)
	})

	mux.HandleFunc("POST /jobs/{id}/ack", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		status := st.ack(id)
		switch status {
		case http.StatusOK:
			w.WriteHeader(http.StatusOK)
		case http.StatusConflict:
			http.Error(w, "job not in-flight", http.StatusConflict)
		case http.StatusNotFound:
			http.Error(w, "job not found", http.StatusNotFound)
		}
	})
}

package main

import (
	"encoding/json"
	"net/http"
	"sync"
)

type Job struct {
	ID     string `json:"id"`
	GPUs   int    `json:"gpus"`
	Status string `json:"status"`
}

type Store struct {
	mu   sync.Mutex
	jobs map[string]*Job
}

func NewStore() *Store {
	return &Store{
		jobs: make(map[string]*Job),
	}
}

func (s *Store) AddJob(job *Job) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.jobs[job.ID] = job
	s.updateMetrics()
}

func (s *Store) GetJobs() []*Job {
	s.mu.Lock()
	defer s.mu.Unlock()
	jobs := make([]*Job, 0, len(s.jobs))
	for _, job := range s.jobs {
		jobs = append(jobs, job)
	}
	return jobs
}

func (s *Store) updateMetrics() {
	queued := 0
	runningGPUs := 0
	for _, job := range s.jobs {
		if job.Status == "queued" {
			queued++
		}
		if job.Status == "running" {
			runningGPUs += job.GPUs
		}
	}
	gpuJobQueueLength.Set(float64(queued))
	// Simulate utilization: running GPUs as percentage of a hypothetical 100 GPU cluster
	gpuUtilizationPercent.Set(float64(runningGPUs))
}

func registerRoutes(mux *http.ServeMux) {
	store := NewStore()

	mux.HandleFunc("POST /jobs", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID   string `json:"id"`
			GPUs int    `json:"gpus"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}
		if req.ID == "" {
			http.Error(w, "missing id", http.StatusBadRequest)
			return
		}
		if req.GPUs <= 0 {
			http.Error(w, "gpus must be positive", http.StatusBadRequest)
			return
		}

		job := &Job{
			ID:     req.ID,
			GPUs:   req.GPUs,
			Status: "queued",
		}
		store.AddJob(job)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(job)
	})

	mux.HandleFunc("GET /jobs", func(w http.ResponseWriter, r *http.Request) {
		jobs := store.GetJobs()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(jobs)
	})
}

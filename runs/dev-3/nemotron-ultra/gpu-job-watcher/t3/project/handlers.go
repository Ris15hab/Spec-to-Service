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

type JobStore struct {
	mu   sync.Mutex
	jobs map[string]*Job
}

func NewJobStore() *JobStore {
	return &JobStore{
		jobs: make(map[string]*Job),
	}
}

func (s *JobStore) Add(job *Job) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.jobs[job.ID] = job
	updateMetrics(s)
}

func (s *JobStore) List() []*Job {
	s.mu.Lock()
	defer s.mu.Unlock()
	jobs := make([]*Job, 0, len(s.jobs))
	for _, job := range s.jobs {
		jobs = append(jobs, job)
	}
	updateMetrics(s)
	return jobs
}

func updateMetrics(s *JobStore) {
	gpuJobQueueLength.Set(float64(len(s.jobs)))

	totalGPUs := 0
	for _, job := range s.jobs {
		totalGPUs += job.GPUs
	}
	if len(s.jobs) > 0 {
		gpuUtilizationPercent.Set(float64(totalGPUs) / float64(len(s.jobs)) * 100)
	} else {
		gpuUtilizationPercent.Set(0)
	}
}

func registerRoutes(mux *http.ServeMux) {
	store := NewJobStore()

	mux.HandleFunc("POST /jobs", func(w http.ResponseWriter, r *http.Request) {
		var job Job
		if err := json.NewDecoder(r.Body).Decode(&job); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}
		if job.ID == "" {
			http.Error(w, "missing id", http.StatusBadRequest)
			return
		}
		if job.GPUs <= 0 {
			http.Error(w, "gpus must be positive", http.StatusBadRequest)
			return
		}
		job.Status = "queued"
		store.Add(&job)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(job)
	})

	mux.HandleFunc("GET /jobs", func(w http.ResponseWriter, r *http.Request) {
		jobs := store.List()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(jobs)
	})
}

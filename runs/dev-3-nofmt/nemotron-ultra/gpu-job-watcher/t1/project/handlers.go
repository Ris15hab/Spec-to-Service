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

func (s *JobStore) AddJob(job *Job) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.jobs[job.ID] = job
	updateQueueLength(len(s.jobs))
}

func (s *JobStore) GetJobs() []*Job {
	s.mu.Lock()
	defer s.mu.Unlock()
	jobs := make([]*Job, 0, len(s.jobs))
	for _, job := range s.jobs {
		jobs = append(jobs, job)
	}
	return jobs
}

func (s *JobStore) QueueLength() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.jobs)
}

func updateQueueLength(length int) {
	gpuJobQueueLength.Set(float64(length))
}

func updateUtilization() {
	// Simple utilization calculation: each job uses its GPUs
	// For now, we'll set a placeholder value
	gpuUtilizationPercent.Set(0.0)
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
		store.AddJob(&job)
		updateUtilization()

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

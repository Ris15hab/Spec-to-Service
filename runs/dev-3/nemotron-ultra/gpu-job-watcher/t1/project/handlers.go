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

func (s *Store) QueueLength() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	count := 0
	for _, job := range s.jobs {
		if job.Status == "queued" {
			count++
		}
	}
	return count
}

func (s *Store) UtilizationPercent() float64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.jobs) == 0 {
		return 0
	}
	totalGPUs := 0
	for _, job := range s.jobs {
		totalGPUs += job.GPUs
	}
	// Simple utilization: total GPUs requested / (max possible GPUs * job count)
	// For this implementation, we'll use a simple ratio
	return float64(totalGPUs) / float64(len(s.jobs)*8) * 100 // assuming 8 GPUs max per job
}

func registerRoutes(mux *http.ServeMux) {
	store := NewStore()

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

		gpuJobQueueLength.Set(float64(store.QueueLength()))
		gpuUtilizationPercent.Set(store.UtilizationPercent())

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(job)
	})

	mux.HandleFunc("GET /jobs", func(w http.ResponseWriter, r *http.Request) {
		jobs := store.GetJobs()

		gpuJobQueueLength.Set(float64(store.QueueLength()))
		gpuUtilizationPercent.Set(store.UtilizationPercent())

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(jobs)
	})
}

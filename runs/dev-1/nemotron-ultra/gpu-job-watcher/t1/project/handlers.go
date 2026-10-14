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
	mu    sync.Mutex
	jobs  map[string]*Job
	queue []*Job
}

var store = &JobStore{
	jobs:  make(map[string]*Job),
	queue: make([]*Job, 0),
}

const totalGPUs = 8

func registerRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /jobs", handlePostJobs)
	mux.HandleFunc("GET /jobs", handleGetJobs)
}

func handlePostJobs(w http.ResponseWriter, r *http.Request) {
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

	store.mu.Lock()
	defer store.mu.Unlock()

	if _, exists := store.jobs[job.ID]; exists {
		http.Error(w, "job already exists", http.StatusConflict)
		return
	}

	job.Status = "queued"
	store.jobs[job.ID] = &job
	store.queue = append(store.queue, &job)

	updateMetrics()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(job)
}

func handleGetJobs(w http.ResponseWriter, r *http.Request) {
	store.mu.Lock()
	defer store.mu.Unlock()

	jobs := make([]*Job, 0, len(store.jobs))
	for _, job := range store.jobs {
		jobs = append(jobs, job)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(jobs)
}

func updateMetrics() {
	queueLen := len(store.queue)
	gpuJobQueueLength.Set(float64(queueLen))

	totalRequested := 0
	for _, job := range store.queue {
		totalRequested += job.GPUs
	}

	var utilization float64
	if totalGPUs > 0 {
		utilization = float64(totalRequested) / float64(totalGPUs) * 100
	}
	gpuUtilizationPercent.Set(utilization)
}

// ResetStore clears all jobs and queue for testing purposes.
func ResetStore() {
	store.mu.Lock()
	defer store.mu.Unlock()
	store.jobs = make(map[string]*Job)
	store.queue = make([]*Job, 0)
	updateMetrics()
}

package main

import (
	"sync"
)

type JobStatus int

const (
	StatusQueued JobStatus = iota
	StatusInFlight
)

type Job struct {
	ID      string
	Payload any
	Status  JobStatus
}

type Store struct {
	mu    sync.Mutex
	jobs  map[string]*Job
	queue []*Job // FIFO order of queued jobs
}

func NewStore() *Store {
	return &Store{
		jobs:  make(map[string]*Job),
		queue: make([]*Job, 0),
	}
}

func (s *Store) Enqueue(id string, payload any) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.jobs[id]; exists {
		return false, nil // id already exists
	}

	job := &Job{
		ID:      id,
		Payload: payload,
		Status:  StatusQueued,
	}
	s.jobs[id] = job
	s.queue = append(s.queue, job)
	return true, nil
}

func (s *Store) Dequeue() *Job {
	s.mu.Lock()
	defer s.mu.Unlock()

	for len(s.queue) > 0 {
		job := s.queue[0]
		s.queue = s.queue[1:]
		if job.Status == StatusQueued {
			job.Status = StatusInFlight
			return job
		}
		// Job was already dequeued/acked, skip it
	}
	return nil
}

func (s *Store) Ack(id string) (bool, bool) { // (found, wasInFlight)
	s.mu.Lock()
	defer s.mu.Unlock()

	job, exists := s.jobs[id]
	if !exists {
		return false, false
	}

	if job.Status != StatusInFlight {
		return true, false
	}

	// Remove from jobs map
	delete(s.jobs, id)
	return true, true
}

func (s *Store) QueuedCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	count := 0
	for _, job := range s.jobs {
		if job.Status == StatusQueued {
			count++
		}
	}
	return count
}

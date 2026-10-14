package main

import (
	"sync"
)

type Job struct {
	ID      string
	Payload any
}

type Store struct {
	mu       sync.Mutex
	queue    []Job
	inFlight map[string]Job
	exists   map[string]bool
}

func NewStore() *Store {
	return &Store{
		queue:    make([]Job, 0),
		inFlight: make(map[string]Job),
		exists:   make(map[string]bool),
	}
}

func (s *Store) Enqueue(job Job) (bool, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.exists[job.ID] {
		return false, false // already exists
	}
	s.queue = append(s.queue, job)
	s.exists[job.ID] = true
	return true, true // success
}

func (s *Store) Dequeue() (Job, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.queue) == 0 {
		return Job{}, false
	}
	job := s.queue[0]
	s.queue = s.queue[1:]
	s.inFlight[job.ID] = job
	return job, true
}

func (s *Store) Ack(id string) (int, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	job, inFlight := s.inFlight[id]
	if !inFlight {
		if s.exists[id] {
			return 409, false // exists but not in-flight
		}
		return 404, false // unknown
	}
	delete(s.inFlight, id)
	delete(s.exists, id)
	_ = job
	return 200, true
}

func (s *Store) QueueDepth() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.queue)
}

package main

import (
	"sync"
)

type Store struct {
	mu     sync.Mutex
	counts map[string]int64
}

func NewStore() *Store {
	return &Store{
		counts: map[string]int64{
			"info":  0,
			"warn":  0,
			"error": 0,
		},
	}
}

func (s *Store) Increment(level string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.counts[level]++
}

func (s *Store) Counts() map[string]int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	// Return a copy to avoid race conditions
	return map[string]int64{
		"info":  s.counts["info"],
		"warn":  s.counts["warn"],
		"error": s.counts["error"],
	}
}

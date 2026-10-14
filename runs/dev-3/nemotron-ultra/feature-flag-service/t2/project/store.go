package main

import (
	"sync"
)

type Flag struct {
	Name           string `json:"name"`
	Enabled        bool   `json:"enabled"`
	RolloutPercent int    `json:"rollout_percent"`
}

type Store struct {
	mu    sync.RWMutex
	flags map[string]Flag
}

func NewStore() *Store {
	return &Store{
		flags: make(map[string]Flag),
	}
}

func (s *Store) Set(name string, flag Flag) (created bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, exists := s.flags[name]
	s.flags[name] = flag
	return !exists
}

func (s *Store) Get(name string) (Flag, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	flag, ok := s.flags[name]
	return flag, ok
}

func (s *Store) List() []Flag {
	s.mu.RLock()
	defer s.mu.RUnlock()
	flags := make([]Flag, 0, len(s.flags))
	for _, f := range s.flags {
		flags = append(flags, f)
	}
	return flags
}

func (s *Store) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.flags)
}

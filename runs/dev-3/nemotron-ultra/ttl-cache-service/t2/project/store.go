package main

import (
	"sync"
	"time"
)

type entry struct {
	value  string
	expiry time.Time
}

type Store struct {
	mu     sync.RWMutex
	data   map[string]entry
	stopCh chan struct{}
}

func NewStore() *Store {
	s := &Store{
		data:   make(map[string]entry),
		stopCh: make(chan struct{}),
	}
	go s.sweeper()
	return s
}

func (s *Store) Stop() {
	close(s.stopCh)
}

func (s *Store) Put(key, value string, ttl time.Duration) {
	expiry := time.Now().Add(ttl)
	s.mu.Lock()
	defer s.mu.Unlock()

	_, existed := s.data[key]
	s.data[key] = entry{value: value, expiry: expiry}

	if !existed {
		cacheEntries.Inc()
	}
}

func (s *Store) Get(key string) (string, bool) {
	s.mu.RLock()
	e, ok := s.data[key]
	s.mu.RUnlock()

	if !ok {
		return "", false
	}

	if time.Now().After(e.expiry) {
		s.Delete(key)
		return "", false
	}

	return e.value, true
}

func (s *Store) Delete(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, ok := s.data[key]
	if ok {
		delete(s.data, key)
		cacheEntries.Dec()
	}
	return ok
}

func (s *Store) sweeper() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.cleanExpired()
		case <-s.stopCh:
			return
		}
	}
}

func (s *Store) cleanExpired() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	for k, e := range s.data {
		if now.After(e.expiry) {
			delete(s.data, k)
			cacheEntries.Dec()
		}
	}
}

func (s *Store) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.data)
}

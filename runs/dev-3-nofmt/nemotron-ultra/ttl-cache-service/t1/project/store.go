package main

import (
	"sync"
	"time"
)

type entry struct {
	value      string
	expiryTime time.Time
}

type Store struct {
	mu      sync.RWMutex
	entries map[string]entry
	stopCh  chan struct{}
}

func NewStore() *Store {
	s := &Store{
		entries: make(map[string]entry),
		stopCh:  make(chan struct{}),
	}
	go s.sweeper()
	return s
}

func (s *Store) Stop() {
	close(s.stopCh)
}

func (s *Store) Put(key, value string, ttlSeconds int) {
	expiry := time.Now().Add(time.Duration(ttlSeconds) * time.Second)
	s.mu.Lock()
	defer s.mu.Unlock()

	_, existed := s.entries[key]
	s.entries[key] = entry{value: value, expiryTime: expiry}

	if !existed {
		cacheEntries.Inc()
	}
}

func (s *Store) Get(key string) (string, bool) {
	s.mu.RLock()
	e, ok := s.entries[key]
	s.mu.RUnlock()

	if !ok {
		return "", false
	}

	if time.Now().After(e.expiryTime) {
		s.mu.Lock()
		if e2, ok2 := s.entries[key]; ok2 && time.Now().After(e2.expiryTime) {
			delete(s.entries, key)
			cacheEntries.Dec()
		}
		s.mu.Unlock()
		return "", false
	}

	return e.value, true
}

func (s *Store) Delete(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.entries[key]; ok {
		delete(s.entries, key)
		cacheEntries.Dec()
		return true
	}
	return false
}

func (s *Store) sweeper() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.mu.Lock()
			now := time.Now()
			for k, e := range s.entries {
				if now.After(e.expiryTime) {
					delete(s.entries, k)
					cacheEntries.Dec()
				}
			}
			s.mu.Unlock()
		case <-s.stopCh:
			return
		}
	}
}

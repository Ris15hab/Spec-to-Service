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

func (s *Store) sweeper() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			s.mu.Lock()
			now := time.Now()
			for k, v := range s.data {
				if now.After(v.expiry) {
					delete(s.data, k)
					cacheEntries.Dec()
				}
			}
			s.mu.Unlock()
		case <-s.stopCh:
			return
		}
	}
}

func (s *Store) Close() {
	close(s.stopCh)
}

func (s *Store) Put(key, value string, ttl time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	existed := false
	if _, ok := s.data[key]; ok {
		existed = true
	}

	s.data[key] = entry{
		value:  value,
		expiry: time.Now().Add(ttl),
	}

	if !existed {
		cacheEntries.Inc()
	}
}

func (s *Store) Get(key string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	e, ok := s.data[key]
	if !ok {
		return "", false
	}

	if time.Now().After(e.expiry) {
		delete(s.data, key)
		cacheEntries.Dec()
		return "", false
	}

	return e.value, true
}

func (s *Store) Delete(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.data[key]; !ok {
		return false
	}

	delete(s.data, key)
	cacheEntries.Dec()
	return true
}

func (s *Store) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.data)
}

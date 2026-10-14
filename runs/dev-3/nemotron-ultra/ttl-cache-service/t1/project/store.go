package main

import (
	"sync"
	"time"
)

type entry struct {
	value     string
	expiresAt time.Time
}

type Store struct {
	mu      sync.RWMutex
	data    map[string]entry
	stopCh  chan struct{}
	sweeper *time.Ticker
}

func NewStore() *Store {
	s := &Store{
		data:   make(map[string]entry),
		stopCh: make(chan struct{}),
	}
	s.sweeper = time.NewTicker(1 * time.Second)
	go s.runSweeper()
	return s
}

func (s *Store) runSweeper() {
	for {
		select {
		case <-s.sweeper.C:
			s.cleanupExpired()
		case <-s.stopCh:
			s.sweeper.Stop()
			return
		}
	}
}

func (s *Store) cleanupExpired() {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	count := 0
	for k, e := range s.data {
		if now.After(e.expiresAt) {
			delete(s.data, k)
			count++
		}
	}
	if count > 0 {
		cacheEntries.Set(float64(len(s.data)))
	}
}

func (s *Store) Put(key, value string, ttl time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[key] = entry{
		value:     value,
		expiresAt: time.Now().Add(ttl),
	}
	cacheEntries.Set(float64(len(s.data)))
}

func (s *Store) Get(key string) (string, bool) {
	s.mu.RLock()
	e, ok := s.data[key]
	s.mu.RUnlock()
	if !ok {
		return "", false
	}
	if time.Now().After(e.expiresAt) {
		s.mu.Lock()
		if e2, ok2 := s.data[key]; ok2 && time.Now().After(e2.expiresAt) {
			delete(s.data, key)
			cacheEntries.Set(float64(len(s.data)))
		}
		s.mu.Unlock()
		return "", false
	}
	return e.value, true
}

func (s *Store) Delete(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.data[key]; ok {
		delete(s.data, key)
		cacheEntries.Set(float64(len(s.data)))
		return true
	}
	return false
}

func (s *Store) Close() {
	close(s.stopCh)
}

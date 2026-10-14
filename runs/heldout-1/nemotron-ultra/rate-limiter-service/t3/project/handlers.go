package main

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

type bucket struct {
	tokens     float64
	lastRefill time.Time
}

type store struct {
	mu      sync.Mutex
	buckets map[string]*bucket
}

const (
	capacity   = 5.0
	refillRate = 1.0 // tokens per second
)

func newStore() *store {
	return &store{
		buckets: make(map[string]*bucket),
	}
}

func (s *store) check(key string) (allowed bool, remaining float64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	b, exists := s.buckets[key]
	now := time.Now()

	if !exists {
		b = &bucket{
			tokens:     capacity,
			lastRefill: now,
		}
		s.buckets[key] = b
		ratelimitBuckets.Inc()
	}

	elapsed := now.Sub(b.lastRefill).Seconds()
	b.tokens = min(capacity, b.tokens+elapsed*refillRate)
	b.lastRefill = now

	if b.tokens >= 1.0 {
		b.tokens -= 1.0
		ratelimitAllowedTotal.Inc()
		return true, b.tokens
	}

	ratelimitDeniedTotal.Inc()
	return false, 0
}

func (s *store) get(key string) (*bucket, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	b, exists := s.buckets[key]
	if !exists {
		return nil, false
	}

	now := time.Now()
	elapsed := now.Sub(b.lastRefill).Seconds()
	tokens := min(capacity, b.tokens+elapsed*refillRate)
	return &bucket{tokens: tokens, lastRefill: now}, true
}

func registerRoutes(mux *http.ServeMux) {
	s := newStore()

	mux.HandleFunc("POST /check", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Key string `json:"key"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "malformed JSON", http.StatusBadRequest)
			return
		}

		if req.Key == "" {
			http.Error(w, "key is required", http.StatusBadRequest)
			return
		}

		allowed, remaining := s.check(req.Key)

		resp := map[string]any{
			"allowed":   allowed,
			"remaining": remaining,
		}

		w.Header().Set("Content-Type", "application/json")
		if allowed {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusTooManyRequests)
		}
		json.NewEncoder(w).Encode(resp)
	})

	mux.HandleFunc("GET /buckets/{key}", func(w http.ResponseWriter, r *http.Request) {
		key := r.PathValue("key")
		if key == "" {
			http.Error(w, "key is required", http.StatusBadRequest)
			return
		}

		b, exists := s.get(key)
		if !exists {
			http.Error(w, "bucket not found", http.StatusNotFound)
			return
		}

		resp := map[string]any{
			"key":    key,
			"tokens": b.tokens,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	})
}

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

func newStore() *store {
	return &store{
		buckets: make(map[string]*bucket),
	}
}

func (s *store) check(key string) (allowed bool, remaining float64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	b, ok := s.buckets[key]
	now := time.Now()
	if !ok {
		b = &bucket{
			tokens:     5.0,
			lastRefill: now,
		}
		s.buckets[key] = b
		ratelimitBuckets.Set(float64(len(s.buckets)))
	}

	elapsed := now.Sub(b.lastRefill).Seconds()
	b.tokens = min(5.0, b.tokens+elapsed*1.0)
	b.lastRefill = now

	if b.tokens >= 1.0 {
		b.tokens -= 1.0
		ratelimitAllowedTotal.Inc()
		return true, b.tokens
	}

	ratelimitDeniedTotal.Inc()
	return false, 0.0
}

func (s *store) get(key string) (float64, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	b, ok := s.buckets[key]
	if !ok {
		return 0, false
	}

	now := time.Now()
	elapsed := now.Sub(b.lastRefill).Seconds()
	tokens := min(5.0, b.tokens+elapsed*1.0)
	return tokens, true
}

func registerRoutes(mux *http.ServeMux) {
	s := newStore()

	mux.HandleFunc("POST /check", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Key string `json:"key"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"malformed JSON"}`, http.StatusBadRequest)
			return
		}
		if req.Key == "" {
			http.Error(w, `{"error":"key is required"}`, http.StatusBadRequest)
			return
		}

		allowed, remaining := s.check(req.Key)
		if allowed {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{
				"allowed":   true,
				"remaining": remaining,
			})
		} else {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			json.NewEncoder(w).Encode(map[string]any{
				"allowed":   false,
				"remaining": 0.0,
			})
		}
	})

	mux.HandleFunc("GET /buckets/{key}", func(w http.ResponseWriter, r *http.Request) {
		key := r.PathValue("key")
		tokens, ok := s.get(key)
		if !ok {
			http.Error(w, `{"error":"bucket not found"}`, http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"key":    key,
			"tokens": tokens,
		})
	})
}

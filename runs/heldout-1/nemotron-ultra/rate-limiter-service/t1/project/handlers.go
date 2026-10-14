package main

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

type bucket struct {
	mu         sync.Mutex
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

func (s *store) getOrCreate(key string) *bucket {
	s.mu.Lock()
	defer s.mu.Unlock()

	b, ok := s.buckets[key]
	if !ok {
		b = &bucket{
			tokens:     5.0,
			lastRefill: time.Now(),
		}
		s.buckets[key] = b
		ratelimitBuckets.Set(float64(len(s.buckets)))
	}
	return b
}

func (s *store) get(key string) (*bucket, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	b, ok := s.buckets[key]
	return b, ok
}

func (b *bucket) refill(now time.Time) {
	elapsed := now.Sub(b.lastRefill).Seconds()
	b.tokens = min(5.0, b.tokens+elapsed)
	b.lastRefill = now
}

func (b *bucket) tryConsume() bool {
	if b.tokens >= 1.0 {
		b.tokens--
		return true
	}
	return false
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

		b := s.getOrCreate(req.Key)
		b.mu.Lock()
		b.refill(time.Now())
		allowed := b.tryConsume()
		remaining := int(b.tokens)
		b.mu.Unlock()

		if allowed {
			ratelimitAllowedTotal.Inc()
			json.NewEncoder(w).Encode(map[string]any{
				"allowed":   true,
				"remaining": remaining,
			})
		} else {
			ratelimitDeniedTotal.Inc()
			w.WriteHeader(http.StatusTooManyRequests)
			json.NewEncoder(w).Encode(map[string]any{
				"allowed":   false,
				"remaining": 0,
			})
		}
	})

	mux.HandleFunc("GET /buckets/{key}", func(w http.ResponseWriter, r *http.Request) {
		key := r.PathValue("key")
		b, ok := s.get(key)
		if !ok {
			http.NotFound(w, r)
			return
		}

		b.mu.Lock()
		b.refill(time.Now())
		tokens := b.tokens
		b.mu.Unlock()

		json.NewEncoder(w).Encode(map[string]any{
			"key":    key,
			"tokens": tokens,
		})
	})
}

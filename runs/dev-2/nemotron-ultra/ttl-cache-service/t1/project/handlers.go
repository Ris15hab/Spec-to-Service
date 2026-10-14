package main

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

type entry struct {
	value     string
	expiresAt time.Time
}

type Store struct {
	mu      sync.RWMutex
	data    map[string]*entry
	stopCh  chan struct{}
	sweeper *time.Ticker
}

func NewStore() *Store {
	s := &Store{
		data:   make(map[string]*entry),
		stopCh: make(chan struct{}),
	}
	s.sweeper = time.NewTicker(1 * time.Second)
	go s.sweepLoop()
	return s
}

func (s *Store) sweepLoop() {
	for {
		select {
		case <-s.sweeper.C:
			s.sweep()
		case <-s.stopCh:
			s.sweeper.Stop()
			return
		}
	}
}

func (s *Store) sweep() {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	for k, e := range s.data {
		if now.After(e.expiresAt) {
			delete(s.data, k)
		}
	}
	cacheEntries.Set(float64(len(s.data)))
}

func (s *Store) Put(key, value string, ttl time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[key] = &entry{
		value:     value,
		expiresAt: time.Now().Add(ttl),
	}
	cacheEntries.Set(float64(len(s.data)))
}

func (s *Store) Get(key string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.data[key]
	if !ok {
		return "", false
	}
	if time.Now().After(e.expiresAt) {
		delete(s.data, key)
		cacheEntries.Set(float64(len(s.data)))
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
		cacheEntries.Set(float64(len(s.data)))
	}
	return ok
}

func (s *Store) Close() {
	close(s.stopCh)
}

type putRequest struct {
	Value      string `json:"value"`
	TTLSeconds int    `json:"ttl_seconds"`
}

func registerRoutes(mux *http.ServeMux) {
	store := NewStore()

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	mux.HandleFunc("GET /metrics", func(w http.ResponseWriter, r *http.Request) {
		PromHandler.ServeHTTP(w, r)
	})

	mux.HandleFunc("PUT /cache/{key}", func(w http.ResponseWriter, r *http.Request) {
		key := r.PathValue("key")
		if key == "" {
			http.Error(w, "key required", http.StatusBadRequest)
			return
		}

		var req putRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "malformed JSON", http.StatusBadRequest)
			return
		}

		if req.Value == "" {
			http.Error(w, "value required", http.StatusBadRequest)
			return
		}
		if req.TTLSeconds <= 0 {
			http.Error(w, "ttl_seconds must be > 0", http.StatusBadRequest)
			return
		}

		store.Put(key, req.Value, time.Duration(req.TTLSeconds)*time.Second)
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"status": "created"})
	})

	mux.HandleFunc("GET /cache/{key}", func(w http.ResponseWriter, r *http.Request) {
		key := r.PathValue("key")
		if key == "" {
			http.Error(w, "key required", http.StatusBadRequest)
			return
		}

		value, ok := store.Get(key)
		if !ok {
			cacheMissesTotal.Inc()
			http.Error(w, "not found", http.StatusNotFound)
			return
		}

		cacheHitsTotal.Inc()
		json.NewEncoder(w).Encode(map[string]string{
			"key":   key,
			"value": value,
		})
	})

	mux.HandleFunc("DELETE /cache/{key}", func(w http.ResponseWriter, r *http.Request) {
		key := r.PathValue("key")
		if key == "" {
			http.Error(w, "key required", http.StatusBadRequest)
			return
		}

		ok := store.Delete(key)
		if !ok {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
}

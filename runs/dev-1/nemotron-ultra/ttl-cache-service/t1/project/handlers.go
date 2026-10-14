package main

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

type cacheEntry struct {
	value     string
	expiresAt time.Time
}

type cacheStore struct {
	mu   sync.RWMutex
	data map[string]cacheEntry
	done chan struct{}
	wg   sync.WaitGroup
}

func newCacheStore() *cacheStore {
	cs := &cacheStore{
		data: make(map[string]cacheEntry),
		done: make(chan struct{}),
	}
	cs.wg.Add(1)
	go cs.sweeper()
	return cs
}

func (cs *cacheStore) sweeper() {
	defer cs.wg.Done()
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-cs.done:
			return
		case <-ticker.C:
			cs.mu.Lock()
			now := time.Now()
			for k, v := range cs.data {
				if now.After(v.expiresAt) {
					delete(cs.data, k)
				}
			}
			cacheEntries.Set(float64(len(cs.data)))
			cs.mu.Unlock()
		}
	}
}

func (cs *cacheStore) close() {
	close(cs.done)
	cs.wg.Wait()
}

func (cs *cacheStore) put(key, value string, ttlSeconds int) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	cs.data[key] = cacheEntry{
		value:     value,
		expiresAt: time.Now().Add(time.Duration(ttlSeconds) * time.Second),
	}
	cacheEntries.Set(float64(len(cs.data)))
}

func (cs *cacheStore) get(key string) (string, bool) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	entry, ok := cs.data[key]
	if !ok {
		return "", false
	}
	if time.Now().After(entry.expiresAt) {
		delete(cs.data, key)
		cacheEntries.Set(float64(len(cs.data)))
		return "", false
	}
	return entry.value, true
}

func (cs *cacheStore) delete(key string) bool {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	_, ok := cs.data[key]
	if !ok {
		return false
	}
	delete(cs.data, key)
	cacheEntries.Set(float64(len(cs.data)))
	return true
}

var store = newCacheStore()

type putRequest struct {
	Value      string `json:"value"`
	TTLSeconds int    `json:"ttl_seconds"`
}

func registerRoutes(mux *http.ServeMux) {
	mux.HandleFunc("PUT /cache/{key}", handlePut)
	mux.HandleFunc("GET /cache/{key}", handleGet)
	mux.HandleFunc("DELETE /cache/{key}", handleDelete)
}

func handlePut(w http.ResponseWriter, r *http.Request) {
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

	store.put(key, req.Value, req.TTLSeconds)
	w.WriteHeader(http.StatusCreated)
}

func handleGet(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	if key == "" {
		http.Error(w, "key required", http.StatusBadRequest)
		return
	}

	value, ok := store.get(key)
	if !ok {
		cacheMissesTotal.Inc()
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	cacheHitsTotal.Inc()
	resp := map[string]string{"key": key, "value": value}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func handleDelete(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	if key == "" {
		http.Error(w, "key required", http.StatusBadRequest)
		return
	}

	if store.delete(key) {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	http.Error(w, "not found", http.StatusNotFound)
}

package main

import (
	"encoding/json"
	"errors"
	"hash/fnv"
	"net/http"
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

func (s *Store) Put(name string, flag Flag) (bool, error) {
	if flag.RolloutPercent < 0 || flag.RolloutPercent > 100 {
		return false, errors.New("rollout_percent out of range")
	}
	flag.Name = name
	s.mu.Lock()
	defer s.mu.Unlock()
	_, existed := s.flags[name]
	s.flags[name] = flag
	return !existed, nil
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

func fnv32a(s string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(s))
	return h.Sum32()
}

func registerRoutes(mux *http.ServeMux) {
	store := NewStore()

	mux.HandleFunc("PUT /flags/{name}", func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")
		if name == "" {
			http.Error(w, "name required", http.StatusBadRequest)
			return
		}

		var req struct {
			Enabled        bool `json:"enabled"`
			RolloutPercent int  `json:"rollout_percent"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "malformed JSON", http.StatusBadRequest)
			return
		}

		flag := Flag{
			Enabled:        req.Enabled,
			RolloutPercent: req.RolloutPercent,
		}

		created, err := store.Put(name, flag)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		flagsTotal.Set(float64(store.Count()))

		w.Header().Set("Content-Type", "application/json")
		if created {
			w.WriteHeader(http.StatusCreated)
		} else {
			w.WriteHeader(http.StatusOK)
		}
		json.NewEncoder(w).Encode(Flag{Name: name, Enabled: req.Enabled, RolloutPercent: req.RolloutPercent})
	})

	mux.HandleFunc("GET /flags/{name}", func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")
		flag, ok := store.Get(name)
		if !ok {
			http.Error(w, "flag not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(flag)
	})

	mux.HandleFunc("GET /flags", func(w http.ResponseWriter, r *http.Request) {
		flags := store.List()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(flags)
	})

	mux.HandleFunc("GET /evaluate", func(w http.ResponseWriter, r *http.Request) {
		flagName := r.URL.Query().Get("flag")
		user := r.URL.Query().Get("user")

		if flagName == "" || user == "" {
			http.Error(w, "flag and user query parameters required", http.StatusBadRequest)
			return
		}

		flag, ok := store.Get(flagName)
		if !ok {
			http.Error(w, "flag not found", http.StatusNotFound)
			return
		}

		var enabled bool
		if flag.Enabled {
			hash := fnv32a(user)
			enabled = (hash % 100) < uint32(flag.RolloutPercent)
		}

		flagEvaluationsTotal.Inc()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]bool{"enabled": enabled})
	})
}

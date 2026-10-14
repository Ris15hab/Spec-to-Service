package main

import (
	"encoding/json"
	"hash/fnv"
	"net/http"
	"sync"
)

type Flag struct {
	Name           string `json:"name"`
	Enabled        bool   `json:"enabled"`
	RolloutPercent int    `json:"rollout_percent"`
}

type FlagStore struct {
	mu    sync.RWMutex
	flags map[string]Flag
}

func NewFlagStore() *FlagStore {
	return &FlagStore{
		flags: make(map[string]Flag),
	}
}

func (s *FlagStore) Set(name string, flag Flag) (created bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, exists := s.flags[name]
	s.flags[name] = flag
	return !exists
}

func (s *FlagStore) Get(name string) (Flag, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	flag, ok := s.flags[name]
	return flag, ok
}

func (s *FlagStore) List() []Flag {
	s.mu.RLock()
	defer s.mu.RUnlock()
	flags := make([]Flag, 0, len(s.flags))
	for _, f := range s.flags {
		flags = append(flags, f)
	}
	return flags
}

func (s *FlagStore) Count() int {
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
	store := NewFlagStore()

	mux.HandleFunc("PUT /flags/{name}", func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")
		if name == "" {
			http.Error(w, "flag name required", http.StatusBadRequest)
			return
		}

		var input struct {
			Enabled        *bool `json:"enabled"`
			RolloutPercent int   `json:"rollout_percent"`
		}

		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			http.Error(w, "malformed JSON", http.StatusBadRequest)
			return
		}

		if input.Enabled == nil {
			http.Error(w, "enabled field required", http.StatusBadRequest)
			return
		}

		if input.RolloutPercent < 0 || input.RolloutPercent > 100 {
			http.Error(w, "rollout_percent must be between 0 and 100", http.StatusBadRequest)
			return
		}

		flag := Flag{
			Name:           name,
			Enabled:        *input.Enabled,
			RolloutPercent: input.RolloutPercent,
		}

		created := store.Set(name, flag)
		flagsTotal.Set(float64(store.Count()))

		w.Header().Set("Content-Type", "application/json")
		if created {
			w.WriteHeader(http.StatusCreated)
		} else {
			w.WriteHeader(http.StatusOK)
		}
		json.NewEncoder(w).Encode(flag)
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

package main

import (
	"encoding/json"
	"errors"
	"hash/fnv"
	"io"
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

func (s *FlagStore) Set(name string, flag Flag) (bool, error) {
	if flag.RolloutPercent < 0 || flag.RolloutPercent > 100 {
		return false, errors.New("rollout_percent must be between 0 and 100")
	}
	flag.Name = name
	s.mu.Lock()
	defer s.mu.Unlock()
	_, existed := s.flags[name]
	s.flags[name] = flag
	flagsTotal.Set(float64(len(s.flags)))
	return existed, nil
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

var store = NewFlagStore()

func registerRoutes(mux *http.ServeMux) {
	mux.HandleFunc("PUT /flags/{name}", handlePutFlag)
	mux.HandleFunc("GET /flags/{name}", handleGetFlag)
	mux.HandleFunc("GET /flags", handleListFlags)
	mux.HandleFunc("GET /evaluate", handleEvaluate)
}

func handlePutFlag(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		http.Error(w, "flag name required", http.StatusBadRequest)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "malformed JSON", http.StatusBadRequest)
		return
	}

	// Check for presence of "enabled" key
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		http.Error(w, "malformed JSON", http.StatusBadRequest)
		return
	}
	if _, ok := raw["enabled"]; !ok {
		http.Error(w, "enabled field required", http.StatusBadRequest)
		return
	}

	var input struct {
		Enabled        bool `json:"enabled"`
		RolloutPercent int  `json:"rollout_percent"`
	}
	if err := json.Unmarshal(body, &input); err != nil {
		http.Error(w, "malformed JSON", http.StatusBadRequest)
		return
	}

	flag := Flag{
		Name:           name,
		Enabled:        input.Enabled,
		RolloutPercent: input.RolloutPercent,
	}

	existed, err := store.Set(name, flag)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if existed {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusCreated)
	}
	json.NewEncoder(w).Encode(flag)
}

func handleGetFlag(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	flag, ok := store.Get(name)
	if !ok {
		http.Error(w, "flag not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(flag)
}

func handleListFlags(w http.ResponseWriter, r *http.Request) {
	flags := store.List()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(flags)
}

func handleEvaluate(w http.ResponseWriter, r *http.Request) {
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

	enabled := false
	if flag.Enabled {
		h := fnv.New32a()
		h.Write([]byte(user))
		hash := h.Sum32()
		if int(hash%100) < flag.RolloutPercent {
			enabled = true
		}
	}

	flagEvaluationsTotal.Inc()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"enabled": enabled})
}

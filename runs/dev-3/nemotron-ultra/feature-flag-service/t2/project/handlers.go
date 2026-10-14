package main

import (
	"encoding/json"
	"net/http"
)

func registerRoutes(mux *http.ServeMux) {
	store := NewStore()

	mux.HandleFunc("PUT /flags/{name}", func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")
		if name == "" {
			http.Error(w, "flag name required", http.StatusBadRequest)
			return
		}

		var req struct {
			Enabled        *bool `json:"enabled"`
			RolloutPercent int   `json:"rollout_percent"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "malformed JSON", http.StatusBadRequest)
			return
		}

		if req.Enabled == nil {
			http.Error(w, "enabled field required", http.StatusBadRequest)
			return
		}

		if req.RolloutPercent < 0 || req.RolloutPercent > 100 {
			http.Error(w, "rollout_percent must be between 0 and 100", http.StatusBadRequest)
			return
		}

		flag := Flag{
			Name:           name,
			Enabled:        *req.Enabled,
			RolloutPercent: req.RolloutPercent,
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

		enabled := false
		if flag.Enabled {
			hash := fnv32a(user)
			enabled = (hash % 100) < uint32(flag.RolloutPercent)
		}

		flagEvaluationsTotal.Inc()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]bool{"enabled": enabled})
	})
}

func fnv32a(s string) uint32 {
	const (
		offset32 = 2166136261
		prime32  = 16777619
	)
	hash := uint32(offset32)
	for i := 0; i < len(s); i++ {
		hash ^= uint32(s[i])
		hash *= prime32
	}
	return hash
}

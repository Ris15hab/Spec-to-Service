package main

import (
	"encoding/json"
	"net/http"
	"time"
)

type putRequest struct {
	Value      string `json:"value"`
	TTLSeconds int    `json:"ttl_seconds"`
}

type getResponse struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

func registerRoutes(mux *http.ServeMux) {
	store := NewStore()

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
		resp := getResponse{Key: key, Value: value}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	mux.HandleFunc("DELETE /cache/{key}", func(w http.ResponseWriter, r *http.Request) {
		key := r.PathValue("key")
		if key == "" {
			http.Error(w, "key required", http.StatusBadRequest)
			return
		}

		if store.Delete(key) {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		http.Error(w, "not found", http.StatusNotFound)
	})

	_ = store // store is used by handlers via closure
}

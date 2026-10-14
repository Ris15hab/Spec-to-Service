package main

import (
	"encoding/json"
	"net/http"
	"regexp"
)

var sha256Regex = regexp.MustCompile(`^[a-f0-9]{64}$`)

type createArtifactRequest struct {
	Name      string `json:"name"`
	Version   string `json:"version"`
	SHA256    string `json:"sha256"`
	SizeBytes int64  `json:"size_bytes"`
}

func registerRoutes(mux *http.ServeMux) {
	store := NewStore()

	mux.HandleFunc("POST /artifacts", func(w http.ResponseWriter, r *http.Request) {
		var req createArtifactRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "malformed JSON", http.StatusBadRequest)
			return
		}

		if req.Name == "" || req.Version == "" || req.SHA256 == "" {
			http.Error(w, "missing required field", http.StatusBadRequest)
			return
		}
		if !sha256Regex.MatchString(req.SHA256) {
			http.Error(w, "invalid sha256", http.StatusBadRequest)
			return
		}
		if req.SizeBytes < 0 {
			http.Error(w, "size_bytes must be >= 0", http.StatusBadRequest)
			return
		}

		artifact := Artifact{
			Name:      req.Name,
			Version:   req.Version,
			SHA256:    req.SHA256,
			SizeBytes: req.SizeBytes,
		}

		created, err := store.Create(artifact)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		if !created {
			http.Error(w, "artifact version already exists", http.StatusConflict)
			return
		}

		artifactsTotal.Set(float64(store.Count()))
		artifactUploadsTotal.Inc()

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(artifact)
	})

	mux.HandleFunc("GET /artifacts", func(w http.ResponseWriter, r *http.Request) {
		artifacts := store.ListAll()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(artifacts)
	})

	mux.HandleFunc("GET /artifacts/{name}", func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")
		versions, ok := store.ListVersions(name)
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(versions)
	})

	mux.HandleFunc("GET /artifacts/{name}/{version}", func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")
		version := r.PathValue("version")
		artifact, ok := store.Get(name, version)
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(artifact)
	})
}

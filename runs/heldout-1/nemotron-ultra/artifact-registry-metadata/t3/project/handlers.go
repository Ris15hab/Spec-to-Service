package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"sync"
)

type Artifact struct {
	Name      string `json:"name"`
	Version   string `json:"version"`
	SHA256    string `json:"sha256"`
	SizeBytes *int64 `json:"size_bytes"`
}

type Store struct {
	mu        sync.Mutex
	artifacts map[string]map[string]Artifact // name -> version -> Artifact
}

func NewStore() *Store {
	return &Store{
		artifacts: make(map[string]map[string]Artifact),
	}
}

func (s *Store) Create(a Artifact) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.artifacts[a.Name][a.Version]; exists {
		return errors.New("conflict")
	}

	if s.artifacts[a.Name] == nil {
		s.artifacts[a.Name] = make(map[string]Artifact)
	}
	s.artifacts[a.Name][a.Version] = a
	return nil
}

func (s *Store) Get(name, version string) (Artifact, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	versions, ok := s.artifacts[name]
	if !ok {
		return Artifact{}, false
	}
	a, ok := versions[version]
	return a, ok
}

func (s *Store) ListVersions(name string) ([]Artifact, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	versions, ok := s.artifacts[name]
	if !ok {
		return nil, false
	}

	result := make([]Artifact, 0, len(versions))
	for _, a := range versions {
		result = append(result, a)
	}
	return result, true
}

func (s *Store) ListAll() []Artifact {
	s.mu.Lock()
	defer s.mu.Unlock()

	var result []Artifact
	for _, versions := range s.artifacts {
		for _, a := range versions {
			result = append(result, a)
		}
	}
	return result
}

func (s *Store) Count() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	count := 0
	for _, versions := range s.artifacts {
		count += len(versions)
	}
	return count
}

var sha256Regex = regexp.MustCompile(`^[a-f0-9]{64}$`)

func validateArtifact(a Artifact) error {
	if a.Name == "" {
		return errors.New("name is required")
	}
	if a.Version == "" {
		return errors.New("version is required")
	}
	if !sha256Regex.MatchString(a.SHA256) {
		return errors.New("sha256 must be 64 lowercase hex characters")
	}
	if a.SizeBytes == nil {
		return errors.New("size_bytes is required")
	}
	if *a.SizeBytes < 0 {
		return errors.New("size_bytes must be >= 0")
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func registerRoutes(mux *http.ServeMux) {
	store := NewStore()

	mux.HandleFunc("POST /artifacts", func(w http.ResponseWriter, r *http.Request) {
		var a Artifact
		if err := json.NewDecoder(r.Body).Decode(&a); err != nil {
			writeError(w, http.StatusBadRequest, "malformed JSON")
			return
		}

		if err := validateArtifact(a); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}

		if err := store.Create(a); err != nil {
			writeError(w, http.StatusConflict, "artifact already exists")
			return
		}

		artifactsTotal.Set(float64(store.Count()))
		artifactUploadsTotal.Inc()

		writeJSON(w, http.StatusCreated, a)
	})

	mux.HandleFunc("GET /artifacts", func(w http.ResponseWriter, r *http.Request) {
		artifacts := store.ListAll()
		writeJSON(w, http.StatusOK, artifacts)
	})

	mux.HandleFunc("GET /artifacts/{name}", func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")
		versions, ok := store.ListVersions(name)
		if !ok {
			writeError(w, http.StatusNotFound, "artifact not found")
			return
		}
		writeJSON(w, http.StatusOK, versions)
	})

	mux.HandleFunc("GET /artifacts/{name}/{version}", func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")
		version := r.PathValue("version")
		a, ok := store.Get(name, version)
		if !ok {
			writeError(w, http.StatusNotFound, "artifact not found")
			return
		}
		writeJSON(w, http.StatusOK, a)
	})
}

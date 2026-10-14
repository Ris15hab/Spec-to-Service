package main

import (
	"encoding/hex"
	"encoding/json"
	"net/http"
	"sync"
)

type Artifact struct {
	Name      string `json:"name"`
	Version   string `json:"version"`
	SHA256    string `json:"sha256"`
	SizeBytes int64  `json:"size_bytes"`
}

type Store struct {
	mu        sync.Mutex
	artifacts map[string]map[string]Artifact // name -> version -> artifact
}

func NewStore() *Store {
	return &Store{
		artifacts: make(map[string]map[string]Artifact),
	}
}

func (s *Store) Create(a Artifact) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.artifacts[a.Name][a.Version]; exists {
		return false, nil // 409 conflict
	}

	if s.artifacts[a.Name] == nil {
		s.artifacts[a.Name] = make(map[string]Artifact)
	}
	s.artifacts[a.Name][a.Version] = a
	return true, nil
}

func (s *Store) GetAll() []Artifact {
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

func (s *Store) GetByName(name string) ([]Artifact, bool) {
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

func (s *Store) GetByNameVersion(name, version string) (Artifact, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	versions, ok := s.artifacts[name]
	if !ok {
		return Artifact{}, false
	}

	a, ok := versions[version]
	return a, ok
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

func validateArtifact(a Artifact) error {
	if a.Name == "" {
		return errMissingField("name")
	}
	if a.Version == "" {
		return errMissingField("version")
	}
	if a.SHA256 == "" {
		return errMissingField("sha256")
	}
	if len(a.SHA256) != 64 {
		return errInvalidField("sha256", "must be 64 hex characters")
	}
	if _, err := hex.DecodeString(a.SHA256); err != nil {
		return errInvalidField("sha256", "must be lowercase hex")
	}
	// Check it's lowercase
	for _, c := range a.SHA256 {
		if c >= 'A' && c <= 'F' {
			return errInvalidField("sha256", "must be lowercase hex")
		}
	}
	if a.SizeBytes < 0 {
		return errInvalidField("size_bytes", "must be >= 0")
	}
	return nil
}

type validationError struct {
	field string
	msg   string
}

func (e *validationError) Error() string {
	return e.field + ": " + e.msg
}

func errMissingField(field string) error {
	return &validationError{field: field, msg: "required"}
}

func errInvalidField(field, msg string) error {
	return &validationError{field: field, msg: msg}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if v != nil {
		json.NewEncoder(w).Encode(v)
	}
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

		created, err := store.Create(a)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}
		if !created {
			writeError(w, http.StatusConflict, "artifact already exists")
			return
		}

		artifactsTotal.Set(float64(store.Count()))
		artifactUploadsTotal.Inc()

		writeJSON(w, http.StatusCreated, a)
	})

	mux.HandleFunc("GET /artifacts", func(w http.ResponseWriter, r *http.Request) {
		artifacts := store.GetAll()
		writeJSON(w, http.StatusOK, artifacts)
	})

	mux.HandleFunc("GET /artifacts/{name}", func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")
		artifacts, ok := store.GetByName(name)
		if !ok {
			writeError(w, http.StatusNotFound, "artifact not found")
			return
		}
		writeJSON(w, http.StatusOK, artifacts)
	})

	mux.HandleFunc("GET /artifacts/{name}/{version}", func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")
		version := r.PathValue("version")
		artifact, ok := store.GetByNameVersion(name, version)
		if !ok {
			writeError(w, http.StatusNotFound, "artifact not found")
			return
		}
		writeJSON(w, http.StatusOK, artifact)
	})
}

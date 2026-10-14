package main

import (
	"sync"
)

type Artifact struct {
	Name      string `json:"name"`
	Version   string `json:"version"`
	SHA256    string `json:"sha256"`
	SizeBytes int64  `json:"size_bytes"`
}

type Store struct {
	mu        sync.RWMutex
	artifacts map[string]map[string]Artifact // name -> version -> Artifact
}

func NewStore() *Store {
	return &Store{
		artifacts: make(map[string]map[string]Artifact),
	}
}

func (s *Store) Create(a Artifact) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.artifacts[a.Name]; !exists {
		s.artifacts[a.Name] = make(map[string]Artifact)
	}
	if _, exists := s.artifacts[a.Name][a.Version]; exists {
		return false, nil // 409 conflict
	}
	s.artifacts[a.Name][a.Version] = a
	return true, nil
}

func (s *Store) Get(name, version string) (Artifact, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	versions, ok := s.artifacts[name]
	if !ok {
		return Artifact{}, false
	}
	a, ok := versions[version]
	return a, ok
}

func (s *Store) ListVersions(name string) ([]Artifact, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

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
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []Artifact
	for _, versions := range s.artifacts {
		for _, a := range versions {
			result = append(result, a)
		}
	}
	return result
}

func (s *Store) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	count := 0
	for _, versions := range s.artifacts {
		count += len(versions)
	}
	return count
}

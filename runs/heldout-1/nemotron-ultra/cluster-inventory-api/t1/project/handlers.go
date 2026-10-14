package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
)

type Node struct {
	Name   string            `json:"name"`
	Labels map[string]string `json:"labels"`
}

type Store struct {
	mu    sync.Mutex
	nodes map[string]Node
}

func NewStore() *Store {
	return &Store{
		nodes: make(map[string]Node),
	}
}

func (s *Store) Add(node Node) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.nodes[node.Name]; exists {
		return false, nil
	}
	s.nodes[node.Name] = node
	return true, nil
}

func (s *Store) Get(name string) (Node, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	node, ok := s.nodes[name]
	return node, ok
}

func (s *Store) List(filters map[string]string) []Node {
	s.mu.Lock()
	defer s.mu.Unlock()

	result := make([]Node, 0, len(s.nodes))
	for _, node := range s.nodes {
		match := true
		for k, v := range filters {
			if node.Labels[k] != v {
				match = false
				break
			}
		}
		if match {
			result = append(result, node)
		}
	}
	return result
}

func (s *Store) Count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.nodes)
}

func registerRoutes(mux *http.ServeMux) {
	store := NewStore()

	mux.HandleFunc("POST /nodes", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Name   string            `json:"name"`
			Labels map[string]string `json:"labels"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "malformed JSON", http.StatusBadRequest)
			return
		}

		if req.Name == "" {
			http.Error(w, "name is required", http.StatusBadRequest)
			return
		}

		if req.Labels == nil {
			req.Labels = make(map[string]string)
		}

		node := Node{
			Name:   req.Name,
			Labels: req.Labels,
		}

		added, _ := store.Add(node)
		if !added {
			http.Error(w, "node already exists", http.StatusConflict)
			return
		}

		inventoryNodes.Set(float64(store.Count()))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(node)
	})

	mux.HandleFunc("GET /nodes", func(w http.ResponseWriter, r *http.Request) {
		inventoryQueriesTotal.Inc()

		labelParams := r.URL.Query()["label"]
		filters := make(map[string]string)

		for _, lp := range labelParams {
			parts := strings.SplitN(lp, "=", 2)
			if len(parts) != 2 {
				http.Error(w, "label parameter must be key=value", http.StatusBadRequest)
				return
			}
			filters[parts[0]] = parts[1]
		}

		nodes := store.List(filters)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(nodes)
	})

	mux.HandleFunc("GET /nodes/{name}", func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")
		node, ok := store.Get(name)
		if !ok {
			http.Error(w, "node not found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(node)
	})
}

package main

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
)

type Node struct {
	Name     string    `json:"node"`
	Status   string    `json:"status"`
	LastSeen time.Time `json:"last_seen"`
}

type Store struct {
	mu         sync.RWMutex
	nodes      map[string]time.Time
	staleAfter time.Duration
}

func NewStore(staleAfter time.Duration) *Store {
	return &Store{
		nodes:      make(map[string]time.Time),
		staleAfter: staleAfter,
	}
}

func (s *Store) Heartbeat(node string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nodes[node] = time.Now()
}

func (s *Store) GetAll() []Node {
	s.mu.RLock()
	defer s.mu.RUnlock()
	now := time.Now()
	result := make([]Node, 0, len(s.nodes))
	for name, lastSeen := range s.nodes {
		status := "healthy"
		if now.Sub(lastSeen) > s.staleAfter {
			status = "stale"
		}
		result = append(result, Node{
			Name:     name,
			Status:   status,
			LastSeen: lastSeen,
		})
	}
	return result
}

func (s *Store) Get(node string) (Node, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	lastSeen, ok := s.nodes[node]
	if !ok {
		return Node{}, false
	}
	status := "healthy"
	if time.Since(lastSeen) > s.staleAfter {
		status = "stale"
	}
	return Node{
		Name:     node,
		Status:   status,
		LastSeen: lastSeen,
	}, true
}

func (s *Store) CountHealthyStale() (healthy, stale int) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	now := time.Now()
	for _, lastSeen := range s.nodes {
		if now.Sub(lastSeen) > s.staleAfter {
			stale++
		} else {
			healthy++
		}
	}
	return
}

func registerRoutes(mux *http.ServeMux) {
	staleAfterSeconds := 30
	if v := os.Getenv("STALE_AFTER_SECONDS"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			staleAfterSeconds = parsed
		}
	}
	staleAfter := time.Duration(staleAfterSeconds) * time.Second

	store := NewStore(staleAfter)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				healthy, stale := store.CountHealthyStale()
				nodesHealthy.Set(float64(healthy))
				nodesStale.Set(float64(stale))
			}
		}
	}()

	// Ensure the background goroutine can be stopped if needed
	_ = cancel

	mux.HandleFunc("POST /heartbeat", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Node string `json:"node"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "malformed JSON", http.StatusBadRequest)
			return
		}
		if req.Node == "" {
			http.Error(w, "node is required", http.StatusBadRequest)
			return
		}
		store.Heartbeat(req.Node)
		w.WriteHeader(http.StatusNoContent)
	})

	mux.HandleFunc("GET /nodes", func(w http.ResponseWriter, r *http.Request) {
		nodes := store.GetAll()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(nodes)
	})

	mux.HandleFunc("GET /nodes/{node}", func(w http.ResponseWriter, r *http.Request) {
		node := r.PathValue("node")
		n, ok := store.Get(node)
		if !ok {
			http.Error(w, "node not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(n)
	})
}

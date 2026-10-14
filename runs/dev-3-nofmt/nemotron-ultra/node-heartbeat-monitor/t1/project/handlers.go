package main

import (
	"encoding/json"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
)

type NodeInfo struct {
	Node     string    `json:"node"`
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

func (s *Store) GetAll() []NodeInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()

	now := time.Now()
	result := make([]NodeInfo, 0, len(s.nodes))
	for node, lastSeen := range s.nodes {
		status := "healthy"
		if now.Sub(lastSeen) > s.staleAfter {
			status = "stale"
		}
		result = append(result, NodeInfo{
			Node:     node,
			Status:   status,
			LastSeen: lastSeen,
		})
	}
	return result
}

func (s *Store) Get(node string) (NodeInfo, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	lastSeen, ok := s.nodes[node]
	if !ok {
		return NodeInfo{}, false
	}

	now := time.Now()
	status := "healthy"
	if now.Sub(lastSeen) > s.staleAfter {
		status = "stale"
	}

	return NodeInfo{
		Node:     node,
		Status:   status,
		LastSeen: lastSeen,
	}, true
}

func (s *Store) CountHealthyStale() (int, int) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	now := time.Now()
	healthy, stale := 0, 0
	for _, lastSeen := range s.nodes {
		if now.Sub(lastSeen) > s.staleAfter {
			stale++
		} else {
			healthy++
		}
	}
	return healthy, stale
}

func registerRoutes(mux *http.ServeMux) {
	staleAfterStr := os.Getenv("STALE_AFTER_SECONDS")
	staleAfterSec, err := strconv.Atoi(staleAfterStr)
	if err != nil || staleAfterSec <= 0 {
		staleAfterSec = 30
	}
	staleAfter := time.Duration(staleAfterSec) * time.Second

	store := NewStore(staleAfter)

	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for range ticker.C {
			healthy, stale := store.CountHealthyStale()
			nodesHealthy.Set(float64(healthy))
			nodesStale.Set(float64(stale))
		}
	}()

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
		info, ok := store.Get(node)
		if !ok {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(info)
	})
}

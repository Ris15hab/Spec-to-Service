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
	mu         sync.Mutex
	nodes      map[string]time.Time
	staleAfter time.Duration
	stopCh     chan struct{}
}

func NewStore(staleAfter time.Duration) *Store {
	s := &Store{
		nodes:      make(map[string]time.Time),
		staleAfter: staleAfter,
		stopCh:     make(chan struct{}),
	}
	go s.updateMetrics()
	return s
}

func (s *Store) updateMetrics() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			s.recomputeMetrics()
		case <-s.stopCh:
			return
		}
	}
}

func (s *Store) recomputeMetrics() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	healthy := 0
	stale := 0
	for _, lastSeen := range s.nodes {
		if now.Sub(lastSeen) <= s.staleAfter {
			healthy++
		} else {
			stale++
		}
	}
	nodesHealthy.Set(float64(healthy))
	nodesStale.Set(float64(stale))
}

func (s *Store) Heartbeat(node string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nodes[node] = time.Now()
}

func (s *Store) GetNodes() []NodeInfo {
	s.mu.Lock()
	defer s.mu.Unlock()

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

func (s *Store) GetNode(node string) (NodeInfo, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

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

func registerRoutes(mux *http.ServeMux) {
	staleAfterStr := os.Getenv("STALE_AFTER_SECONDS")
	staleAfterSec := 30
	if staleAfterStr != "" {
		if v, err := strconv.Atoi(staleAfterStr); err == nil && v > 0 {
			staleAfterSec = v
		}
	}

	store := NewStore(time.Duration(staleAfterSec) * time.Second)

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
		nodes := store.GetNodes()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(nodes)
	})

	mux.HandleFunc("GET /nodes/{node}", func(w http.ResponseWriter, r *http.Request) {
		node := r.PathValue("node")
		info, ok := store.GetNode(node)
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(info)
	})
}

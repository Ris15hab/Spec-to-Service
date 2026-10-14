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

type nodeInfo struct {
	Node     string    `json:"node"`
	Status   string    `json:"status"`
	LastSeen time.Time `json:"last_seen"`
}

type store struct {
	mu         sync.Mutex
	nodes      map[string]time.Time
	staleAfter time.Duration
}

func newStore() *store {
	staleAfterSec := 30
	if v := os.Getenv("STALE_AFTER_SECONDS"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			staleAfterSec = parsed
		}
	}
	return &store{
		nodes:      make(map[string]time.Time),
		staleAfter: time.Duration(staleAfterSec) * time.Second,
	}
}

func (s *store) heartbeat(node string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nodes[node] = time.Now()
}

func (s *store) getAll() []nodeInfo {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	result := make([]nodeInfo, 0, len(s.nodes))
	for node, lastSeen := range s.nodes {
		status := "healthy"
		if now.Sub(lastSeen) > s.staleAfter {
			status = "stale"
		}
		result = append(result, nodeInfo{
			Node:     node,
			Status:   status,
			LastSeen: lastSeen,
		})
	}
	return result
}

func (s *store) getOne(node string) (nodeInfo, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	lastSeen, ok := s.nodes[node]
	if !ok {
		return nodeInfo{}, false
	}
	status := "healthy"
	if time.Since(lastSeen) > s.staleAfter {
		status = "stale"
	}
	return nodeInfo{
		Node:     node,
		Status:   status,
		LastSeen: lastSeen,
	}, true
}

func (s *store) updateMetrics() {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	healthy := 0
	stale := 0
	for _, lastSeen := range s.nodes {
		if now.Sub(lastSeen) > s.staleAfter {
			stale++
		} else {
			healthy++
		}
	}
	nodesHealthy.Set(float64(healthy))
	nodesStale.Set(float64(stale))
}

func registerRoutes(mux *http.ServeMux) {
	s := newStore()

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.updateMetrics()
			}
		}
	}()

	// Ensure metrics are updated at least once on startup
	s.updateMetrics()

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
		s.heartbeat(req.Node)
		s.updateMetrics()
		w.WriteHeader(http.StatusNoContent)
	})

	mux.HandleFunc("GET /nodes", func(w http.ResponseWriter, r *http.Request) {
		nodes := s.getAll()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(nodes)
	})

	mux.HandleFunc("GET /nodes/{node}", func(w http.ResponseWriter, r *http.Request) {
		node := r.PathValue("node")
		info, ok := s.getOne(node)
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(info)
	})

	// Store cancel function for potential cleanup (not required for tests)
	_ = cancel
}

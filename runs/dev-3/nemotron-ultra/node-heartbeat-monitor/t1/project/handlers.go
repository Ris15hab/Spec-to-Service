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
	mu       sync.RWMutex
	nodes    map[string]time.Time
	staleDur time.Duration
}

func newStore() *store {
	staleAfter := 30 * time.Second
	if v := os.Getenv("STALE_AFTER_SECONDS"); v != "" {
		if secs, err := strconv.Atoi(v); err == nil && secs > 0 {
			staleAfter = time.Duration(secs) * time.Second
		}
	}
	return &store{
		nodes:    make(map[string]time.Time),
		staleDur: staleAfter,
	}
}

func (s *store) heartbeat(node string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nodes[node] = time.Now()
}

func (s *store) getAll() []nodeInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()
	now := time.Now()
	result := make([]nodeInfo, 0, len(s.nodes))
	for node, lastSeen := range s.nodes {
		status := "healthy"
		if now.Sub(lastSeen) > s.staleDur {
			status = "stale"
		}
		result = append(result, nodeInfo{
			Node:     node,
			Status:   status,
			LastSeen: lastSeen.UTC(),
		})
	}
	return result
}

func (s *store) get(node string) (nodeInfo, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	lastSeen, ok := s.nodes[node]
	if !ok {
		return nodeInfo{}, false
	}
	now := time.Now()
	status := "healthy"
	if now.Sub(lastSeen) > s.staleDur {
		status = "stale"
	}
	return nodeInfo{
		Node:     node,
		Status:   status,
		LastSeen: lastSeen.UTC(),
	}, true
}

func (s *store) recomputeGauges() {
	s.mu.RLock()
	defer s.mu.RUnlock()
	now := time.Now()
	healthy := 0
	stale := 0
	for _, lastSeen := range s.nodes {
		if now.Sub(lastSeen) > s.staleDur {
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

	// Background goroutine to recompute gauges every second
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.recomputeGauges()
			}
		}
	}()
	_ = cancel // not used in tests, but keeps the pattern clean

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
		w.WriteHeader(http.StatusNoContent)
	})

	mux.HandleFunc("GET /nodes", func(w http.ResponseWriter, r *http.Request) {
		nodes := s.getAll()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(nodes)
	})

	mux.HandleFunc("GET /nodes/{node}", func(w http.ResponseWriter, r *http.Request) {
		node := r.PathValue("node")
		info, ok := s.get(node)
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(info)
	})
}

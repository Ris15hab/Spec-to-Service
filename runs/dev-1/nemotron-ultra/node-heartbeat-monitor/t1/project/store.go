package main

import (
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
	stopCh     chan struct{}
}

func NewStore() *Store {
	staleAfterSec := 30
	if v := os.Getenv("STALE_AFTER_SECONDS"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			staleAfterSec = parsed
		}
	}
	s := &Store{
		nodes:      make(map[string]time.Time),
		staleAfter: time.Duration(staleAfterSec) * time.Second,
		stopCh:     make(chan struct{}),
	}
	go s.updateMetricsLoop()
	return s
}

func (s *Store) updateMetricsLoop() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			s.updateMetrics()
		case <-s.stopCh:
			return
		}
	}
}

func (s *Store) updateMetrics() {
	s.mu.RLock()
	defer s.mu.RUnlock()
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

func (s *Store) GetAllNodes() []NodeInfo {
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

func (s *Store) GetNode(node string) (NodeInfo, bool) {
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

func (s *Store) Stop() {
	close(s.stopCh)
}

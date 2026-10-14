package main

import (
	"encoding/json"
	"net/http"
	"os"
	"syscall"
	"time"
)

func registerRoutes(mux *http.ServeMux) {
	store := &diskStore{}
	store.startBackgroundRefresh()

	mux.HandleFunc("GET /disk", store.handleDisk)
}

type diskStore struct{}

func (s *diskStore) startBackgroundRefresh() {
	go func() {
		s.refreshRootMetrics()
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			s.refreshRootMetrics()
		}
	}()
}

func (s *diskStore) refreshRootMetrics() {
	var stat syscall.Statfs_t
	if err := syscall.Statfs("/", &stat); err != nil {
		return
	}
	total := uint64(stat.Blocks) * uint64(stat.Bsize)
	free := uint64(stat.Bfree) * uint64(stat.Bsize)
	used := total - free
	var usedPercent float64
	if total > 0 {
		usedPercent = float64(used) / float64(total) * 100
	}
	diskUsedPercent.Set(usedPercent)
	diskFreeBytes.Set(float64(free))
}

func (s *diskStore) handleDisk(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		http.Error(w, `{"error":"path query parameter is required"}`, http.StatusBadRequest)
		return
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		http.Error(w, `{"error":"path not found"}`, http.StatusNotFound)
		return
	}

	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		http.Error(w, `{"error":"failed to get filesystem stats"}`, http.StatusInternalServerError)
		return
	}

	total := uint64(stat.Blocks) * uint64(stat.Bsize)
	free := uint64(stat.Bfree) * uint64(stat.Bsize)
	used := total - free
	var usedPercent float64
	if total > 0 {
		usedPercent = float64(used) / float64(total) * 100
	}

	resp := map[string]any{
		"path":          path,
		"total_bytes":   total,
		"free_bytes":    free,
		"used_percent":  usedPercent,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

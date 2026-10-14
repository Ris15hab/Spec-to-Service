package main

import (
	"encoding/json"
	"net/http"
	"sync"
	"syscall"
	"time"
)

type DiskStats struct {
	Path        string `json:"path"`
	TotalBytes  uint64 `json:"total_bytes"`
	FreeBytes   uint64 `json:"free_bytes"`
	UsedPercent uint64 `json:"used_percent"`
}

type store struct {
	mu       sync.Mutex
	rootPath string
}

func newStore() *store {
	return &store{
		rootPath: "/",
	}
}

func (s *store) getDiskStats(path string) (*DiskStats, error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return nil, err
	}

	totalBytes := uint64(stat.Blocks) * uint64(stat.Bsize)
	freeBytes := uint64(stat.Bfree) * uint64(stat.Bsize)
	usedBytes := totalBytes - freeBytes
	var usedPercent uint64
	if totalBytes > 0 {
		usedPercent = (usedBytes * 100) / totalBytes
	}

	return &DiskStats{
		Path:        path,
		TotalBytes:  totalBytes,
		FreeBytes:   freeBytes,
		UsedPercent: usedPercent,
	}, nil
}

func (s *store) refreshRootMetrics() {
	stats, err := s.getDiskStats(s.rootPath)
	if err != nil {
		return
	}
	diskUsedPercent.Set(float64(stats.UsedPercent))
	diskFreeBytes.Set(float64(stats.FreeBytes))
}

func (s *store) startBackgroundRefresh() {
	s.refreshRootMetrics()
	ticker := time.NewTicker(10 * time.Second)
	go func() {
		for range ticker.C {
			s.refreshRootMetrics()
		}
	}()
}

func registerRoutes(mux *http.ServeMux) {
	s := newStore()
	s.startBackgroundRefresh()

	mux.HandleFunc("GET /disk", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Query().Get("path")
		if path == "" {
			http.Error(w, "path query parameter is required", http.StatusBadRequest)
			return
		}

		stats, err := s.getDiskStats(path)
		if err != nil {
			http.Error(w, "path not found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(stats)
	})
}

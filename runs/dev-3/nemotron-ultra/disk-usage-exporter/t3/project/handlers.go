package main

import (
	"encoding/json"
	"net/http"
	"sync"
	"syscall"
	"time"
)

type diskInfo struct {
	Path        string `json:"path"`
	TotalBytes  uint64 `json:"total_bytes"`
	FreeBytes   uint64 `json:"free_bytes"`
	UsedPercent uint64 `json:"used_percent"`
}

type store struct {
	mu sync.Mutex
}

func newStore() *store {
	return &store{}
}

func (s *store) getDiskInfo(path string) (*diskInfo, error) {
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

	return &diskInfo{
		Path:        path,
		TotalBytes:  totalBytes,
		FreeBytes:   freeBytes,
		UsedPercent: usedPercent,
	}, nil
}

func (s *store) refreshRootMetrics() {
	info, err := s.getDiskInfo("/")
	if err != nil {
		return
	}
	diskUsedPercent.Set(float64(info.UsedPercent))
	diskFreeBytes.Set(float64(info.FreeBytes))
}

func registerRoutes(mux *http.ServeMux) {
	st := newStore()

	// Initial metrics refresh
	st.refreshRootMetrics()

	// Background goroutine for periodic refresh
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			st.refreshRootMetrics()
		}
	}()

	mux.HandleFunc("GET /disk", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Query().Get("path")
		if path == "" {
			http.Error(w, "path query parameter is required", http.StatusBadRequest)
			return
		}

		info, err := st.getDiskInfo(path)
		if err != nil {
			http.Error(w, "path not found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(info)
	})
}

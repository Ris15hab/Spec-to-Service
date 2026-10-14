package main

import (
	"encoding/json"
	"net/http"
	"sync"
	"syscall"
	"time"
)

type DiskInfo struct {
	Path        string  `json:"path"`
	TotalBytes  uint64  `json:"total_bytes"`
	FreeBytes   uint64  `json:"free_bytes"`
	UsedPercent float64 `json:"used_percent"`
}

type store struct {
	mu sync.Mutex
}

func newStore() *store {
	return &store{}
}

func (s *store) getDiskInfo(path string) (DiskInfo, error) {
	var stat syscall.Statfs_t
	err := syscall.Statfs(path, &stat)
	if err != nil {
		return DiskInfo{}, err
	}

	totalBytes := uint64(stat.Blocks) * uint64(stat.Bsize)
	freeBytes := uint64(stat.Bfree) * uint64(stat.Bsize)
	usedBytes := totalBytes - freeBytes
	usedPercent := 0.0
	if totalBytes > 0 {
		usedPercent = float64(usedBytes) / float64(totalBytes) * 100.0
	}

	return DiskInfo{
		Path:        path,
		TotalBytes:  totalBytes,
		FreeBytes:   freeBytes,
		UsedPercent: usedPercent,
	}, nil
}

func (s *store) updateRootMetrics() {
	info, err := s.getDiskInfo("/")
	if err != nil {
		return
	}
	diskUsedPercent.Set(info.UsedPercent)
	diskFreeBytes.Set(float64(info.FreeBytes))
}

func registerRoutes(mux *http.ServeMux) {
	st := newStore()

	// Background goroutine to refresh root filesystem metrics
	go func() {
		st.updateRootMetrics()
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			st.updateRootMetrics()
		}
	}()

	mux.HandleFunc("GET /disk", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Query().Get("path")
		if path == "" {
			http.Error(w, `{"error": "path query parameter is required"}`, http.StatusBadRequest)
			return
		}

		info, err := st.getDiskInfo(path)
		if err != nil {
			http.Error(w, `{"error": "path not found"}`, http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(info)
	})
}

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

type Store struct {
	mu sync.Mutex
}

func NewStore() *Store {
	return &Store{}
}

func (s *Store) GetDiskInfo(path string) (DiskInfo, error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
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

func (s *Store) RefreshRootMetrics() {
	info, err := s.GetDiskInfo("/")
	if err != nil {
		return
	}
	diskUsedPercent.Set(info.UsedPercent)
	diskFreeBytes.Set(float64(info.FreeBytes))
}

func diskHandler(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Query().Get("path")
		if path == "" {
			http.Error(w, "path query parameter is required", http.StatusBadRequest)
			return
		}

		info, err := store.GetDiskInfo(path)
		if err != nil {
			http.Error(w, "path not found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(info)
	}
}

func registerRoutes(mux *http.ServeMux) {
	store := NewStore()

	// Initial metrics refresh
	store.RefreshRootMetrics()

	// Background goroutine to refresh metrics every 10 seconds
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			store.RefreshRootMetrics()
		}
	}()

	mux.HandleFunc("GET /disk", diskHandler(store))
}

package main

import (
	"encoding/json"
	"net/http"
	"syscall"
)

func registerRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /disk", handleDisk)
}

func handleDisk(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		http.Error(w, `{"error": "path query parameter is required"}`, http.StatusBadRequest)
		return
	}

	var stat syscall.Statfs_t
	err := syscall.Statfs(path, &stat)
	if err != nil {
		http.Error(w, `{"error": "path not found"}`, http.StatusNotFound)
		return
	}

	totalBytes := uint64(stat.Blocks) * uint64(stat.Bsize)
	freeBytes := uint64(stat.Bfree) * uint64(stat.Bsize)
	usedBytes := totalBytes - freeBytes
	usedPercent := float64(usedBytes) / float64(totalBytes) * 100

	response := map[string]interface{}{
		"path":         path,
		"total_bytes":  totalBytes,
		"free_bytes":   freeBytes,
		"used_percent": usedPercent,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

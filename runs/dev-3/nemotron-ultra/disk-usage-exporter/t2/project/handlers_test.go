package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDiskHandlerSuccess(t *testing.T) {
	server := newServer()
	req := httptest.NewRequest(http.MethodGet, "/disk?path=/", nil)
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var info DiskInfo
	if err := json.NewDecoder(w.Body).Decode(&info); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if info.Path != "/" {
		t.Errorf("expected path '/', got %s", info.Path)
	}
	if info.TotalBytes == 0 {
		t.Error("expected total_bytes > 0")
	}
	if info.FreeBytes == 0 {
		t.Error("expected free_bytes > 0")
	}
	if info.UsedPercent < 0 || info.UsedPercent > 100 {
		t.Errorf("expected used_percent between 0 and 100, got %f", info.UsedPercent)
	}
}

func TestDiskHandlerMissingPath(t *testing.T) {
	server := newServer()
	req := httptest.NewRequest(http.MethodGet, "/disk", nil)
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestDiskHandlerNotFound(t *testing.T) {
	server := newServer()
	req := httptest.NewRequest(http.MethodGet, "/disk?path=/definitely/not/here", nil)
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

func TestHealthEndpoint(t *testing.T) {
	server := newServer()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestMetricsEndpoint(t *testing.T) {
	server := newServer()
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

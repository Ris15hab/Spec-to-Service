package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDiskEndpointSuccess(t *testing.T) {
	server := newServer()
	req := httptest.NewRequest(http.MethodGet, "/disk?path=/", nil)
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var info diskInfo
	if err := json.NewDecoder(w.Body).Decode(&info); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if info.Path != "/" {
		t.Errorf("expected path '/', got %q", info.Path)
	}
	if info.TotalBytes == 0 {
		t.Error("expected total_bytes > 0")
	}
	if info.FreeBytes == 0 {
		t.Error("expected free_bytes > 0")
	}
	if info.UsedPercent > 100 {
		t.Errorf("expected used_percent <= 100, got %d", info.UsedPercent)
	}
}

func TestDiskEndpointMissingPath(t *testing.T) {
	server := newServer()
	req := httptest.NewRequest(http.MethodGet, "/disk", nil)
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestDiskEndpointNotFound(t *testing.T) {
	server := newServer()
	req := httptest.NewRequest(http.MethodGet, "/disk?path=/definitely/not/here", nil)
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

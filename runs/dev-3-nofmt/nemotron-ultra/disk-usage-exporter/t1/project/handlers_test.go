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

	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp["path"] != "/" {
		t.Errorf("expected path '/', got %v", resp["path"])
	}
	if _, ok := resp["total_bytes"]; !ok {
		t.Error("missing total_bytes in response")
	}
	if _, ok := resp["free_bytes"]; !ok {
		t.Error("missing free_bytes in response")
	}
	if _, ok := resp["used_percent"]; !ok {
		t.Error("missing used_percent in response")
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

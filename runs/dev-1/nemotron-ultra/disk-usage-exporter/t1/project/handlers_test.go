package main

import (
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

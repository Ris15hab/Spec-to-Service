package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestPutAndGet(t *testing.T) {
	srv := newServer()

	// PUT a value
	reqBody := `{"value": "v", "ttl_seconds": 60}`
	req := httptest.NewRequest(http.MethodPut, "/cache/k1", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated && rr.Code != http.StatusOK {
		t.Errorf("PUT /cache/k1: expected 200 or 201, got %d", rr.Code)
	}

	// GET the value
	req = httptest.NewRequest(http.MethodGet, "/cache/k1", nil)
	rr = httptest.NewRecorder()
	srv.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("GET /cache/k1: expected 200, got %d", rr.Code)
	}
	var resp map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp["key"] != "k1" || resp["value"] != "v" {
		t.Errorf("unexpected response: %+v", resp)
	}
}

func TestGetMissingKey(t *testing.T) {
	srv := newServer()

	req := httptest.NewRequest(http.MethodGet, "/cache/nope", nil)
	rr := httptest.NewRecorder()
	srv.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Errorf("GET /cache/nope: expected 404, got %d", rr.Code)
	}
}

func TestPutInvalidRequests(t *testing.T) {
	srv := newServer()

	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{"missing value", `{"ttl_seconds": 5}`, http.StatusBadRequest},
		{"zero ttl", `{"value": "v", "ttl_seconds": 0}`, http.StatusBadRequest},
		{"negative ttl", `{"value": "v", "ttl_seconds": -1}`, http.StatusBadRequest},
		{"malformed JSON", `{invalid}`, http.StatusBadRequest},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPut, "/cache/k2", bytes.NewBufferString(tc.body))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()
			srv.ServeHTTP(rr, req)
			if rr.Code != tc.wantStatus {
				t.Errorf("expected %d, got %d", tc.wantStatus, rr.Code)
			}
		})
	}
}

func TestDelete(t *testing.T) {
	srv := newServer()

	// First put a value
	reqBody := `{"value": "v", "ttl_seconds": 60}`
	req := httptest.NewRequest(http.MethodPut, "/cache/k1", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated && rr.Code != http.StatusOK {
		t.Fatalf("PUT failed: %d", rr.Code)
	}

	// Delete it
	req = httptest.NewRequest(http.MethodDelete, "/cache/k1", nil)
	rr = httptest.NewRecorder()
	srv.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK && rr.Code != http.StatusNoContent {
		t.Errorf("DELETE /cache/k1: expected 200 or 204, got %d", rr.Code)
	}

	// Verify it's gone
	req = httptest.NewRequest(http.MethodGet, "/cache/k1", nil)
	rr = httptest.NewRecorder()
	srv.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Errorf("GET after DELETE: expected 404, got %d", rr.Code)
	}
}

func TestTTLExpiration(t *testing.T) {
	srv := newServer()

	// PUT with very short TTL
	reqBody := `{"value": "v", "ttl_seconds": 1}`
	req := httptest.NewRequest(http.MethodPut, "/cache/ttlkey", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated && rr.Code != http.StatusOK {
		t.Fatalf("PUT failed: %d", rr.Code)
	}

	// Immediately GET - should work
	req = httptest.NewRequest(http.MethodGet, "/cache/ttlkey", nil)
	rr = httptest.NewRecorder()
	srv.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("immediate GET: expected 200, got %d", rr.Code)
	}

	// Wait for expiration
	time.Sleep(1500 * time.Millisecond)

	// GET after expiration - should 404
	req = httptest.NewRequest(http.MethodGet, "/cache/ttlkey", nil)
	rr = httptest.NewRecorder()
	srv.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Errorf("expired GET: expected 404, got %d", rr.Code)
	}
}

func TestHealthAndMetrics(t *testing.T) {
	srv := newServer()

	// Health endpoint
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()
	srv.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("/health: expected 200, got %d", rr.Code)
	}

	// Metrics endpoint
	req = httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rr = httptest.NewRecorder()
	srv.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("/metrics: expected 200, got %d", rr.Code)
	}
}

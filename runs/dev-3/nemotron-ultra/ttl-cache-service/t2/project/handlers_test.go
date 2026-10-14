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
	server := newServer()

	// PUT a value
	body := `{"value": "v", "ttl_seconds": 60}`
	req := httptest.NewRequest(http.MethodPut, "/cache/k1", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated && rr.Code != http.StatusOK {
		t.Errorf("PUT /cache/k1: expected 200 or 201, got %d", rr.Code)
	}

	// GET the value
	req = httptest.NewRequest(http.MethodGet, "/cache/k1", nil)
	rr = httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("GET /cache/k1: expected 200, got %d", rr.Code)
	}

	var resp getResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if resp.Key != "k1" || resp.Value != "v" {
		t.Errorf("unexpected response: %+v", resp)
	}
}

func TestGetMissingKey(t *testing.T) {
	server := newServer()

	req := httptest.NewRequest(http.MethodGet, "/cache/nope", nil)
	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("GET /cache/nope: expected 404, got %d", rr.Code)
	}
}

func TestPutInvalidRequests(t *testing.T) {
	server := newServer()

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
			server.ServeHTTP(rr, req)

			if rr.Code != tc.wantStatus {
				t.Errorf("expected %d, got %d", tc.wantStatus, rr.Code)
			}
		})
	}
}

func TestDelete(t *testing.T) {
	server := newServer()

	// First PUT a value
	body := `{"value": "v", "ttl_seconds": 60}`
	req := httptest.NewRequest(http.MethodPut, "/cache/k1", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	// DELETE it
	req = httptest.NewRequest(http.MethodDelete, "/cache/k1", nil)
	rr = httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK && rr.Code != http.StatusNoContent {
		t.Errorf("DELETE /cache/k1: expected 200 or 204, got %d", rr.Code)
	}

	// GET should now 404
	req = httptest.NewRequest(http.MethodGet, "/cache/k1", nil)
	rr = httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("GET /cache/k1 after delete: expected 404, got %d", rr.Code)
	}
}

func TestExpiry(t *testing.T) {
	server := newServer()

	// PUT with 1 second TTL
	body := `{"value": "v", "ttl_seconds": 1}`
	req := httptest.NewRequest(http.MethodPut, "/cache/expire", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("PUT: expected 201, got %d", rr.Code)
	}

	// Immediately GET - should work
	req = httptest.NewRequest(http.MethodGet, "/cache/expire", nil)
	rr = httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("immediate GET: expected 200, got %d", rr.Code)
	}

	// Wait for expiry
	time.Sleep(1100 * time.Millisecond)

	// GET after expiry - should 404
	req = httptest.NewRequest(http.MethodGet, "/cache/expire", nil)
	rr = httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expired GET: expected 404, got %d", rr.Code)
	}
}

func TestMetrics(t *testing.T) {
	server := newServer()

	// PUT a value
	body := `{"value": "v", "ttl_seconds": 60}`
	req := httptest.NewRequest(http.MethodPut, "/cache/m1", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	// GET hit
	req = httptest.NewRequest(http.MethodGet, "/cache/m1", nil)
	rr = httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	// GET miss
	req = httptest.NewRequest(http.MethodGet, "/cache/missing", nil)
	rr = httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	// Check metrics endpoint responds
	req = httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rr = httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("/metrics: expected 200, got %d", rr.Code)
	}
}

func TestHealth(t *testing.T) {
	server := newServer()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("/health: expected 200, got %d", rr.Code)
	}
}

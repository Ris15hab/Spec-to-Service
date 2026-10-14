package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestPutAndGet(t *testing.T) {
	srv := newServer()

	// PUT a value
	req := httptest.NewRequest(http.MethodPut, "/cache/k1", nil)
	req.Header.Set("Content-Type", "application/json")
	body := `{"value": "v", "ttl_seconds": 60}`
	req.Body = jsonBody(body)
	rr := httptest.NewRecorder()
	srv.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated && rr.Code != http.StatusOK {
		t.Errorf("PUT /cache/k1: expected 201 or 200, got %d", rr.Code)
	}

	// GET the value
	req = httptest.NewRequest(http.MethodGet, "/cache/k1", nil)
	rr = httptest.NewRecorder()
	srv.ServeHTTP(rr, req)
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
		{"malformed JSON", `{not json}`, http.StatusBadRequest},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPut, "/cache/k2", nil)
			req.Header.Set("Content-Type", "application/json")
			req.Body = jsonBody(tc.body)
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

	// PUT a value
	req := httptest.NewRequest(http.MethodPut, "/cache/k1", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Body = jsonBody(`{"value": "v", "ttl_seconds": 60}`)
	rr := httptest.NewRecorder()
	srv.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated && rr.Code != http.StatusOK {
		t.Fatalf("PUT failed: %d", rr.Code)
	}

	// DELETE the value
	req = httptest.NewRequest(http.MethodDelete, "/cache/k1", nil)
	rr = httptest.NewRecorder()
	srv.ServeHTTP(rr, req)
	if rr.Code != http.StatusNoContent && rr.Code != http.StatusOK {
		t.Errorf("DELETE /cache/k1: expected 204 or 200, got %d", rr.Code)
	}

	// GET should now 404
	req = httptest.NewRequest(http.MethodGet, "/cache/k1", nil)
	rr = httptest.NewRecorder()
	srv.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Errorf("GET after DELETE: expected 404, got %d", rr.Code)
	}
}

func TestTTLExpiry(t *testing.T) {
	srv := newServer()

	// PUT with 1 second TTL
	req := httptest.NewRequest(http.MethodPut, "/cache/k1", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Body = jsonBody(`{"value": "v", "ttl_seconds": 1}`)
	rr := httptest.NewRecorder()
	srv.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated && rr.Code != http.StatusOK {
		t.Fatalf("PUT failed: %d", rr.Code)
	}

	// Immediately GET - should succeed
	req = httptest.NewRequest(http.MethodGet, "/cache/k1", nil)
	rr = httptest.NewRecorder()
	srv.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("immediate GET: expected 200, got %d", rr.Code)
	}

	// Wait for expiry
	time.Sleep(1100 * time.Millisecond)

	// GET after expiry - should 404
	req = httptest.NewRequest(http.MethodGet, "/cache/k1", nil)
	rr = httptest.NewRecorder()
	srv.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Errorf("after expiry GET: expected 404, got %d", rr.Code)
	}
}

func TestMetricsIncrement(t *testing.T) {
	srv := newServer()

	// Initial state - get baseline
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rr := httptest.NewRecorder()
	srv.ServeHTTP(rr, req)
	_ = rr.Body.String() // baseline

	// PUT a value
	req = httptest.NewRequest(http.MethodPut, "/cache/m1", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Body = jsonBody(`{"value": "v", "ttl_seconds": 60}`)
	rr = httptest.NewRecorder()
	srv.ServeHTTP(rr, req)

	// GET hit
	req = httptest.NewRequest(http.MethodGet, "/cache/m1", nil)
	rr = httptest.NewRecorder()
	srv.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("GET hit failed: %d", rr.Code)
	}

	// GET miss
	req = httptest.NewRequest(http.MethodGet, "/cache/missing", nil)
	rr = httptest.NewRecorder()
	srv.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("GET miss failed: %d", rr.Code)
	}

	// Check metrics
	req = httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rr = httptest.NewRecorder()
	srv.ServeHTTP(rr, req)
	metricsOut := rr.Body.String()

	// Just verify metrics endpoint works and contains our metric names
	if !contains(metricsOut, "cache_hits_total") {
		t.Errorf("metrics missing cache_hits_total")
	}
	if !contains(metricsOut, "cache_misses_total") {
		t.Errorf("metrics missing cache_misses_total")
	}
	if !contains(metricsOut, "cache_entries") {
		t.Errorf("metrics missing cache_entries")
	}
}

func jsonBody(s string) *jsonBodyReader {
	return &jsonBodyReader{data: []byte(s)}
}

type jsonBodyReader struct {
	data []byte
	pos  int
}

func (r *jsonBodyReader) Read(p []byte) (n int, err error) {
	if r.pos >= len(r.data) {
		return 0, nil // EOF
	}
	n = copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}

func (r *jsonBodyReader) Close() error { return nil }

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

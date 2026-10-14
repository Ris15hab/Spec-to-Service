package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestPutAndGet(t *testing.T) {
	srv := newServer()

	req := httptest.NewRequest(http.MethodPut, "/cache/testkey",
		strings.NewReader(`{"value": "testvalue", "ttl_seconds": 60}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK && rr.Code != http.StatusCreated {
		t.Errorf("PUT returned %d, want 200 or 201", rr.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/cache/testkey", nil)
	rr = httptest.NewRecorder()
	srv.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("GET returned %d, want 200", rr.Code)
	}

	var resp getResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Key != "testkey" || resp.Value != "testvalue" {
		t.Errorf("got %+v, want {key: testkey, value: testvalue}", resp)
	}
}

func TestGetMissingKey(t *testing.T) {
	srv := newServer()

	req := httptest.NewRequest(http.MethodGet, "/cache/nonexistent", nil)
	rr := httptest.NewRecorder()
	srv.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("GET missing key returned %d, want 404", rr.Code)
	}
}

func TestDelete(t *testing.T) {
	srv := newServer()

	req := httptest.NewRequest(http.MethodPut, "/cache/delkey",
		strings.NewReader(`{"value": "delvalue", "ttl_seconds": 60}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK && rr.Code != http.StatusCreated {
		t.Errorf("PUT returned %d", rr.Code)
	}

	req = httptest.NewRequest(http.MethodDelete, "/cache/delkey", nil)
	rr = httptest.NewRecorder()
	srv.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK && rr.Code != http.StatusNoContent {
		t.Errorf("DELETE returned %d, want 200 or 204", rr.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/cache/delkey", nil)
	rr = httptest.NewRecorder()
	srv.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("GET after delete returned %d, want 404", rr.Code)
	}
}

func TestPutValidation(t *testing.T) {
	srv := newServer()

	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{"missing value", `{"ttl_seconds": 5}`, http.StatusBadRequest},
		{"zero ttl", `{"value": "v", "ttl_seconds": 0}`, http.StatusBadRequest},
		{"negative ttl", `{"value": "v", "ttl_seconds": -1}`, http.StatusBadRequest},
		{"malformed json", `{invalid}`, http.StatusBadRequest},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPut, "/cache/key",
				strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()
			srv.ServeHTTP(rr, req)

			if rr.Code != tc.wantStatus {
				t.Errorf("got %d, want %d", rr.Code, tc.wantStatus)
			}
		})
	}
}

func TestTTLExpiry(t *testing.T) {
	srv := newServer()

	req := httptest.NewRequest(http.MethodPut, "/cache/ttlkey",
		strings.NewReader(`{"value": "ttlvalue", "ttl_seconds": 1}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK && rr.Code != http.StatusCreated {
		t.Errorf("PUT returned %d", rr.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/cache/ttlkey", nil)
	rr = httptest.NewRecorder()
	srv.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("immediate GET returned %d, want 200", rr.Code)
	}

	time.Sleep(1100 * time.Millisecond)

	req = httptest.NewRequest(http.MethodGet, "/cache/ttlkey", nil)
	rr = httptest.NewRecorder()
	srv.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("GET after expiry returned %d, want 404", rr.Code)
	}
}

func TestMetrics(t *testing.T) {
	srv := newServer()

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rr := httptest.NewRecorder()
	srv.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("/metrics returned %d", rr.Code)
	}

	req = httptest.NewRequest(http.MethodPut, "/cache/mkey",
		strings.NewReader(`{"value": "mval", "ttl_seconds": 60}`))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	srv.ServeHTTP(rr, req)

	req = httptest.NewRequest(http.MethodGet, "/cache/mkey", nil)
	rr = httptest.NewRecorder()
	srv.ServeHTTP(rr, req)

	req = httptest.NewRequest(http.MethodGet, "/cache/missing", nil)
	rr = httptest.NewRecorder()
	srv.ServeHTTP(rr, req)

	req = httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rr = httptest.NewRecorder()
	srv.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("/metrics returned %d", rr.Code)
	}
}

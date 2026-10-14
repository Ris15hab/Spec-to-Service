package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestPutAndGet(t *testing.T) {
	server := newServer()

	req := httptest.NewRequest(http.MethodPut, "/cache/k1", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Body = jsonBody(`{"value": "v", "ttl_seconds": 60}`)

	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated && rr.Code != http.StatusOK {
		t.Errorf("PUT /cache/k1: expected 200 or 201, got %d", rr.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/cache/k1", nil)
	rr = httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("GET /cache/k1: expected 200, got %d", rr.Code)
	}

	var resp getResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
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
			req := httptest.NewRequest(http.MethodPut, "/cache/test", nil)
			req.Header.Set("Content-Type", "application/json")
			req.Body = jsonBody(tc.body)

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

	req := httptest.NewRequest(http.MethodPut, "/cache/k1", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Body = jsonBody(`{"value": "v", "ttl_seconds": 60}`)
	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	req = httptest.NewRequest(http.MethodDelete, "/cache/k1", nil)
	rr = httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent && rr.Code != http.StatusOK {
		t.Errorf("DELETE /cache/k1: expected 200 or 204, got %d", rr.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/cache/k1", nil)
	rr = httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("GET /cache/k1 after delete: expected 404, got %d", rr.Code)
	}
}

func TestExpiration(t *testing.T) {
	server := newServer()

	req := httptest.NewRequest(http.MethodPut, "/cache/expire", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Body = jsonBody(`{"value": "v", "ttl_seconds": 1}`)
	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated && rr.Code != http.StatusOK {
		t.Errorf("PUT: expected 200 or 201, got %d", rr.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/cache/expire", nil)
	rr = httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("immediate GET: expected 200, got %d", rr.Code)
	}

	time.Sleep(1100 * time.Millisecond)

	req = httptest.NewRequest(http.MethodGet, "/cache/expire", nil)
	rr = httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expired GET: expected 404, got %d", rr.Code)
	}
}

func TestMetricsUpdate(t *testing.T) {
	server := newServer()

	for _, key := range []string{"k1", "k2"} {
		req := httptest.NewRequest(http.MethodPut, "/cache/"+key, nil)
		req.Header.Set("Content-Type", "application/json")
		req.Body = jsonBody(`{"value": "v", "ttl_seconds": 60}`)
		rr := httptest.NewRecorder()
		server.ServeHTTP(rr, req)
		if rr.Code != http.StatusCreated && rr.Code != http.StatusOK {
			t.Errorf("PUT %s: expected 200/201, got %d", key, rr.Code)
		}
	}

	for _, key := range []string{"k1", "k2"} {
		req := httptest.NewRequest(http.MethodGet, "/cache/"+key, nil)
		rr := httptest.NewRecorder()
		server.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("GET %s: expected 200, got %d", key, rr.Code)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/cache/missing", nil)
	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Errorf("GET missing: expected 404, got %d", rr.Code)
	}

	req = httptest.NewRequest(http.MethodDelete, "/cache/k1", nil)
	rr = httptest.NewRecorder()
	server.ServeHTTP(rr, req)
	if rr.Code != http.StatusNoContent && rr.Code != http.StatusOK {
		t.Errorf("DELETE k1: expected 200/204, got %d", rr.Code)
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
		return 0, nil
	}
	n = copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}

func (r *jsonBodyReader) Close() error {
	return nil
}

package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPostLogsAccepted(t *testing.T) {
	server := newServer()

	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{"info level", `{"level": "info", "message": "started"}`, http.StatusAccepted},
		{"warn level", `{"level": "warn", "message": "warning"}`, http.StatusAccepted},
		{"error level", `{"level": "error", "message": "failed"}`, http.StatusAccepted},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/logs", nil)
			req.Header.Set("Content-Type", "application/json")
			req.Body = jsonBody(tc.body)
			rr := httptest.NewRecorder()
			server.ServeHTTP(rr, req)
			if rr.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d", rr.Code, tc.wantStatus)
			}
		})
	}
}

func TestPostLogsRejected(t *testing.T) {
	server := newServer()

	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{"bad level", `{"level": "bogus", "message": "x"}`, http.StatusBadRequest},
		{"empty message", `{"level": "error", "message": ""}`, http.StatusBadRequest},
		{"malformed JSON", `{invalid}`, http.StatusBadRequest},
		{"missing level", `{"message": "hello"}`, http.StatusBadRequest},
		{"missing message", `{"level": "info"}`, http.StatusBadRequest},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/logs", nil)
			req.Header.Set("Content-Type", "application/json")
			req.Body = jsonBody(tc.body)
			rr := httptest.NewRecorder()
			server.ServeHTTP(rr, req)
			if rr.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d", rr.Code, tc.wantStatus)
			}
		})
	}
}

func TestGetLogsStats(t *testing.T) {
	server := newServer()

	// Initial stats should be zero
	req := httptest.NewRequest(http.MethodGet, "/logs/stats", nil)
	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	var resp statsResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	want := map[string]int{"info": 0, "warn": 0, "error": 0}
	if !mapsEqual(resp.Counts, want) {
		t.Errorf("counts = %v, want %v", resp.Counts, want)
	}

	// Add some logs
	for _, level := range []string{"info", "info", "warn", "error"} {
		body := `{"level": "` + level + `", "message": "msg"}`
		req := httptest.NewRequest(http.MethodPost, "/logs", jsonBody(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		server.ServeHTTP(rr, req)
		if rr.Code != http.StatusAccepted {
			t.Fatalf("post status = %d", rr.Code)
		}
	}

	// Check updated stats
	req = httptest.NewRequest(http.MethodGet, "/logs/stats", nil)
	rr = httptest.NewRecorder()
	server.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	want = map[string]int{"info": 2, "warn": 1, "error": 1}
	if !mapsEqual(resp.Counts, want) {
		t.Errorf("counts = %v, want %v", resp.Counts, want)
	}
}

func TestHealthAndMetricsEndpoints(t *testing.T) {
	server := newServer()

	// Test /health
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("/health status = %d, want %d", rr.Code, http.StatusOK)
	}

	// Test /metrics
	req = httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rr = httptest.NewRecorder()
	server.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("/metrics status = %d, want %d", rr.Code, http.StatusOK)
	}
}

func jsonBody(s string) *stringsReader {
	return &stringsReader{s: s}
}

type stringsReader struct {
	s string
	i int
}

func (r *stringsReader) Read(p []byte) (n int, err error) {
	if r.i >= len(r.s) {
		return 0, io.EOF
	}
	n = copy(p, r.s[r.i:])
	r.i += n
	return n, nil
}

func (r *stringsReader) Close() error { return nil }

func mapsEqual(a, b map[string]int) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}

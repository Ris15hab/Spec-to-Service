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
		wantLevel  string
	}{
		{"info level", `{"level": "info", "message": "started"}`, http.StatusAccepted, "info"},
		{"warn level", `{"level": "warn", "message": "warning"}`, http.StatusAccepted, "warn"},
		{"error level", `{"level": "error", "message": "failed"}`, http.StatusAccepted, "error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/logs", nil)
			req.Header.Set("Content-Type", "application/json")
			req.Body = jsonBody(tt.body)

			rr := httptest.NewRecorder()
			server.ServeHTTP(rr, req)

			if rr.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rr.Code, tt.wantStatus)
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
		{"malformed JSON", `{not json}`, http.StatusBadRequest},
		{"missing level", `{"message": "hello"}`, http.StatusBadRequest},
		{"missing message", `{"level": "info"}`, http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/logs", nil)
			req.Header.Set("Content-Type", "application/json")
			req.Body = jsonBody(tt.body)

			rr := httptest.NewRecorder()
			server.ServeHTTP(rr, req)

			if rr.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rr.Code, tt.wantStatus)
			}
		})
	}
}

func TestGetLogsStats(t *testing.T) {
	server := newServer()

	// Post some logs first
	for _, level := range []string{"info", "warn", "error"} {
		req := httptest.NewRequest(http.MethodPost, "/logs", jsonBody(`{"level": "`+level+`", "message": "test"}`))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		server.ServeHTTP(rr, req)
		if rr.Code != http.StatusAccepted {
			t.Fatalf("setup failed: status = %d", rr.Code)
		}
	}

	// Get stats
	req := httptest.NewRequest(http.MethodGet, "/logs/stats", nil)
	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusOK)
	}

	var resp statsResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	expected := map[string]int{"info": 1, "warn": 1, "error": 1}
	for k, v := range expected {
		if resp.Counts[k] != v {
			t.Errorf("counts[%s] = %d, want %d", k, resp.Counts[k], v)
		}
	}

	// Verify all three keys present
	for _, k := range []string{"info", "warn", "error"} {
		if _, ok := resp.Counts[k]; !ok {
			t.Errorf("missing key %q in counts", k)
		}
	}
}

func TestStatsIsolation(t *testing.T) {
	// Each call to newServer should start with empty state
	server1 := newServer()
	server2 := newServer()

	// Add log to server1
	req := httptest.NewRequest(http.MethodPost, "/logs", jsonBody(`{"level": "info", "message": "test"}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	server1.ServeHTTP(rr, req)

	// Check server1 has count
	req = httptest.NewRequest(http.MethodGet, "/logs/stats", nil)
	rr = httptest.NewRecorder()
	server1.ServeHTTP(rr, req)
	var resp1 statsResponse
	json.NewDecoder(rr.Body).Decode(&resp1)

	// Check server2 has zero count
	req = httptest.NewRequest(http.MethodGet, "/logs/stats", nil)
	rr = httptest.NewRecorder()
	server2.ServeHTTP(rr, req)
	var resp2 statsResponse
	json.NewDecoder(rr.Body).Decode(&resp2)

	if resp1.Counts["info"] != 1 {
		t.Errorf("server1 info count = %d, want 1", resp1.Counts["info"])
	}
	if resp2.Counts["info"] != 0 {
		t.Errorf("server2 info count = %d, want 0", resp2.Counts["info"])
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

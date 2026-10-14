package main

import (
	"encoding/json"
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
		{"malformed JSON", `{invalid}`, http.StatusBadRequest},
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

func TestGetStats(t *testing.T) {
	server := newServer()

	// Ingest some logs first
	logs := []string{
		`{"level": "info", "message": "a"}`,
		`{"level": "info", "message": "b"}`,
		`{"level": "warn", "message": "c"}`,
		`{"level": "error", "message": "d"}`,
		`{"level": "error", "message": "e"}`,
		`{"level": "error", "message": "f"}`,
	}

	for _, body := range logs {
		req := httptest.NewRequest(http.MethodPost, "/logs", jsonBody(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		server.ServeHTTP(rr, req)
		if rr.Code != http.StatusAccepted {
			t.Fatalf("failed to ingest log: %d", rr.Code)
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

	want := map[string]int{"info": 2, "warn": 1, "error": 3}
	if resp.Counts["info"] != want["info"] {
		t.Errorf("info count = %d, want %d", resp.Counts["info"], want["info"])
	}
	if resp.Counts["warn"] != want["warn"] {
		t.Errorf("warn count = %d, want %d", resp.Counts["warn"], want["warn"])
	}
	if resp.Counts["error"] != want["error"] {
		t.Errorf("error count = %d, want %d", resp.Counts["error"], want["error"])
	}
}

func TestStatsAllKeysPresent(t *testing.T) {
	server := newServer()

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

	for _, key := range []string{"info", "warn", "error"} {
		if _, ok := resp.Counts[key]; !ok {
			t.Errorf("missing key %q in counts", key)
		}
		if resp.Counts[key] != 0 {
			t.Errorf("count for %q = %d, want 0", key, resp.Counts[key])
		}
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

func (r *jsonBodyReader) Close() error { return nil }

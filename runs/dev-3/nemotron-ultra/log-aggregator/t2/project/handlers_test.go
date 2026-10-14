package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
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

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/logs", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")

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
			req := httptest.NewRequest(http.MethodPost, "/logs", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")

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

	ingest := func(level, msg string) {
		body := `{"level": "` + level + `", "message": "` + msg + `"}`
		req := httptest.NewRequest(http.MethodPost, "/logs", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		server.ServeHTTP(rr, req)
		if rr.Code != http.StatusAccepted {
			t.Fatalf("ingest failed: %d", rr.Code)
		}
	}

	ingest("info", "first")
	ingest("info", "second")
	ingest("warn", "third")
	ingest("error", "fourth")

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

	want := map[string]int{"info": 2, "warn": 1, "error": 1}
	if resp.Counts["info"] != want["info"] || resp.Counts["warn"] != want["warn"] || resp.Counts["error"] != want["error"] {
		t.Errorf("counts = %v, want %v", resp.Counts, want)
	}

	for _, k := range []string{"info", "warn", "error"} {
		if _, ok := resp.Counts[k]; !ok {
			t.Errorf("missing key %q in response", k)
		}
	}
}

func TestHealthEndpoint(t *testing.T) {
	server := newServer()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusOK)
	}
}

func TestMetricsEndpoint(t *testing.T) {
	server := newServer()

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusOK)
	}
}

package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPostLogsAccepted(t *testing.T) {
	server := newServer()

	tests := []struct {
		name       string
		level      string
		message    string
		wantStatus int
	}{
		{"info level", "info", "started", http.StatusAccepted},
		{"warn level", "warn", "warning", http.StatusAccepted},
		{"error level", "error", "failed", http.StatusAccepted},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := map[string]string{"level": tt.level, "message": tt.message}
			jsonBody, _ := json.Marshal(body)
			req := httptest.NewRequest(http.MethodPost, "/logs", bytes.NewReader(jsonBody))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()
			server.ServeHTTP(rr, req)

			if rr.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, rr.Code)
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
		{"invalid level", `{"level": "bogus", "message": "x"}`, http.StatusBadRequest},
		{"empty message", `{"level": "error", "message": ""}`, http.StatusBadRequest},
		{"malformed JSON", `{not json}`, http.StatusBadRequest},
		{"missing level", `{"message": "hello"}`, http.StatusBadRequest},
		{"missing message", `{"level": "info"}`, http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/logs", bytes.NewReader([]byte(tt.body)))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()
			server.ServeHTTP(rr, req)

			if rr.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, rr.Code)
			}
		})
	}
}

func TestGetStats(t *testing.T) {
	server := newServer()

	// Test initial stats are all zero
	req := httptest.NewRequest(http.MethodGet, "/logs/stats", nil)
	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var resp statsResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}

	expected := map[string]int{"info": 0, "warn": 0, "error": 0}
	if resp.Counts["info"] != expected["info"] || resp.Counts["warn"] != expected["warn"] || resp.Counts["error"] != expected["error"] {
		t.Errorf("expected %v, got %v", expected, resp.Counts)
	}

	// Ingest some logs
	postLog := func(level, message string) {
		body := map[string]string{"level": level, "message": message}
		jsonBody, _ := json.Marshal(body)
		req := httptest.NewRequest(http.MethodPost, "/logs", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		server.ServeHTTP(rr, req)
		if rr.Code != http.StatusAccepted {
			t.Fatalf("post log failed: %d", rr.Code)
		}
	}

	postLog("info", "first")
	postLog("info", "second")
	postLog("warn", "warning")
	postLog("error", "error1")
	postLog("error", "error2")
	postLog("error", "error3")

	// Check stats
	req = httptest.NewRequest(http.MethodGet, "/logs/stats", nil)
	rr = httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}

	expected = map[string]int{"info": 2, "warn": 1, "error": 3}
	if resp.Counts["info"] != expected["info"] || resp.Counts["warn"] != expected["warn"] || resp.Counts["error"] != expected["error"] {
		t.Errorf("expected %v, got %v", expected, resp.Counts)
	}
}

func TestRejectedLinesNotCounted(t *testing.T) {
	server := newServer()

	// Try to post invalid logs
	invalidBodies := []string{
		`{"level": "bogus", "message": "x"}`,
		`{"level": "error", "message": ""}`,
		`{not json}`,
	}

	for _, body := range invalidBodies {
		req := httptest.NewRequest(http.MethodPost, "/logs", bytes.NewReader([]byte(body)))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		server.ServeHTTP(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected 400 for %s, got %d", body, rr.Code)
		}
	}

	// Stats should still be all zero
	req := httptest.NewRequest(http.MethodGet, "/logs/stats", nil)
	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	var resp statsResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}

	expected := map[string]int{"info": 0, "warn": 0, "error": 0}
	if resp.Counts["info"] != expected["info"] || resp.Counts["warn"] != expected["warn"] || resp.Counts["error"] != expected["error"] {
		t.Errorf("rejected lines were counted: expected %v, got %v", expected, resp.Counts)
	}
}

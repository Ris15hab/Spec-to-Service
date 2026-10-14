package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newTestServer() *httptest.Server {
	mux := http.NewServeMux()
	registerRoutes(mux)
	return httptest.NewServer(mux)
}

func TestPostLogsAccepted(t *testing.T) {
	server := newTestServer()
	defer server.Close()

	body := `{"level": "info", "message": "started"}`
	req, err := http.NewRequest(http.MethodPost, server.URL+"/logs", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	resp, err := server.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		t.Errorf("expected 202, got %d", resp.StatusCode)
	}
}

func TestPostLogsBadLevel(t *testing.T) {
	server := newTestServer()
	defer server.Close()

	body := `{"level": "bogus", "message": "x"}`
	req, err := http.NewRequest(http.MethodPost, server.URL+"/logs", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	resp, err := server.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestPostLogsEmptyMessage(t *testing.T) {
	server := newTestServer()
	defer server.Close()

	body := `{"level": "error", "message": ""}`
	req, err := http.NewRequest(http.MethodPost, server.URL+"/logs", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	resp, err := server.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestPostLogsMalformedJSON(t *testing.T) {
	server := newTestServer()
	defer server.Close()

	body := `{invalid json}`
	req, err := http.NewRequest(http.MethodPost, server.URL+"/logs", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	resp, err := server.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestGetStats(t *testing.T) {
	server := newTestServer()
	defer server.Close()

	// First, post some logs
	testCases := []struct {
		level   string
		message string
	}{
		{"info", "first"},
		{"info", "second"},
		{"warn", "warning"},
		{"error", "error"},
	}

	for _, tc := range testCases {
		body, _ := json.Marshal(map[string]string{"level": tc.level, "message": tc.message})
		req, err := http.NewRequest(http.MethodPost, server.URL+"/logs", strings.NewReader(string(body)))
		if err != nil {
			t.Fatal(err)
		}
		resp, err := server.Client().Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusAccepted {
			t.Fatalf("failed to post log: %d", resp.StatusCode)
		}
	}

	// Now get stats
	req, err := http.NewRequest(http.MethodGet, server.URL+"/logs/stats", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := server.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	var respBody statsResponse
	if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	expected := map[string]int64{"info": 2, "warn": 1, "error": 1}
	for k, v := range expected {
		if respBody.Counts[k] != v {
			t.Errorf("counts[%s] = %d, want %d", k, respBody.Counts[k], v)
		}
	}
	// Ensure all three keys present
	for _, k := range []string{"info", "warn", "error"} {
		if _, ok := respBody.Counts[k]; !ok {
			t.Errorf("missing key %s in counts", k)
		}
	}
}

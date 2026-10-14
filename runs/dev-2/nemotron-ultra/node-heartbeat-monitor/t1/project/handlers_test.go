package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func TestHeartbeatValid(t *testing.T) {
	server := newServer()

	req := httptest.NewRequest(http.MethodPost, "/heartbeat", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Body = jsonBody(`{"node": "n1"}`)

	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", rr.Code)
	}

	// Verify node appears in list
	req = httptest.NewRequest(http.MethodGet, "/nodes", nil)
	rr = httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}

	var nodes []NodeInfo
	if err := json.NewDecoder(rr.Body).Decode(&nodes); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if len(nodes) != 1 || nodes[0].Node != "n1" || nodes[0].Status != "healthy" {
		t.Errorf("unexpected nodes: %+v", nodes)
	}
}

func TestHeartbeatInvalid(t *testing.T) {
	server := newServer()

	tests := []struct {
		name string
		body string
	}{
		{"empty JSON", `{}`},
		{"missing node", `{"node": ""}`},
		{"malformed JSON", `{invalid`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/heartbeat", nil)
			req.Header.Set("Content-Type", "application/json")
			req.Body = jsonBody(tc.body)

			rr := httptest.NewRecorder()
			server.ServeHTTP(rr, req)

			if rr.Code != http.StatusBadRequest {
				t.Errorf("expected 400, got %d", rr.Code)
			}
		})
	}
}

func TestNodesEndpoints(t *testing.T) {
	server := newServer()

	// Add two nodes
	for _, node := range []string{"n1", "n2"} {
		req := httptest.NewRequest(http.MethodPost, "/heartbeat", nil)
		req.Header.Set("Content-Type", "application/json")
		req.Body = jsonBody(`{"node": "` + node + `"}`)
		rr := httptest.NewRecorder()
		server.ServeHTTP(rr, req)
		if rr.Code != http.StatusNoContent {
			t.Fatalf("heartbeat failed: %d", rr.Code)
		}
	}

	// Test GET /nodes
	req := httptest.NewRequest(http.MethodGet, "/nodes", nil)
	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("GET /nodes: expected 200, got %d", rr.Code)
	}

	var nodes []NodeInfo
	if err := json.NewDecoder(rr.Body).Decode(&nodes); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if len(nodes) != 2 {
		t.Errorf("expected 2 nodes, got %d", len(nodes))
	}

	// Test GET /nodes/n1
	req = httptest.NewRequest(http.MethodGet, "/nodes/n1", nil)
	rr = httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("GET /nodes/n1: expected 200, got %d", rr.Code)
	}

	var node NodeInfo
	if err := json.NewDecoder(rr.Body).Decode(&node); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if node.Node != "n1" || node.Status != "healthy" {
		t.Errorf("unexpected node: %+v", node)
	}

	// Test GET /nodes/unknown
	req = httptest.NewRequest(http.MethodGet, "/nodes/unknown", nil)
	rr = httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("GET /nodes/unknown: expected 404, got %d", rr.Code)
	}
}

func TestStaleDetection(t *testing.T) {
	// Use short stale threshold for test
	os.Setenv("STALE_AFTER_SECONDS", "1")
	defer os.Unsetenv("STALE_AFTER_SECONDS")

	server := newServer()

	// Add node
	req := httptest.NewRequest(http.MethodPost, "/heartbeat", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Body = jsonBody(`{"node": "n1"}`)
	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("heartbeat failed: %d", rr.Code)
	}

	// Immediately check - should be healthy
	req = httptest.NewRequest(http.MethodGet, "/nodes/n1", nil)
	rr = httptest.NewRecorder()
	server.ServeHTTP(rr, req)
	var node NodeInfo
	json.NewDecoder(rr.Body).Decode(&node)
	if node.Status != "healthy" {
		t.Errorf("expected healthy immediately, got %s", node.Status)
	}

	// Wait for stale threshold + metrics update interval
	time.Sleep(2 * time.Second)

	// Check again - should be stale
	req = httptest.NewRequest(http.MethodGet, "/nodes/n1", nil)
	rr = httptest.NewRecorder()
	server.ServeHTTP(rr, req)
	json.NewDecoder(rr.Body).Decode(&node)
	if node.Status != "stale" {
		t.Errorf("expected stale after timeout, got %s", node.Status)
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

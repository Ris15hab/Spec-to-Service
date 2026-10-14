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

func TestHeartbeatSuccess(t *testing.T) {
	server := newServer()

	req := httptest.NewRequest(http.MethodPost, "/heartbeat", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Body = jsonBody(`{"node": "n1"}`)
	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", rr.Code)
	}
}

func TestHeartbeatMissingNode(t *testing.T) {
	server := newServer()

	req := httptest.NewRequest(http.MethodPost, "/heartbeat", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Body = jsonBody(`{}`)
	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestHeartbeatMalformedJSON(t *testing.T) {
	server := newServer()

	req := httptest.NewRequest(http.MethodPost, "/heartbeat", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Body = jsonBody(`{invalid}`)
	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestNodesList(t *testing.T) {
	server := newServer()

	// Post a heartbeat first
	req := httptest.NewRequest(http.MethodPost, "/heartbeat", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Body = jsonBody(`{"node": "n1"}`)
	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("heartbeat failed: %d", rr.Code)
	}

	// Get nodes list
	req = httptest.NewRequest(http.MethodGet, "/nodes", nil)
	rr = httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}

	var nodes []Node
	if err := json.NewDecoder(rr.Body).Decode(&nodes); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(nodes) != 1 {
		t.Errorf("expected 1 node, got %d", len(nodes))
	}
	if nodes[0].Name != "n1" {
		t.Errorf("expected node n1, got %s", nodes[0].Name)
	}
	if nodes[0].Status != "healthy" {
		t.Errorf("expected healthy, got %s", nodes[0].Status)
	}
}

func TestNodesGetSingle(t *testing.T) {
	server := newServer()

	// Post a heartbeat first
	req := httptest.NewRequest(http.MethodPost, "/heartbeat", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Body = jsonBody(`{"node": "n1"}`)
	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("heartbeat failed: %d", rr.Code)
	}

	// Get single node
	req = httptest.NewRequest(http.MethodGet, "/nodes/n1", nil)
	rr = httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}

	var node Node
	if err := json.NewDecoder(rr.Body).Decode(&node); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if node.Name != "n1" {
		t.Errorf("expected node n1, got %s", node.Name)
	}
	if node.Status != "healthy" {
		t.Errorf("expected healthy, got %s", node.Status)
	}
}

func TestNodesGetUnknown(t *testing.T) {
	server := newServer()

	req := httptest.NewRequest(http.MethodGet, "/nodes/unknown", nil)
	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

func TestStaleDetection(t *testing.T) {
	// Use a very short stale threshold for testing
	os.Setenv("STALE_AFTER_SECONDS", "1")
	defer os.Unsetenv("STALE_AFTER_SECONDS")

	server := newServer()

	// Post a heartbeat
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
	var node Node
	json.NewDecoder(rr.Body).Decode(&node)
	if node.Status != "healthy" {
		t.Errorf("expected healthy immediately after heartbeat, got %s", node.Status)
	}

	// Wait for stale threshold to pass
	time.Sleep(1500 * time.Millisecond)

	// Check again - should be stale
	req = httptest.NewRequest(http.MethodGet, "/nodes/n1", nil)
	rr = httptest.NewRecorder()
	server.ServeHTTP(rr, req)
	json.NewDecoder(rr.Body).Decode(&node)
	if node.Status != "stale" {
		t.Errorf("expected stale after timeout, got %s", node.Status)
	}
}

func TestHealthEndpoint(t *testing.T) {
	server := newServer()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
}

func TestMetricsEndpoint(t *testing.T) {
	server := newServer()

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
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

func (r *stringsReader) Close() error {
	return nil
}

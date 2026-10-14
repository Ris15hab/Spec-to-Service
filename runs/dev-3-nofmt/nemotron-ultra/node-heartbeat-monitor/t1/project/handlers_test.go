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

	// Verify node appears in /nodes
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

func TestHeartbeatInvalidJSON(t *testing.T) {
	server := newServer()

	req := httptest.NewRequest(http.MethodPost, "/heartbeat", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Body = jsonBody(`{invalid`)

	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
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

func TestNodesListAndSingle(t *testing.T) {
	server := newServer()

	// Post heartbeats for two nodes
	for _, node := range []string{"n1", "n2"} {
		req := httptest.NewRequest(http.MethodPost, "/heartbeat", nil)
		req.Header.Set("Content-Type", "application/json")
		req.Body = jsonBody(`{"node": "` + node + `"}`)
		rr := httptest.NewRecorder()
		server.ServeHTTP(rr, req)
		if rr.Code != http.StatusNoContent {
			t.Errorf("heartbeat for %s: expected 204, got %d", node, rr.Code)
		}
	}

	// GET /nodes
	req := httptest.NewRequest(http.MethodGet, "/nodes", nil)
	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}

	var nodes []NodeInfo
	if err := json.NewDecoder(rr.Body).Decode(&nodes); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if len(nodes) != 2 {
		t.Errorf("expected 2 nodes, got %d", len(nodes))
	}

	// GET /nodes/n1
	req = httptest.NewRequest(http.MethodGet, "/nodes/n1", nil)
	rr = httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}

	var info NodeInfo
	if err := json.NewDecoder(rr.Body).Decode(&info); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if info.Node != "n1" || info.Status != "healthy" {
		t.Errorf("unexpected node info: %+v", info)
	}

	// GET /nodes/unknown
	req = httptest.NewRequest(http.MethodGet, "/nodes/unknown", nil)
	rr = httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

func TestStaleDetection(t *testing.T) {
	// Use a short stale threshold for testing
	os.Setenv("STALE_AFTER_SECONDS", "1")
	defer os.Unsetenv("STALE_AFTER_SECONDS")

	server := newServer()

	// Post heartbeat
	req := httptest.NewRequest(http.MethodPost, "/heartbeat", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Body = jsonBody(`{"node": "n1"}`)
	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("heartbeat: expected 204, got %d", rr.Code)
	}

	// Immediately check - should be healthy
	req = httptest.NewRequest(http.MethodGet, "/nodes/n1", nil)
	rr = httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	var info NodeInfo
	json.NewDecoder(rr.Body).Decode(&info)
	if info.Status != "healthy" {
		t.Errorf("expected healthy immediately, got %s", info.Status)
	}

	// Wait for staleness
	time.Sleep(1500 * time.Millisecond)

	// Check again - should be stale
	req = httptest.NewRequest(http.MethodGet, "/nodes/n1", nil)
	rr = httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	json.NewDecoder(rr.Body).Decode(&info)
	if info.Status != "stale" {
		t.Errorf("expected stale after wait, got %s", info.Status)
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

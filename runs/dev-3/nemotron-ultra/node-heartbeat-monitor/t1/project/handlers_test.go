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
	srv := newServer()
	req := httptest.NewRequest(http.MethodPost, "/heartbeat", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Body = jsonBody(`{"node": "n1"}`)
	rr := httptest.NewRecorder()
	srv.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", rr.Code)
	}
}

func TestHeartbeatMissingNode(t *testing.T) {
	srv := newServer()
	req := httptest.NewRequest(http.MethodPost, "/heartbeat", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Body = jsonBody(`{}`)
	rr := httptest.NewRecorder()
	srv.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestHeartbeatMalformedJSON(t *testing.T) {
	srv := newServer()
	req := httptest.NewRequest(http.MethodPost, "/heartbeat", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Body = jsonBody(`{invalid}`)
	rr := httptest.NewRecorder()
	srv.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestNodesList(t *testing.T) {
	srv := newServer()
	// Post a heartbeat first
	postHeartbeat(t, srv, "n1")

	req := httptest.NewRequest(http.MethodGet, "/nodes", nil)
	rr := httptest.NewRecorder()
	srv.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}

	var nodes []nodeInfo
	if err := json.NewDecoder(rr.Body).Decode(&nodes); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(nodes) != 1 {
		t.Errorf("expected 1 node, got %d", len(nodes))
	}
	if nodes[0].Node != "n1" {
		t.Errorf("expected node n1, got %s", nodes[0].Node)
	}
	if nodes[0].Status != "healthy" {
		t.Errorf("expected healthy, got %s", nodes[0].Status)
	}
}

func TestNodesGetOne(t *testing.T) {
	srv := newServer()
	postHeartbeat(t, srv, "n1")

	req := httptest.NewRequest(http.MethodGet, "/nodes/n1", nil)
	rr := httptest.NewRecorder()
	srv.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}

	var info nodeInfo
	if err := json.NewDecoder(rr.Body).Decode(&info); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if info.Node != "n1" {
		t.Errorf("expected node n1, got %s", info.Node)
	}
}

func TestNodesGetUnknown(t *testing.T) {
	srv := newServer()

	req := httptest.NewRequest(http.MethodGet, "/nodes/unknown", nil)
	rr := httptest.NewRecorder()
	srv.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

func TestStaleDetection(t *testing.T) {
	// Use a short stale duration for testing
	os.Setenv("STALE_AFTER_SECONDS", "1")
	defer os.Unsetenv("STALE_AFTER_SECONDS")

	srv := newServer()
	postHeartbeat(t, srv, "n1")

	// Immediately check - should be healthy
	req := httptest.NewRequest(http.MethodGet, "/nodes/n1", nil)
	rr := httptest.NewRecorder()
	srv.ServeHTTP(rr, req)
	var info nodeInfo
	json.NewDecoder(rr.Body).Decode(&info)
	if info.Status != "healthy" {
		t.Errorf("expected healthy immediately, got %s", info.Status)
	}

	// Wait for staleness
	time.Sleep(1500 * time.Millisecond)

	req = httptest.NewRequest(http.MethodGet, "/nodes/n1", nil)
	rr = httptest.NewRecorder()
	srv.ServeHTTP(rr, req)
	json.NewDecoder(rr.Body).Decode(&info)
	if info.Status != "stale" {
		t.Errorf("expected stale after wait, got %s", info.Status)
	}
}

func postHeartbeat(t *testing.T, srv http.Handler, node string) {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/heartbeat", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Body = jsonBody(`{"node": "` + node + `"}`)
	rr := httptest.NewRecorder()
	srv.ServeHTTP(rr, req)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("heartbeat failed: %d", rr.Code)
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
		return 0, io.EOF
	}
	n = copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}

func (r *jsonBodyReader) Close() error { return nil }

package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func TestHeartbeatSuccess(t *testing.T) {
	srv := newServer()
	req := httptest.NewRequest(http.MethodPost, "/heartbeat", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Body = jsonBody(`{"node": "n1"}`)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", rec.Code)
	}
}

func TestHeartbeatMissingNode(t *testing.T) {
	srv := newServer()
	req := httptest.NewRequest(http.MethodPost, "/heartbeat", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Body = jsonBody(`{}`)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestHeartbeatMalformedJSON(t *testing.T) {
	srv := newServer()
	req := httptest.NewRequest(http.MethodPost, "/heartbeat", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Body = jsonBody(`{invalid}`)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestListNodes(t *testing.T) {
	srv := newServer()
	// First add a node
	req := httptest.NewRequest(http.MethodPost, "/heartbeat", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Body = jsonBody(`{"node": "n1"}`)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("heartbeat failed: %d", rec.Code)
	}

	// Now list nodes
	req = httptest.NewRequest(http.MethodGet, "/nodes", nil)
	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	var nodes []NodeInfo
	if err := json.NewDecoder(rec.Body).Decode(&nodes); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if len(nodes) != 1 || nodes[0].Node != "n1" || nodes[0].Status != "healthy" {
		t.Errorf("unexpected nodes: %+v", nodes)
	}
}

func TestGetNode(t *testing.T) {
	srv := newServer()
	// Add a node
	req := httptest.NewRequest(http.MethodPost, "/heartbeat", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Body = jsonBody(`{"node": "n1"}`)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("heartbeat failed: %d", rec.Code)
	}

	// Get the node
	req = httptest.NewRequest(http.MethodGet, "/nodes/n1", nil)
	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	var node NodeInfo
	if err := json.NewDecoder(rec.Body).Decode(&node); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if node.Node != "n1" || node.Status != "healthy" {
		t.Errorf("unexpected node: %+v", node)
	}
}

func TestGetUnknownNode(t *testing.T) {
	srv := newServer()
	req := httptest.NewRequest(http.MethodGet, "/nodes/unknown", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestStaleNode(t *testing.T) {
	// Use a very short stale threshold for testing
	oldEnv := os.Getenv("STALE_AFTER_SECONDS")
	os.Setenv("STALE_AFTER_SECONDS", "1")
	defer func() {
		if oldEnv == "" {
			os.Unsetenv("STALE_AFTER_SECONDS")
		} else {
			os.Setenv("STALE_AFTER_SECONDS", oldEnv)
		}
	}()

	srv := newServer()
	req := httptest.NewRequest(http.MethodPost, "/heartbeat", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Body = jsonBody(`{"node": "n1"}`)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("heartbeat failed: %d", rec.Code)
	}

	// Wait for node to become stale
	time.Sleep(2 * time.Second)

	req = httptest.NewRequest(http.MethodGet, "/nodes/n1", nil)
	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	var node NodeInfo
	if err := json.NewDecoder(rec.Body).Decode(&node); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if node.Status != "stale" {
		t.Errorf("expected stale, got %s", node.Status)
	}
}

func jsonBody(s string) io.ReadCloser {
	return io.NopCloser(strings.NewReader(s))
}

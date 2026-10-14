package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRegisterAndGetNode(t *testing.T) {
	server := newServer()

	// Register a node
	reqBody := `{"name": "n1", "labels": {"zone": "a", "gpu": "h100"}}`
	req := httptest.NewRequest(http.MethodPost, "/nodes", nil)
	req.Body = jsonBody(reqBody)
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var node Node
	if err := json.Unmarshal(w.Body.Bytes(), &node); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if node.Name != "n1" {
		t.Errorf("expected name n1, got %s", node.Name)
	}
	if node.Labels["zone"] != "a" || node.Labels["gpu"] != "h100" {
		t.Errorf("labels mismatch: %v", node.Labels)
	}

	// Get the node
	req = httptest.NewRequest(http.MethodGet, "/nodes/n1", nil)
	w = httptest.NewRecorder()
	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var gotNode Node
	if err := json.Unmarshal(w.Body.Bytes(), &gotNode); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if gotNode.Name != "n1" {
		t.Errorf("expected name n1, got %s", gotNode.Name)
	}
}

func TestDuplicateNodeReturnsConflict(t *testing.T) {
	server := newServer()

	reqBody := `{"name": "n1", "labels": {}}`
	req := httptest.NewRequest(http.MethodPost, "/nodes", nil)
	req.Body = jsonBody(reqBody)
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("first request expected 201, got %d", w.Code)
	}

	// Try to register same name again
	req = httptest.NewRequest(http.MethodPost, "/nodes", nil)
	req.Body = jsonBody(reqBody)
	w = httptest.NewRecorder()
	server.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", w.Code, w.Body.String())
	}
}

func TestMissingNameReturnsBadRequest(t *testing.T) {
	server := newServer()

	reqBody := `{"labels": {}}`
	req := httptest.NewRequest(http.MethodPost, "/nodes", nil)
	req.Body = jsonBody(reqBody)
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestListNodesWithLabelFilter(t *testing.T) {
	server := newServer()

	// Register two nodes
	nodes := []string{
		`{"name": "n1", "labels": {"zone": "a", "gpu": "h100"}}`,
		`{"name": "n2", "labels": {"zone": "b", "gpu": "a100"}}`,
	}
	for _, body := range nodes {
		req := httptest.NewRequest(http.MethodPost, "/nodes", nil)
		req.Body = jsonBody(body)
		w := httptest.NewRecorder()
		server.ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("setup failed: expected 201, got %d", w.Code)
		}
	}

	// Filter by zone=a
	req := httptest.NewRequest(http.MethodGet, "/nodes?label=zone=a", nil)
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var result []Node
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 node, got %d", len(result))
	}
	if result[0].Name != "n1" {
		t.Errorf("expected n1, got %s", result[0].Name)
	}

	// Filter by two labels (both must match)
	req = httptest.NewRequest(http.MethodGet, "/nodes?label=zone=a&label=gpu=h100", nil)
	w = httptest.NewRecorder()
	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	result = nil
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 node, got %d", len(result))
	}
	if result[0].Name != "n1" {
		t.Errorf("expected n1, got %s", result[0].Name)
	}

	// Filter with no matches
	req = httptest.NewRequest(http.MethodGet, "/nodes?label=zone=c", nil)
	w = httptest.NewRecorder()
	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	result = nil
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected 0 nodes, got %d", len(result))
	}
}

func TestInvalidLabelParameterReturnsBadRequest(t *testing.T) {
	server := newServer()

	req := httptest.NewRequest(http.MethodGet, "/nodes?label=broken", nil)
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUnknownNodeReturnsNotFound(t *testing.T) {
	server := newServer()

	req := httptest.NewRequest(http.MethodGet, "/nodes/zzz", nil)
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
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

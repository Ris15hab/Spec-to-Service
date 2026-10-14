package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRegisterNode(t *testing.T) {
	server := newServer()

	// Test successful registration
	body := `{"name": "n1", "labels": {"zone": "a", "gpu": "h100"}}`
	req := httptest.NewRequest(http.MethodPost, "/nodes", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d", rr.Code)
	}

	var node Node
	if err := json.NewDecoder(rr.Body).Decode(&node); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if node.Name != "n1" {
		t.Errorf("expected name n1, got %s", node.Name)
	}
	if node.Labels["zone"] != "a" || node.Labels["gpu"] != "h100" {
		t.Errorf("labels mismatch: %v", node.Labels)
	}

	// Test duplicate registration returns 409
	req2 := httptest.NewRequest(http.MethodPost, "/nodes", strings.NewReader(body))
	req2.Header.Set("Content-Type", "application/json")
	rr2 := httptest.NewRecorder()
	server.ServeHTTP(rr2, req2)

	if rr2.Code != http.StatusConflict {
		t.Errorf("expected 409 for duplicate, got %d", rr2.Code)
	}

	// Test missing name returns 400
	badBody := `{"labels": {}}`
	req3 := httptest.NewRequest(http.MethodPost, "/nodes", strings.NewReader(badBody))
	req3.Header.Set("Content-Type", "application/json")
	rr3 := httptest.NewRecorder()
	server.ServeHTTP(rr3, req3)

	if rr3.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing name, got %d", rr3.Code)
	}

	// Test malformed JSON returns 400
	malformed := `{invalid json}`
	req4 := httptest.NewRequest(http.MethodPost, "/nodes", strings.NewReader(malformed))
	req4.Header.Set("Content-Type", "application/json")
	rr4 := httptest.NewRecorder()
	server.ServeHTTP(rr4, req4)

	if rr4.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for malformed JSON, got %d", rr4.Code)
	}
}

func TestListNodes(t *testing.T) {
	server := newServer()

	// Register a few nodes
	nodes := []Node{
		{Name: "n1", Labels: map[string]string{"zone": "a", "gpu": "h100"}},
		{Name: "n2", Labels: map[string]string{"zone": "b", "gpu": "a100"}},
		{Name: "n3", Labels: map[string]string{"zone": "a", "gpu": "h100"}},
	}
	for _, n := range nodes {
		body, _ := json.Marshal(n)
		req := httptest.NewRequest(http.MethodPost, "/nodes", strings.NewReader(string(body)))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		server.ServeHTTP(rr, req)
		if rr.Code != http.StatusCreated {
			t.Fatalf("failed to register node %s: %d", n.Name, rr.Code)
		}
	}

	// Test list all nodes
	req := httptest.NewRequest(http.MethodGet, "/nodes", nil)
	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}

	var allNodes []Node
	if err := json.NewDecoder(rr.Body).Decode(&allNodes); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(allNodes) != 3 {
		t.Errorf("expected 3 nodes, got %d", len(allNodes))
	}

	// Test filter by single label
	req2 := httptest.NewRequest(http.MethodGet, "/nodes?label=zone=a", nil)
	rr2 := httptest.NewRecorder()
	server.ServeHTTP(rr2, req2)

	if rr2.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr2.Code)
	}

	var filtered []Node
	if err := json.NewDecoder(rr2.Body).Decode(&filtered); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(filtered) != 2 {
		t.Errorf("expected 2 nodes with zone=a, got %d", len(filtered))
	}
	for _, n := range filtered {
		if n.Labels["zone"] != "a" {
			t.Errorf("node %s has zone=%s, expected a", n.Name, n.Labels["zone"])
		}
	}

	// Test filter by multiple labels (AND)
	req3 := httptest.NewRequest(http.MethodGet, "/nodes?label=zone=a&label=gpu=h100", nil)
	rr3 := httptest.NewRecorder()
	server.ServeHTTP(rr3, req3)

	if rr3.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr3.Code)
	}

	var filtered2 []Node
	if err := json.NewDecoder(rr3.Body).Decode(&filtered2); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(filtered2) != 2 {
		t.Errorf("expected 2 nodes with zone=a AND gpu=h100, got %d", len(filtered2))
	}

	// Test invalid label parameter (no =)
	req4 := httptest.NewRequest(http.MethodGet, "/nodes?label=broken", nil)
	rr4 := httptest.NewRecorder()
	server.ServeHTTP(rr4, req4)

	if rr4.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid label param, got %d", rr4.Code)
	}
}

func TestGetNode(t *testing.T) {
	server := newServer()

	// Register a node
	node := Node{Name: "n1", Labels: map[string]string{"zone": "a"}}
	body, _ := json.Marshal(node)
	req := httptest.NewRequest(http.MethodPost, "/nodes", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("failed to register node: %d", rr.Code)
	}

	// Test get existing node
	req2 := httptest.NewRequest(http.MethodGet, "/nodes/n1", nil)
	rr2 := httptest.NewRecorder()
	server.ServeHTTP(rr2, req2)

	if rr2.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr2.Code)
	}

	var got Node
	if err := json.NewDecoder(rr2.Body).Decode(&got); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if got.Name != "n1" || got.Labels["zone"] != "a" {
		t.Errorf("node mismatch: %+v", got)
	}

	// Test get non-existent node
	req3 := httptest.NewRequest(http.MethodGet, "/nodes/zzz", nil)
	rr3 := httptest.NewRecorder()
	server.ServeHTTP(rr3, req3)

	if rr3.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr3.Code)
	}
}

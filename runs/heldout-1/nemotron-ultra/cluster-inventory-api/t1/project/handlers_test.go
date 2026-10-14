package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCreateNode(t *testing.T) {
	server := newServer()

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
}

func TestCreateDuplicateNode(t *testing.T) {
	server := newServer()

	body := `{"name": "n1", "labels": {}}`
	req1 := httptest.NewRequest(http.MethodPost, "/nodes", strings.NewReader(body))
	req1.Header.Set("Content-Type", "application/json")
	rr1 := httptest.NewRecorder()
	server.ServeHTTP(rr1, req1)

	if rr1.Code != http.StatusCreated {
		t.Errorf("first request expected 201, got %d", rr1.Code)
	}

	req2 := httptest.NewRequest(http.MethodPost, "/nodes", strings.NewReader(body))
	req2.Header.Set("Content-Type", "application/json")
	rr2 := httptest.NewRecorder()
	server.ServeHTTP(rr2, req2)

	if rr2.Code != http.StatusConflict {
		t.Errorf("second request expected 409, got %d", rr2.Code)
	}
}

func TestCreateNodeMissingName(t *testing.T) {
	server := newServer()

	body := `{"labels": {}}`
	req := httptest.NewRequest(http.MethodPost, "/nodes", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestListNodesWithLabelFilter(t *testing.T) {
	server := newServer()

	// Create two nodes
	nodes := []string{
		`{"name": "n1", "labels": {"zone": "a", "gpu": "h100"}}`,
		`{"name": "n2", "labels": {"zone": "b", "gpu": "a100"}}`,
	}
	for _, body := range nodes {
		req := httptest.NewRequest(http.MethodPost, "/nodes", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		server.ServeHTTP(rr, req)
		if rr.Code != http.StatusCreated {
			t.Fatalf("failed to create node: %d", rr.Code)
		}
	}

	// Filter by zone=a
	req := httptest.NewRequest(http.MethodGet, "/nodes?label=zone=a", nil)
	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}

	var result []Node
	if err := json.NewDecoder(rr.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(result) != 1 || result[0].Name != "n1" {
		t.Errorf("expected only n1, got %v", result)
	}
}

func TestListNodesInvalidLabelParam(t *testing.T) {
	server := newServer()

	req := httptest.NewRequest(http.MethodGet, "/nodes?label=broken", nil)
	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestGetNode(t *testing.T) {
	server := newServer()

	body := `{"name": "n1", "labels": {"zone": "a"}}`
	req := httptest.NewRequest(http.MethodPost, "/nodes", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("failed to create node: %d", rr.Code)
	}

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
		t.Errorf("expected name n1, got %s", node.Name)
	}
}

func TestGetUnknownNode(t *testing.T) {
	server := newServer()

	req := httptest.NewRequest(http.MethodGet, "/nodes/zzz", nil)
	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

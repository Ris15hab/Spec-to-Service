package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDiffSuccess(t *testing.T) {
	server := newServer()

	body := `{"old": {"a": "1", "b": "2"}, "new": {"a": "1", "c": "3"}}`
	req := httptest.NewRequest(http.MethodPost, "/diff", bytes.NewBufferString(body))
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp diffResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if len(resp.Added) != 1 || resp.Added["c"] != "3" {
		t.Errorf("added: got %v, want {\"c\":\"3\"}", resp.Added)
	}
	if len(resp.Removed) != 1 || resp.Removed["b"] != "2" {
		t.Errorf("removed: got %v, want {\"b\":\"2\"}", resp.Removed)
	}
	if len(resp.Changed) != 0 {
		t.Errorf("changed: got %v, want empty", resp.Changed)
	}
}

func TestDiffMissingNew(t *testing.T) {
	server := newServer()

	body := `{"old": {"a": "1"}}`
	req := httptest.NewRequest(http.MethodPost, "/diff", bytes.NewBufferString(body))
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestDiffInvalidTypes(t *testing.T) {
	server := newServer()

	body := `{"old": "x", "new": {}}`
	req := httptest.NewRequest(http.MethodPost, "/diff", bytes.NewBufferString(body))
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestDiffIdentical(t *testing.T) {
	server := newServer()

	body := `{"old": {"a": "1", "b": "2"}, "new": {"a": "1", "b": "2"}}`
	req := httptest.NewRequest(http.MethodPost, "/diff", bytes.NewBufferString(body))
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp diffResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if len(resp.Added) != 0 || len(resp.Removed) != 0 || len(resp.Changed) != 0 {
		t.Errorf("expected all empty, got added=%v removed=%v changed=%v", resp.Added, resp.Removed, resp.Changed)
	}
}

func TestDiffChangedValues(t *testing.T) {
	server := newServer()

	body := `{"old": {"a": "1", "b": "2"}, "new": {"a": "10", "b": "2"}}`
	req := httptest.NewRequest(http.MethodPost, "/diff", bytes.NewBufferString(body))
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp diffResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if len(resp.Changed) != 1 {
		t.Fatalf("expected 1 changed, got %d", len(resp.Changed))
	}
	if resp.Changed["a"].Old != "1" || resp.Changed["a"].New != "10" {
		t.Errorf("changed a: got %v, want {old:1 new:10}", resp.Changed["a"])
	}
	if len(resp.Added) != 0 || len(resp.Removed) != 0 {
		t.Errorf("unexpected added/removed: %v %v", resp.Added, resp.Removed)
	}
}

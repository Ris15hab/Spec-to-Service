package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDiffSuccess(t *testing.T) {
	server := newServer()

	body := `{"old": {"a": "1", "b": "2"}, "new": {"a": "1", "c": "3"}}`
	req := httptest.NewRequest(http.MethodPost, "/diff", strings.NewReader(body))
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp diffResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	expectedAdded := map[string]string{"c": "3"}
	expectedRemoved := map[string]string{"b": "2"}
	expectedChanged := map[string]changedValue{}

	if !mapsEqual(resp.Added, expectedAdded) {
		t.Errorf("added: got %v, want %v", resp.Added, expectedAdded)
	}
	if !mapsEqual(resp.Removed, expectedRemoved) {
		t.Errorf("removed: got %v, want %v", resp.Removed, expectedRemoved)
	}
	if !changedMapsEqual(resp.Changed, expectedChanged) {
		t.Errorf("changed: got %v, want %v", resp.Changed, expectedChanged)
	}
}

func TestDiffMissingNew(t *testing.T) {
	server := newServer()

	body := `{"old": {"a": "1"}}`
	req := httptest.NewRequest(http.MethodPost, "/diff", strings.NewReader(body))
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestDiffInvalidTypes(t *testing.T) {
	server := newServer()

	body := `{"old": "x", "new": {}}`
	req := httptest.NewRequest(http.MethodPost, "/diff", strings.NewReader(body))
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestDiffIdenticalConfigs(t *testing.T) {
	server := newServer()

	body := `{"old": {"a": "1", "b": "2"}, "new": {"a": "1", "b": "2"}}`
	req := httptest.NewRequest(http.MethodPost, "/diff", strings.NewReader(body))
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
		t.Errorf("expected empty diff, got added=%v removed=%v changed=%v", resp.Added, resp.Removed, resp.Changed)
	}
}

func mapsEqual(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}

func changedMapsEqual(a, b map[string]changedValue) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}

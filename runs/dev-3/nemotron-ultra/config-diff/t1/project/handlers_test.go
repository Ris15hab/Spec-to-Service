package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDiffSuccess(t *testing.T) {
	server := newServer()

	body := `{"old": {"a": "1", "b": "2"}, "new": {"a": "1", "c": "3"}}`
	req := httptest.NewRequest(http.MethodPost, "/diff", io.NopCloser(strings.NewReader(body)))
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	added := resp["added"].(map[string]interface{})
	if added["c"] != "3" {
		t.Errorf("expected added.c=3, got %v", added["c"])
	}

	removed := resp["removed"].(map[string]interface{})
	if removed["b"] != "2" {
		t.Errorf("expected removed.b=2, got %v", removed["b"])
	}

	changed := resp["changed"].(map[string]interface{})
	if changed == nil || len(changed) != 0 {
		t.Errorf("expected no changed keys, got %v", changed)
	}
}

func TestDiffMissingNew(t *testing.T) {
	server := newServer()

	body := `{"old": {"a": "1"}}`
	req := httptest.NewRequest(http.MethodPost, "/diff", io.NopCloser(strings.NewReader(body)))
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestDiffInvalidType(t *testing.T) {
	server := newServer()

	body := `{"old": "x", "new": {}}`
	req := httptest.NewRequest(http.MethodPost, "/diff", io.NopCloser(strings.NewReader(body)))
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestDiffIdenticalConfigs(t *testing.T) {
	server := newServer()

	body := `{"old": {"a": "1", "b": "2"}, "new": {"a": "1", "b": "2"}}`
	req := httptest.NewRequest(http.MethodPost, "/diff", io.NopCloser(strings.NewReader(body)))
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	added := resp["added"].(map[string]interface{})
	if len(added) != 0 {
		t.Errorf("expected empty added, got %v", added)
	}

	removed := resp["removed"].(map[string]interface{})
	if len(removed) != 0 {
		t.Errorf("expected empty removed, got %v", removed)
	}

	changed := resp["changed"].(map[string]interface{})
	if len(changed) != 0 {
		t.Errorf("expected empty changed, got %v", changed)
	}
}

func TestDiffChangedValues(t *testing.T) {
	server := newServer()

	body := `{"old": {"a": "1", "b": "2"}, "new": {"a": "10", "b": "2"}}`
	req := httptest.NewRequest(http.MethodPost, "/diff", io.NopCloser(strings.NewReader(body)))
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	changed := resp["changed"].(map[string]interface{})
	aChanged := changed["a"].(map[string]interface{})
	if aChanged["old"] != "1" || aChanged["new"] != "10" {
		t.Errorf("expected changed.a={old:1, new:10}, got %v", aChanged)
	}
}

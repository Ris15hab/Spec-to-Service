package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDiffSuccess(t *testing.T) {
	server := newServer()

	body := `{"old": {"a": "1", "b": "2"}, "new": {"a": "1", "c": "3"}}`
	req := httptest.NewRequest(http.MethodPost, "/diff", nil)
	req.Body = jsonBody(body)
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
		t.Errorf("added: got %v, want {c:3}", resp.Added)
	}
	if len(resp.Removed) != 1 || resp.Removed["b"] != "2" {
		t.Errorf("removed: got %v, want {b:2}", resp.Removed)
	}
	if len(resp.Changed) != 0 {
		t.Errorf("changed: got %v, want empty", resp.Changed)
	}
}

func TestDiffMissingNew(t *testing.T) {
	server := newServer()

	body := `{"old": {"a": "1"}}`
	req := httptest.NewRequest(http.MethodPost, "/diff", nil)
	req.Body = jsonBody(body)
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestDiffInvalidType(t *testing.T) {
	server := newServer()

	body := `{"old": "x", "new": {}}`
	req := httptest.NewRequest(http.MethodPost, "/diff", nil)
	req.Body = jsonBody(body)
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestDiffIdentical(t *testing.T) {
	server := newServer()

	body := `{"old": {"a": "1", "b": "2"}, "new": {"a": "1", "b": "2"}}`
	req := httptest.NewRequest(http.MethodPost, "/diff", nil)
	req.Body = jsonBody(body)
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

	body := `{"old": {"a": "1", "b": "2"}, "new": {"a": "10", "b": "20"}}`
	req := httptest.NewRequest(http.MethodPost, "/diff", nil)
	req.Body = jsonBody(body)
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp diffResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if len(resp.Added) != 0 || len(resp.Removed) != 0 {
		t.Errorf("expected no added/removed, got added=%v removed=%v", resp.Added, resp.Removed)
	}
	if len(resp.Changed) != 2 {
		t.Errorf("expected 2 changed, got %d", len(resp.Changed))
	}
	if resp.Changed["a"].Old != "1" || resp.Changed["a"].New != "10" {
		t.Errorf("a changed: got %v", resp.Changed["a"])
	}
	if resp.Changed["b"].Old != "2" || resp.Changed["b"].New != "20" {
		t.Errorf("b changed: got %v", resp.Changed["b"])
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
		return 0, nil
	}
	n = copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}

func (r *jsonBodyReader) Close() error {
	return nil
}

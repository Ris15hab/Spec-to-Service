package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCreateArtifactSuccess(t *testing.T) {
	server := newServer()

	body := `{"name": "svc", "version": "1.0.0", "sha256": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "size_bytes": 10}`
	req := httptest.NewRequest(http.MethodPost, "/artifacts", strings.NewReader(body))
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var artifact Artifact
	if err := json.NewDecoder(w.Body).Decode(&artifact); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if artifact.Name != "svc" || artifact.Version != "1.0.0" || artifact.SizeBytes != 10 {
		t.Fatalf("unexpected artifact: %+v", artifact)
	}
}

func TestCreateArtifactConflict(t *testing.T) {
	server := newServer()

	body := `{"name": "svc", "version": "1.0.0", "sha256": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "size_bytes": 10}`
	req1 := httptest.NewRequest(http.MethodPost, "/artifacts", strings.NewReader(body))
	w1 := httptest.NewRecorder()
	server.ServeHTTP(w1, req1)
	if w1.Code != http.StatusCreated {
		t.Fatalf("first request expected 201, got %d", w1.Code)
	}

	req2 := httptest.NewRequest(http.MethodPost, "/artifacts", strings.NewReader(body))
	w2 := httptest.NewRecorder()
	server.ServeHTTP(w2, req2)
	if w2.Code != http.StatusConflict {
		t.Fatalf("second request expected 409, got %d: %s", w2.Code, w2.Body.String())
	}
}

func TestCreateArtifactInvalidSHA256(t *testing.T) {
	server := newServer()

	body := `{"name": "svc", "version": "1.0.1", "sha256": "xyz", "size_bytes": 10}`
	req := httptest.NewRequest(http.MethodPost, "/artifacts", strings.NewReader(body))
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestListArtifactsByName(t *testing.T) {
	server := newServer()

	// Create two versions
	for _, v := range []string{"1.0.0", "1.0.1"} {
		body := `{"name": "svc", "version": "` + v + `", "sha256": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "size_bytes": 10}`
		req := httptest.NewRequest(http.MethodPost, "/artifacts", strings.NewReader(body))
		w := httptest.NewRecorder()
		server.ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("create failed: %d", w.Code)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/artifacts/svc", nil)
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var versions []Artifact
	if err := json.NewDecoder(w.Body).Decode(&versions); err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if len(versions) != 2 {
		t.Fatalf("expected 2 versions, got %d", len(versions))
	}
}

func TestGetArtifactByNameAndVersion(t *testing.T) {
	server := newServer()

	body := `{"name": "svc", "version": "1.0.0", "sha256": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "size_bytes": 10}`
	req := httptest.NewRequest(http.MethodPost, "/artifacts", strings.NewReader(body))
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create failed: %d", w.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/artifacts/svc/1.0.0", nil)
	w = httptest.NewRecorder()
	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var artifact Artifact
	if err := json.NewDecoder(w.Body).Decode(&artifact); err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if artifact.Version != "1.0.0" {
		t.Fatalf("wrong version: %s", artifact.Version)
	}
}

func TestGetArtifactNotFound(t *testing.T) {
	server := newServer()

	req := httptest.NewRequest(http.MethodGet, "/artifacts/nope", nil)
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for unknown name, got %d", w.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/artifacts/svc/9.9.9", nil)
	w = httptest.NewRecorder()
	server.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for unknown version, got %d", w.Code)
	}
}

func TestListAllArtifacts(t *testing.T) {
	server := newServer()

	for _, name := range []string{"svc1", "svc2"} {
		body := `{"name": "` + name + `", "version": "1.0.0", "sha256": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "size_bytes": 10}`
		req := httptest.NewRequest(http.MethodPost, "/artifacts", strings.NewReader(body))
		w := httptest.NewRecorder()
		server.ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("create failed: %d", w.Code)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/artifacts", nil)
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var artifacts []Artifact
	if err := json.NewDecoder(w.Body).Decode(&artifacts); err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if len(artifacts) != 2 {
		t.Fatalf("expected 2 artifacts, got %d", len(artifacts))
	}
}

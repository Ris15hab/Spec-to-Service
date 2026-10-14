package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreateArtifactSuccess(t *testing.T) {
	server := newServer()

	body := `{"name": "svc", "version": "1.0.0", "sha256": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "size_bytes": 10}`
	req := httptest.NewRequest(http.MethodPost, "/artifacts", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}

	var artifact Artifact
	if err := json.NewDecoder(rr.Body).Decode(&artifact); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if artifact.Name != "svc" || artifact.Version != "1.0.0" {
		t.Fatalf("unexpected artifact: %+v", artifact)
	}
}

func TestCreateArtifactConflict(t *testing.T) {
	server := newServer()

	body := `{"name": "svc", "version": "1.0.0", "sha256": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "size_bytes": 10}`
	req1 := httptest.NewRequest(http.MethodPost, "/artifacts", bytes.NewBufferString(body))
	req1.Header.Set("Content-Type", "application/json")
	rr1 := httptest.NewRecorder()
	server.ServeHTTP(rr1, req1)
	if rr1.Code != http.StatusCreated {
		t.Fatalf("first request expected 201, got %d", rr1.Code)
	}

	req2 := httptest.NewRequest(http.MethodPost, "/artifacts", bytes.NewBufferString(body))
	req2.Header.Set("Content-Type", "application/json")
	rr2 := httptest.NewRecorder()
	server.ServeHTTP(rr2, req2)

	if rr2.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", rr2.Code, rr2.Body.String())
	}
}

func TestCreateArtifactInvalidSHA256(t *testing.T) {
	server := newServer()

	body := `{"name": "svc", "version": "1.0.1", "sha256": "xyz", "size_bytes": 10}`
	req := httptest.NewRequest(http.MethodPost, "/artifacts", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestGetArtifactsByName(t *testing.T) {
	server := newServer()

	// Create two versions
	for _, version := range []string{"1.0.0", "1.0.1"} {
		body := `{"name": "svc", "version": "` + version + `", "sha256": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "size_bytes": 10}`
		req := httptest.NewRequest(http.MethodPost, "/artifacts", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		server.ServeHTTP(rr, req)
		if rr.Code != http.StatusCreated {
			t.Fatalf("create failed: %d", rr.Code)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/artifacts/svc", nil)
	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var artifacts []Artifact
	if err := json.NewDecoder(rr.Body).Decode(&artifacts); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if len(artifacts) != 2 {
		t.Fatalf("expected 2 artifacts, got %d", len(artifacts))
	}
}

func TestGetArtifactByNameVersion(t *testing.T) {
	server := newServer()

	body := `{"name": "svc", "version": "1.0.0", "sha256": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "size_bytes": 10}`
	req := httptest.NewRequest(http.MethodPost, "/artifacts", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("create failed: %d", rr.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/artifacts/svc/1.0.0", nil)
	rr = httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var artifact Artifact
	if err := json.NewDecoder(rr.Body).Decode(&artifact); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if artifact.Name != "svc" || artifact.Version != "1.0.0" {
		t.Fatalf("unexpected artifact: %+v", artifact)
	}
}

func TestGetArtifactNotFound(t *testing.T) {
	server := newServer()

	req := httptest.NewRequest(http.MethodGet, "/artifacts/nope", nil)
	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestGetArtifactVersionNotFound(t *testing.T) {
	server := newServer()

	body := `{"name": "svc", "version": "1.0.0", "sha256": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "size_bytes": 10}`
	req := httptest.NewRequest(http.MethodPost, "/artifacts", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("create failed: %d", rr.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/artifacts/svc/9.9.9", nil)
	rr = httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rr.Code, rr.Body.String())
	}
}

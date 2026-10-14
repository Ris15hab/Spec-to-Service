package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreateAndGetArtifact(t *testing.T) {
	server := newServer()

	// Create artifact
	body := `{"name": "svc", "version": "1.0.0", "sha256": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "size_bytes": 10}`
	req := httptest.NewRequest(http.MethodPost, "/artifacts", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}

	var created Artifact
	if err := json.NewDecoder(rr.Body).Decode(&created); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if created.Name != "svc" || created.Version != "1.0.0" {
		t.Fatalf("unexpected artifact: %+v", created)
	}

	// Get by name and version
	req = httptest.NewRequest(http.MethodGet, "/artifacts/svc/1.0.0", nil)
	rr = httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var got Artifact
	if err := json.NewDecoder(rr.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.Name != "svc" || got.Version != "1.0.0" {
		t.Fatalf("unexpected artifact: %+v", got)
	}
}

func TestDuplicateArtifactReturns409(t *testing.T) {
	server := newServer()

	body := `{"name": "svc", "version": "1.0.0", "sha256": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "size_bytes": 10}`

	// First create
	req := httptest.NewRequest(http.MethodPost, "/artifacts", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("first create: expected 201, got %d", rr.Code)
	}

	// Duplicate create
	req = httptest.NewRequest(http.MethodPost, "/artifacts", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	if rr.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestInvalidArtifactReturns400(t *testing.T) {
	server := newServer()

	tests := []struct {
		name string
		body string
	}{
		{"invalid sha256", `{"name": "svc", "version": "1.0.1", "sha256": "xyz", "size_bytes": 10}`},
		{"missing name", `{"version": "1.0.0", "sha256": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "size_bytes": 10}`},
		{"missing version", `{"name": "svc", "sha256": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "size_bytes": 10}`},
		{"missing sha256", `{"name": "svc", "version": "1.0.0", "size_bytes": 10}`},
		{"missing size_bytes", `{"name": "svc", "version": "1.0.0", "sha256": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}`},
		{"negative size", `{"name": "svc", "version": "1.0.0", "sha256": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "size_bytes": -1}`},
		{"malformed JSON", `{invalid json`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/artifacts", bytes.NewBufferString(tc.body))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()
			server.ServeHTTP(rr, req)

			if rr.Code != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d: %s", rr.Code, rr.Body.String())
			}
		})
	}
}

func TestListArtifacts(t *testing.T) {
	server := newServer()

	// Create a couple artifacts
	for _, v := range []string{"1.0.0", "1.0.1"} {
		body := `{"name": "svc", "version": "` + v + `", "sha256": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "size_bytes": 10}`
		req := httptest.NewRequest(http.MethodPost, "/artifacts", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		server.ServeHTTP(rr, req)
		if rr.Code != http.StatusCreated {
			t.Fatalf("create %s: expected 201, got %d", v, rr.Code)
		}
	}

	// List all
	req := httptest.NewRequest(http.MethodGet, "/artifacts", nil)
	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var artifacts []Artifact
	if err := json.NewDecoder(rr.Body).Decode(&artifacts); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(artifacts) != 2 {
		t.Fatalf("expected 2 artifacts, got %d", len(artifacts))
	}
}

func TestGetUnknownArtifactReturns404(t *testing.T) {
	server := newServer()

	// Unknown name
	req := httptest.NewRequest(http.MethodGet, "/artifacts/nope", nil)
	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for unknown name, got %d", rr.Code)
	}

	// Known name, unknown version
	body := `{"name": "svc", "version": "1.0.0", "sha256": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "size_bytes": 10}`
	req = httptest.NewRequest(http.MethodPost, "/artifacts", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	server.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d", rr.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/artifacts/svc/9.9.9", nil)
	rr = httptest.NewRecorder()
	server.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for unknown version, got %d", rr.Code)
	}
}

func TestListVersions(t *testing.T) {
	server := newServer()

	// Create multiple versions
	for _, v := range []string{"1.0.0", "1.0.1", "2.0.0"} {
		body := `{"name": "svc", "version": "` + v + `", "sha256": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "size_bytes": 10}`
		req := httptest.NewRequest(http.MethodPost, "/artifacts", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		server.ServeHTTP(rr, req)
		if rr.Code != http.StatusCreated {
			t.Fatalf("create %s: expected 201, got %d", v, rr.Code)
		}
	}

	// List versions
	req := httptest.NewRequest(http.MethodGet, "/artifacts/svc", nil)
	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var versions []Artifact
	if err := json.NewDecoder(rr.Body).Decode(&versions); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(versions) != 3 {
		t.Fatalf("expected 3 versions, got %d", len(versions))
	}
}

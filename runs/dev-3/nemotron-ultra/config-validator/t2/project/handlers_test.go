package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestValidateValidConfig(t *testing.T) {
	server := newServer()

	body := `{"name": "api", "replicas": 3, "port": 8080}`
	req := httptest.NewRequest(http.MethodPost, "/validate", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	server.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var resp ValidationResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if !resp.Valid {
		t.Errorf("expected valid=true, got false")
	}
	if len(resp.Errors) != 0 {
		t.Errorf("expected empty errors, got %v", resp.Errors)
	}
}

func TestValidateInvalidReplicas(t *testing.T) {
	server := newServer()

	body := `{"name": "api", "replicas": 0, "port": 8080}`
	req := httptest.NewRequest(http.MethodPost, "/validate", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	server.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected status 422, got %d", rr.Code)
	}

	var resp ValidationResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Valid {
		t.Errorf("expected valid=false, got true")
	}
	if len(resp.Errors) != 1 {
		t.Errorf("expected 1 error, got %d: %v", len(resp.Errors), resp.Errors)
	}
	if resp.Errors[0] != "replicas must be between 1 and 100" {
		t.Errorf("unexpected error: %s", resp.Errors[0])
	}
}

func TestValidateMultipleErrors(t *testing.T) {
	server := newServer()

	body := `{"name": "", "replicas": 500, "port": 0}`
	req := httptest.NewRequest(http.MethodPost, "/validate", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	server.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected status 422, got %d", rr.Code)
	}

	var resp ValidationResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Valid {
		t.Errorf("expected valid=false, got true")
	}
	if len(resp.Errors) != 3 {
		t.Errorf("expected 3 errors, got %d: %v", len(resp.Errors), resp.Errors)
	}
}

func TestValidateMalformedJSON(t *testing.T) {
	server := newServer()

	body := `not json`
	req := httptest.NewRequest(http.MethodPost, "/validate", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	server.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
}

func TestHealthEndpoint(t *testing.T) {
	server := newServer()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()

	server.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}
}

func TestMetricsEndpoint(t *testing.T) {
	server := newServer()

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rr := httptest.NewRecorder()

	server.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}
}

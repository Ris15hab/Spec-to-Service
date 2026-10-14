package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPostJobsMissingIDReturns400(t *testing.T) {
	server := newServer()

	body := `{"gpus": 1}`
	req := httptest.NewRequest(http.MethodPost, "/jobs", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
}

func TestPostJobsSuccessReturns201(t *testing.T) {
	server := newServer()

	body := `{"id": "j1", "gpus": 1}`
	req := httptest.NewRequest(http.MethodPost, "/jobs", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated && rr.Code != http.StatusOK {
		t.Errorf("expected status 201 or 200, got %d", rr.Code)
	}

	var job Job
	if err := json.NewDecoder(rr.Body).Decode(&job); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if job.ID != "j1" {
		t.Errorf("expected job ID j1, got %s", job.ID)
	}
	if job.GPUs != 1 {
		t.Errorf("expected gpus 1, got %d", job.GPUs)
	}
	if job.Status != "queued" {
		t.Errorf("expected status queued, got %s", job.Status)
	}
}

func TestGetJobsReturnsList(t *testing.T) {
	server := newServer()

	// First, add a job
	postBody := `{"id": "j1", "gpus": 2}`
	postReq := httptest.NewRequest(http.MethodPost, "/jobs", bytes.NewBufferString(postBody))
	postReq.Header.Set("Content-Type", "application/json")
	postRR := httptest.NewRecorder()
	server.ServeHTTP(postRR, postReq)

	if postRR.Code != http.StatusCreated && postRR.Code != http.StatusOK {
		t.Fatalf("failed to create job: status %d", postRR.Code)
	}

	// Now list jobs
	getReq := httptest.NewRequest(http.MethodGet, "/jobs", nil)
	getRR := httptest.NewRecorder()
	server.ServeHTTP(getRR, getReq)

	if getRR.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", getRR.Code)
	}

	var jobs []*Job
	if err := json.NewDecoder(getRR.Body).Decode(&jobs); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(jobs) != 1 {
		t.Errorf("expected 1 job, got %d", len(jobs))
	}
	if jobs[0].ID != "j1" {
		t.Errorf("expected job ID j1, got %s", jobs[0].ID)
	}
}

func TestPostJobsInvalidJSONReturns400(t *testing.T) {
	server := newServer()

	body := `{"id": "j1", "gpus: 1}`
	req := httptest.NewRequest(http.MethodPost, "/jobs", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
}

func TestPostJobsZeroGPUsReturns400(t *testing.T) {
	server := newServer()

	body := `{"id": "j1", "gpus": 0}`
	req := httptest.NewRequest(http.MethodPost, "/jobs", bytes.NewBufferString(body))
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

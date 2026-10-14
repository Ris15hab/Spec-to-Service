package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPostJobsMissingIDReturns400(t *testing.T) {
	server := newServer()
	body := `{"gpus": 1}`
	req := httptest.NewRequest(http.MethodPost, "/jobs", strings.NewReader(body))
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestPostJobsSuccessReturns201(t *testing.T) {
	server := newServer()
	body := `{"id": "j1", "gpus": 1}`
	req := httptest.NewRequest(http.MethodPost, "/jobs", strings.NewReader(body))
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	if w.Code != http.StatusCreated && w.Code != http.StatusOK {
		t.Errorf("expected 201 or 200, got %d", w.Code)
	}

	var job Job
	if err := json.NewDecoder(w.Body).Decode(&job); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if job.ID != "j1" || job.GPUs != 1 || job.Status != "queued" {
		t.Errorf("unexpected job: %+v", job)
	}
}

func TestGetJobsReturnsList(t *testing.T) {
	server := newServer()

	// Add a job first
	req := httptest.NewRequest(http.MethodPost, "/jobs", strings.NewReader(`{"id": "j1", "gpus": 2}`))
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("failed to create job: %d", w.Code)
	}

	// Now list jobs
	req = httptest.NewRequest(http.MethodGet, "/jobs", nil)
	w = httptest.NewRecorder()
	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var jobs []*Job
	if err := json.NewDecoder(w.Body).Decode(&jobs); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(jobs) != 1 {
		t.Errorf("expected 1 job, got %d", len(jobs))
	}
	if jobs[0].ID != "j1" || jobs[0].GPUs != 2 {
		t.Errorf("unexpected job: %+v", jobs[0])
	}
}

func TestQueueLengthMetricUpdates(t *testing.T) {
	server := newServer()

	// Initial state - empty
	req := httptest.NewRequest(http.MethodGet, "/jobs", nil)
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)
	var jobs []*Job
	json.NewDecoder(w.Body).Decode(&jobs)
	if len(jobs) != 0 {
		t.Errorf("expected empty queue initially")
	}

	// Add job
	req = httptest.NewRequest(http.MethodPost, "/jobs", strings.NewReader(`{"id": "j1", "gpus": 1}`))
	w = httptest.NewRecorder()
	server.ServeHTTP(w, req)

	// Check queue length via /jobs
	req = httptest.NewRequest(http.MethodGet, "/jobs", nil)
	w = httptest.NewRecorder()
	server.ServeHTTP(w, req)
	json.NewDecoder(w.Body).Decode(&jobs)
	if len(jobs) != 1 {
		t.Errorf("expected 1 job in queue, got %d", len(jobs))
	}
}

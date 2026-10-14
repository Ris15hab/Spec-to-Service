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

func TestPostJobsSuccess(t *testing.T) {
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
	if job.ID != "j1" {
		t.Errorf("expected id j1, got %s", job.ID)
	}
	if job.GPUs != 1 {
		t.Errorf("expected gpus 1, got %d", job.GPUs)
	}
	if job.Status != "queued" {
		t.Errorf("expected status queued, got %s", job.Status)
	}
}

func TestGetJobsList(t *testing.T) {
	server := newServer()

	body := `{"id": "j1", "gpus": 2}`
	req := httptest.NewRequest(http.MethodPost, "/jobs", strings.NewReader(body))
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("failed to create job: %d", w.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/jobs", nil)
	w = httptest.NewRecorder()
	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var jobs []Job
	if err := json.NewDecoder(w.Body).Decode(&jobs); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(jobs) != 1 {
		t.Errorf("expected 1 job, got %d", len(jobs))
	}
	if jobs[0].ID != "j1" {
		t.Errorf("expected job id j1, got %s", jobs[0].ID)
	}
}

func TestQueueLengthMetricUpdates(t *testing.T) {
	server := newServer()

	body := `{"id": "j1", "gpus": 1}`
	req := httptest.NewRequest(http.MethodPost, "/jobs", strings.NewReader(body))
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("failed to create job: %d", w.Code)
	}

	body = `{"id": "j2", "gpus": 2}`
	req = httptest.NewRequest(http.MethodPost, "/jobs", strings.NewReader(body))
	w = httptest.NewRecorder()
	server.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("failed to create job: %d", w.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/jobs", nil)
	w = httptest.NewRecorder()
	server.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	var jobs []Job
	json.NewDecoder(w.Body).Decode(&jobs)
	if len(jobs) != 2 {
		t.Errorf("expected 2 jobs, got %d", len(jobs))
	}
}

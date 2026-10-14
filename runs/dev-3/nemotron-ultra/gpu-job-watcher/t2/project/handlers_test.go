package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPostJobsMissingIDReturns400(t *testing.T) {
	server := newServer()
	req := httptest.NewRequest(http.MethodPost, "/jobs", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Body = jsonBody(`{"gpus": 1}`)
	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestPostJobsSuccessReturns201(t *testing.T) {
	server := newServer()
	req := httptest.NewRequest(http.MethodPost, "/jobs", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Body = jsonBody(`{"id": "j1", "gpus": 1}`)
	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d", rr.Code)
	}

	var job Job
	if err := json.NewDecoder(rr.Body).Decode(&job); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if job.ID != "j1" || job.GPUs != 1 || job.Status != "queued" {
		t.Errorf("unexpected job: %+v", job)
	}
}

func TestGetJobsReturnsList(t *testing.T) {
	server := newServer()

	// Add a job first
	req := httptest.NewRequest(http.MethodPost, "/jobs", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Body = jsonBody(`{"id": "j1", "gpus": 2}`)
	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("failed to create job: %d", rr.Code)
	}

	// List jobs
	req = httptest.NewRequest(http.MethodGet, "/jobs", nil)
	rr = httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}

	var jobs []*Job
	if err := json.NewDecoder(rr.Body).Decode(&jobs); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(jobs) != 1 {
		t.Errorf("expected 1 job, got %d", len(jobs))
	}
	if jobs[0].ID != "j1" || jobs[0].GPUs != 2 {
		t.Errorf("unexpected job: %+v", jobs[0])
	}
}

func jsonBody(s string) *stringsReader {
	return &stringsReader{s: s}
}

type stringsReader struct {
	s string
	i int
}

func (r *stringsReader) Read(p []byte) (n int, err error) {
	if r.i >= len(r.s) {
		return 0, io.EOF
	}
	n = copy(p, r.s[r.i:])
	r.i += n
	return n, nil
}

func (r *stringsReader) Close() error {
	return nil
}

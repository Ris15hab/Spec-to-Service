package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPostJobsMissingID(t *testing.T) {
	ResetStore()
	server := newServer()

	body := `{"gpus": 1}`
	req := httptest.NewRequest(http.MethodPost, "/jobs", jsonBody(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
}

func TestPostJobsSuccess(t *testing.T) {
	ResetStore()
	server := newServer()

	body := `{"id": "j1", "gpus": 1}`
	req := httptest.NewRequest(http.MethodPost, "/jobs", jsonBody(body))
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

	if job.ID != "j1" || job.GPUs != 1 || job.Status != "queued" {
		t.Errorf("unexpected job: %+v", job)
	}
}

func TestGetJobs(t *testing.T) {
	ResetStore()
	server := newServer()

	// First, add a job
	body := `{"id": "j1", "gpus": 2}`
	req := httptest.NewRequest(http.MethodPost, "/jobs", jsonBody(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated && rr.Code != http.StatusOK {
		t.Fatalf("failed to create job: %d", rr.Code)
	}

	// Now list jobs
	req = httptest.NewRequest(http.MethodGet, "/jobs", nil)
	rr = httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var jobs []*Job
	if err := json.NewDecoder(rr.Body).Decode(&jobs); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(jobs) != 1 {
		t.Errorf("expected 1 job, got %d", len(jobs))
	}

	if jobs[0].ID != "j1" || jobs[0].GPUs != 2 || jobs[0].Status != "queued" {
		t.Errorf("unexpected job: %+v", jobs[0])
	}
}

func TestPostJobsDuplicateID(t *testing.T) {
	ResetStore()
	server := newServer()

	body := `{"id": "j1", "gpus": 1}`
	req := httptest.NewRequest(http.MethodPost, "/jobs", jsonBody(body))
	req.Header.Set("Content-Type", "application/json")

	// First request
	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated && rr.Code != http.StatusOK {
		t.Fatalf("first request failed: %d", rr.Code)
	}

	// Second request with same ID
	rr = httptest.NewRecorder()
	server.ServeHTTP(rr, req)
	if rr.Code != http.StatusConflict {
		t.Errorf("expected status 409 for duplicate, got %d", rr.Code)
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
		return 0, io.EOF
	}
	n = copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}

func (r *jsonBodyReader) Close() error {
	return nil
}

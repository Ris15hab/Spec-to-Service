package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestEnqueueDequeueAck(t *testing.T) {
	srv := newServer()

	// Enqueue a job
	req := httptest.NewRequest(http.MethodPost, "/jobs", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Body = jsonBody(`{"id": "job1", "payload": {"data": "test"}}`)
	rr := httptest.NewRecorder()
	srv.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Errorf("enqueue: expected 201, got %d", rr.Code)
	}

	// Dequeue the job
	req = httptest.NewRequest(http.MethodPost, "/jobs/dequeue", nil)
	rr = httptest.NewRecorder()
	srv.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("dequeue: expected 200, got %d", rr.Code)
	}
	var job map[string]any
	json.NewDecoder(rr.Body).Decode(&job)
	if job["id"] != "job1" {
		t.Errorf("dequeue: expected id job1, got %v", job["id"])
	}

	// Acknowledge the job
	req = httptest.NewRequest(http.MethodPost, "/jobs/job1/ack", nil)
	rr = httptest.NewRecorder()
	srv.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("ack: expected 200, got %d", rr.Code)
	}
}

func TestEnqueueDuplicateID(t *testing.T) {
	srv := newServer()

	body := `{"id": "dup", "payload": 1}`
	req := httptest.NewRequest(http.MethodPost, "/jobs", jsonBody(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Errorf("first enqueue: expected 201, got %d", rr.Code)
	}

	rr = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/jobs", jsonBody(body))
	req.Header.Set("Content-Type", "application/json")
	srv.ServeHTTP(rr, req)
	if rr.Code != http.StatusConflict {
		t.Errorf("duplicate enqueue: expected 409, got %d", rr.Code)
	}
}

func TestEnqueueMissingID(t *testing.T) {
	srv := newServer()

	req := httptest.NewRequest(http.MethodPost, "/jobs", jsonBody(`{"payload": 1}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("missing id: expected 400, got %d", rr.Code)
	}
}

func TestDequeueEmptyQueue(t *testing.T) {
	srv := newServer()

	req := httptest.NewRequest(http.MethodPost, "/jobs/dequeue", nil)
	rr := httptest.NewRecorder()
	srv.ServeHTTP(rr, req)
	if rr.Code != http.StatusNoContent {
		t.Errorf("empty dequeue: expected 204, got %d", rr.Code)
	}
}

func TestAckUnknownJob(t *testing.T) {
	srv := newServer()

	req := httptest.NewRequest(http.MethodPost, "/jobs/unknown/ack", nil)
	rr := httptest.NewRecorder()
	srv.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Errorf("ack unknown: expected 404, got %d", rr.Code)
	}
}

func TestAckNotInFlight(t *testing.T) {
	srv := newServer()

	// Enqueue but don't dequeue
	req := httptest.NewRequest(http.MethodPost, "/jobs", jsonBody(`{"id": "j1", "payload": 1}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("enqueue failed: %d", rr.Code)
	}

	// Try to ack without dequeuing
	req = httptest.NewRequest(http.MethodPost, "/jobs/j1/ack", nil)
	rr = httptest.NewRecorder()
	srv.ServeHTTP(rr, req)
	if rr.Code != http.StatusConflict {
		t.Errorf("ack not in-flight: expected 409, got %d", rr.Code)
	}
}

func TestFIFOOrder(t *testing.T) {
	srv := newServer()

	// Enqueue three jobs
	for _, id := range []string{"first", "second", "third"} {
		req := httptest.NewRequest(http.MethodPost, "/jobs", jsonBody(`{"id": "`+id+`", "payload": 1}`))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		srv.ServeHTTP(rr, req)
		if rr.Code != http.StatusCreated {
			t.Fatalf("enqueue %s: %d", id, rr.Code)
		}
	}

	// Dequeue and verify order
	expected := []string{"first", "second", "third"}
	for _, exp := range expected {
		req := httptest.NewRequest(http.MethodPost, "/jobs/dequeue", nil)
		rr := httptest.NewRecorder()
		srv.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("dequeue: expected 200, got %d", rr.Code)
		}
		var job map[string]any
		json.NewDecoder(rr.Body).Decode(&job)
		if job["id"] != exp {
			t.Errorf("expected %s, got %v", exp, job["id"])
		}
	}
}

func jsonBody(s string) io.ReadCloser {
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

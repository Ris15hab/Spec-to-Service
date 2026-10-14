package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestEnqueueDequeueAck(t *testing.T) {
	server := newServer()

	// Enqueue job
	req := httptest.NewRequest(http.MethodPost, "/jobs", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Body = jsonBody(`{"id": "a", "payload": 1}`)
	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("enqueue: expected 201, got %d", rr.Code)
	}

	// Enqueue duplicate should fail
	req = httptest.NewRequest(http.MethodPost, "/jobs", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Body = jsonBody(`{"id": "a", "payload": 1}`)
	rr = httptest.NewRecorder()
	server.ServeHTTP(rr, req)
	if rr.Code != http.StatusConflict {
		t.Fatalf("duplicate enqueue: expected 409, got %d", rr.Code)
	}

	// Missing id should fail
	req = httptest.NewRequest(http.MethodPost, "/jobs", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Body = jsonBody(`{"payload": 1}`)
	rr = httptest.NewRecorder()
	server.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("missing id: expected 400, got %d", rr.Code)
	}

	// Dequeue should return the job
	req = httptest.NewRequest(http.MethodPost, "/jobs/dequeue", nil)
	rr = httptest.NewRecorder()
	server.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("dequeue: expected 200, got %d", rr.Code)
	}
	var job Job
	if err := json.NewDecoder(rr.Body).Decode(&job); err != nil {
		t.Fatalf("decode dequeue response: %v", err)
	}
	if job.ID != "a" {
		t.Fatalf("unexpected job id: %s", job.ID)
	}
	// JSON numbers decode as float64
	if job.Payload != 1.0 {
		t.Fatalf("unexpected job payload: %v (type %T)", job.Payload, job.Payload)
	}

	// Ack should succeed
	req = httptest.NewRequest(http.MethodPost, "/jobs/a/ack", nil)
	rr = httptest.NewRecorder()
	server.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("ack: expected 200, got %d", rr.Code)
	}

	// Ack unknown should 404
	req = httptest.NewRequest(http.MethodPost, "/jobs/zzz/ack", nil)
	rr = httptest.NewRecorder()
	server.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("ack unknown: expected 404, got %d", rr.Code)
	}
}

func TestDequeueEmptyQueue(t *testing.T) {
	server := newServer()

	req := httptest.NewRequest(http.MethodPost, "/jobs/dequeue", nil)
	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("dequeue empty: expected 204, got %d", rr.Code)
	}
}

func TestAckNotInFlight(t *testing.T) {
	server := newServer()

	// Enqueue but don't dequeue
	req := httptest.NewRequest(http.MethodPost, "/jobs", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Body = jsonBody(`{"id": "b", "payload": 2}`)
	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("enqueue: expected 201, got %d", rr.Code)
	}

	// Ack without dequeue should 409
	req = httptest.NewRequest(http.MethodPost, "/jobs/b/ack", nil)
	rr = httptest.NewRecorder()
	server.ServeHTTP(rr, req)
	if rr.Code != http.StatusConflict {
		t.Fatalf("ack not in-flight: expected 409, got %d", rr.Code)
	}
}

func TestFIFOOrder(t *testing.T) {
	server := newServer()

	// Enqueue multiple jobs
	for i := 1; i <= 3; i++ {
		req := httptest.NewRequest(http.MethodPost, "/jobs", nil)
		req.Header.Set("Content-Type", "application/json")
		req.Body = jsonBody(`{"id": "` + string(rune('a'+i-1)) + `", "payload": ` + string(rune('0'+i)) + `}`)
		rr := httptest.NewRecorder()
		server.ServeHTTP(rr, req)
		if rr.Code != http.StatusCreated {
			t.Fatalf("enqueue %d: expected 201, got %d", i, rr.Code)
		}
	}

	// Dequeue should return in FIFO order
	expected := []string{"a", "b", "c"}
	for _, expID := range expected {
		req := httptest.NewRequest(http.MethodPost, "/jobs/dequeue", nil)
		rr := httptest.NewRecorder()
		server.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("dequeue %s: expected 200, got %d", expID, rr.Code)
		}
		var job Job
		json.NewDecoder(rr.Body).Decode(&job)
		if job.ID != expID {
			t.Fatalf("expected id %s, got %s", expID, job.ID)
		}
	}
}

func jsonBody(s string) *jsonBodyReader {
	return &jsonBodyReader{s: s}
}

type jsonBodyReader struct {
	s string
	i int
}

func (r *jsonBodyReader) Read(p []byte) (n int, err error) {
	if r.i >= len(r.s) {
		return 0, nil
	}
	n = copy(p, r.s[r.i:])
	r.i += n
	return n, nil
}

func (r *jsonBodyReader) Close() error { return nil }

package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestEnqueueDequeueAck(t *testing.T) {
	srv := newServer()

	// Enqueue job
	req := httptest.NewRequest(http.MethodPost, "/jobs", bytes.NewBufferString(`{"id": "a", "payload": 1}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("enqueue: expected 201, got %d", rr.Code)
	}

	// Enqueue duplicate should fail
	req = httptest.NewRequest(http.MethodPost, "/jobs", bytes.NewBufferString(`{"id": "a", "payload": 1}`))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	srv.ServeHTTP(rr, req)
	if rr.Code != http.StatusConflict {
		t.Fatalf("duplicate enqueue: expected 409, got %d", rr.Code)
	}

	// Enqueue without id should fail
	req = httptest.NewRequest(http.MethodPost, "/jobs", bytes.NewBufferString(`{"payload": 1}`))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	srv.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("missing id: expected 400, got %d", rr.Code)
	}

	// Dequeue should return the job
	req = httptest.NewRequest(http.MethodPost, "/jobs/dequeue", nil)
	rr = httptest.NewRecorder()
	srv.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("dequeue: expected 200, got %d", rr.Code)
	}
	var j job
	if err := json.NewDecoder(rr.Body).Decode(&j); err != nil {
		t.Fatalf("decode dequeue response: %v", err)
	}
	if j.ID != "a" {
		t.Fatalf("dequeue: expected id 'a', got %q", j.ID)
	}

	// Ack should succeed
	req = httptest.NewRequest(http.MethodPost, "/jobs/a/ack", nil)
	rr = httptest.NewRecorder()
	srv.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("ack: expected 200, got %d", rr.Code)
	}

	// Ack unknown job should 404
	req = httptest.NewRequest(http.MethodPost, "/jobs/zzz/ack", nil)
	rr = httptest.NewRecorder()
	srv.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("ack unknown: expected 404, got %d", rr.Code)
	}
}

func TestDequeueEmptyQueue(t *testing.T) {
	srv := newServer()

	req := httptest.NewRequest(http.MethodPost, "/jobs/dequeue", nil)
	rr := httptest.NewRecorder()
	srv.ServeHTTP(rr, req)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("dequeue empty: expected 204, got %d", rr.Code)
	}
}

func TestAckNotInFlight(t *testing.T) {
	srv := newServer()

	// Enqueue a job
	req := httptest.NewRequest(http.MethodPost, "/jobs", bytes.NewBufferString(`{"id": "b", "payload": 2}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("enqueue: expected 201, got %d", rr.Code)
	}

	// Ack without dequeuing should 409
	req = httptest.NewRequest(http.MethodPost, "/jobs/b/ack", nil)
	rr = httptest.NewRecorder()
	srv.ServeHTTP(rr, req)
	if rr.Code != http.StatusConflict {
		t.Fatalf("ack not in-flight: expected 409, got %d", rr.Code)
	}
}

func TestFIFOOrder(t *testing.T) {
	srv := newServer()

	// Enqueue multiple jobs
	for _, id := range []string{"first", "second", "third"} {
		req := httptest.NewRequest(http.MethodPost, "/jobs", bytes.NewBufferString(`{"id": "`+id+`", "payload": 1}`))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		srv.ServeHTTP(rr, req)
		if rr.Code != http.StatusCreated {
			t.Fatalf("enqueue %s: expected 201, got %d", id, rr.Code)
		}
	}

	// Dequeue should return in FIFO order
	expected := []string{"first", "second", "third"}
	for _, expID := range expected {
		req := httptest.NewRequest(http.MethodPost, "/jobs/dequeue", nil)
		rr := httptest.NewRecorder()
		srv.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("dequeue: expected 200, got %d", rr.Code)
		}
		var j job
		if err := json.NewDecoder(rr.Body).Decode(&j); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if j.ID != expID {
			t.Fatalf("expected %q, got %q", expID, j.ID)
		}

		// Ack it
		req = httptest.NewRequest(http.MethodPost, "/jobs/"+expID+"/ack", nil)
		rr = httptest.NewRecorder()
		srv.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("ack %s: expected 200, got %d", expID, rr.Code)
		}
	}

	// Queue should now be empty
	req := httptest.NewRequest(http.MethodPost, "/jobs/dequeue", nil)
	rr := httptest.NewRecorder()
	srv.ServeHTTP(rr, req)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("dequeue empty: expected 204, got %d", rr.Code)
	}
}

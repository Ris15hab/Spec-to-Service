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

	// Enqueue a job
	reqBody := `{"id": "job1", "payload": {"data": "test"}}`
	resp := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/jobs", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	srv.ServeHTTP(resp, req)
	if resp.Code != http.StatusCreated {
		t.Fatalf("enqueue: expected 201, got %d", resp.Code)
	}

	// Enqueue another job
	reqBody2 := `{"id": "job2", "payload": 42}`
	resp2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodPost, "/jobs", bytes.NewBufferString(reqBody2))
	req2.Header.Set("Content-Type", "application/json")
	srv.ServeHTTP(resp2, req2)
	if resp2.Code != http.StatusCreated {
		t.Fatalf("enqueue2: expected 201, got %d", resp2.Code)
	}

	// Dequeue first job (FIFO)
	resp3 := httptest.NewRecorder()
	req3 := httptest.NewRequest(http.MethodPost, "/jobs/dequeue", nil)
	srv.ServeHTTP(resp3, req3)
	if resp3.Code != http.StatusOK {
		t.Fatalf("dequeue: expected 200, got %d", resp3.Code)
	}
	var job1 job
	if err := json.NewDecoder(resp3.Body).Decode(&job1); err != nil {
		t.Fatalf("dequeue: invalid json: %v", err)
	}
	if job1.ID != "job1" {
		t.Fatalf("dequeue: expected job1, got %s", job1.ID)
	}

	// Acknowledge the dequeued job
	resp4 := httptest.NewRecorder()
	req4 := httptest.NewRequest(http.MethodPost, "/jobs/job1/ack", nil)
	srv.ServeHTTP(resp4, req4)
	if resp4.Code != http.StatusOK {
		t.Fatalf("ack: expected 200, got %d", resp4.Code)
	}

	// Dequeue second job
	resp5 := httptest.NewRecorder()
	req5 := httptest.NewRequest(http.MethodPost, "/jobs/dequeue", nil)
	srv.ServeHTTP(resp5, req5)
	if resp5.Code != http.StatusOK {
		t.Fatalf("dequeue2: expected 200, got %d", resp5.Code)
	}
	var job2 job
	if err := json.NewDecoder(resp5.Body).Decode(&job2); err != nil {
		t.Fatalf("dequeue2: invalid json: %v", err)
	}
	if job2.ID != "job2" {
		t.Fatalf("dequeue2: expected job2, got %s", job2.ID)
	}

	// Acknowledge second job
	resp6 := httptest.NewRecorder()
	req6 := httptest.NewRequest(http.MethodPost, "/jobs/job2/ack", nil)
	srv.ServeHTTP(resp6, req6)
	if resp6.Code != http.StatusOK {
		t.Fatalf("ack2: expected 200, got %d", resp6.Code)
	}

	// Queue should be empty now
	resp7 := httptest.NewRecorder()
	req7 := httptest.NewRequest(http.MethodPost, "/jobs/dequeue", nil)
	srv.ServeHTTP(resp7, req7)
	if resp7.Code != http.StatusNoContent {
		t.Fatalf("dequeue empty: expected 204, got %d", resp7.Code)
	}
}

func TestEnqueueErrors(t *testing.T) {
	srv := newServer()

	// Missing ID
	resp := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/jobs", bytes.NewBufferString(`{"payload": 1}`))
	req.Header.Set("Content-Type", "application/json")
	srv.ServeHTTP(resp, req)
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("missing id: expected 400, got %d", resp.Code)
	}

	// Valid enqueue
	resp2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodPost, "/jobs", bytes.NewBufferString(`{"id": "a", "payload": 1}`))
	req2.Header.Set("Content-Type", "application/json")
	srv.ServeHTTP(resp2, req2)
	if resp2.Code != http.StatusCreated {
		t.Fatalf("valid enqueue: expected 201, got %d", resp2.Code)
	}

	// Duplicate ID
	resp3 := httptest.NewRecorder()
	req3 := httptest.NewRequest(http.MethodPost, "/jobs", bytes.NewBufferString(`{"id": "a", "payload": 1}`))
	req3.Header.Set("Content-Type", "application/json")
	srv.ServeHTTP(resp3, req3)
	if resp3.Code != http.StatusConflict {
		t.Fatalf("duplicate id: expected 409, got %d", resp3.Code)
	}

	// Invalid JSON
	resp4 := httptest.NewRecorder()
	req4 := httptest.NewRequest(http.MethodPost, "/jobs", bytes.NewBufferString(`{invalid}`))
	req4.Header.Set("Content-Type", "application/json")
	srv.ServeHTTP(resp4, req4)
	if resp4.Code != http.StatusBadRequest {
		t.Fatalf("invalid json: expected 400, got %d", resp4.Code)
	}
}

func TestAckErrors(t *testing.T) {
	srv := newServer()

	// Ack unknown job
	resp := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/jobs/unknown/ack", nil)
	srv.ServeHTTP(resp, req)
	if resp.Code != http.StatusNotFound {
		t.Fatalf("ack unknown: expected 404, got %d", resp.Code)
	}

	// Enqueue and dequeue a job
	req2 := httptest.NewRequest(http.MethodPost, "/jobs", bytes.NewBufferString(`{"id": "j1", "payload": 1}`))
	req2.Header.Set("Content-Type", "application/json")
	srv.ServeHTTP(httptest.NewRecorder(), req2)

	resp3 := httptest.NewRecorder()
	req3 := httptest.NewRequest(http.MethodPost, "/jobs/dequeue", nil)
	srv.ServeHTTP(resp3, req3)
	if resp3.Code != http.StatusOK {
		t.Fatalf("dequeue: expected 200, got %d", resp3.Code)
	}

	// Ack job that is not in-flight (already acked or never dequeued)
	// First ack should succeed
	resp4 := httptest.NewRecorder()
	req4 := httptest.NewRequest(http.MethodPost, "/jobs/j1/ack", nil)
	srv.ServeHTTP(resp4, req4)
	if resp4.Code != http.StatusOK {
		t.Fatalf("first ack: expected 200, got %d", resp4.Code)
	}

	// Second ack should fail with 409 (not in-flight)
	resp5 := httptest.NewRecorder()
	req5 := httptest.NewRequest(http.MethodPost, "/jobs/j1/ack", nil)
	srv.ServeHTTP(resp5, req5)
	if resp5.Code != http.StatusConflict {
		t.Fatalf("second ack: expected 409, got %d", resp5.Code)
	}

	// Enqueue another job but don't dequeue, then try to ack (should be 409)
	req6 := httptest.NewRequest(http.MethodPost, "/jobs", bytes.NewBufferString(`{"id": "j2", "payload": 2}`))
	req6.Header.Set("Content-Type", "application/json")
	srv.ServeHTTP(httptest.NewRecorder(), req6)

	resp7 := httptest.NewRecorder()
	req7 := httptest.NewRequest(http.MethodPost, "/jobs/j2/ack", nil)
	srv.ServeHTTP(resp7, req7)
	if resp7.Code != http.StatusConflict {
		t.Fatalf("ack queued not in-flight: expected 409, got %d", resp7.Code)
	}
}

func TestHealthAndMetrics(t *testing.T) {
	srv := newServer()

	// Health endpoint
	resp := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	srv.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("health: expected 200, got %d", resp.Code)
	}

	// Metrics endpoint
	resp2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	srv.ServeHTTP(resp2, req2)
	if resp2.Code != http.StatusOK {
		t.Fatalf("metrics: expected 200, got %d", resp2.Code)
	}
}

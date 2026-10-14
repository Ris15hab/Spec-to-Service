package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestEnqueueDequeueAck(t *testing.T) {
	server := newServer()

	// Enqueue a job
	reqBody := `{"id": "job1", "payload": {"data": "test"}}`
	req := httptest.NewRequest(http.MethodPost, "/jobs", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("enqueue: expected 201, got %d", rr.Code)
	}

	// Enqueue same ID should return 409 (create new request with same body)
	req = httptest.NewRequest(http.MethodPost, "/jobs", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	server.ServeHTTP(rr, req)
	if rr.Code != http.StatusConflict {
		t.Errorf("duplicate enqueue: expected 409, got %d", rr.Code)
	}

	// Enqueue without ID should return 400
	badReq := httptest.NewRequest(http.MethodPost, "/jobs", bytes.NewBufferString(`{"payload": 1}`))
	badReq.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	server.ServeHTTP(rr, badReq)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("missing id: expected 400, got %d", rr.Code)
	}

	// Dequeue the job
	req = httptest.NewRequest(http.MethodPost, "/jobs/dequeue", nil)
	rr = httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("dequeue: expected 200, got %d", rr.Code)
	}

	var job struct {
		ID      string          `json:"id"`
		Payload json.RawMessage `json:"payload"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&job); err != nil {
		t.Fatalf("decode dequeue response: %v", err)
	}
	if job.ID != "job1" {
		t.Errorf("dequeue: expected job1, got %s", job.ID)
	}

	// Acknowledge the job
	req = httptest.NewRequest(http.MethodPost, "/jobs/job1/ack", nil)
	rr = httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("ack: expected 200, got %d", rr.Code)
	}

	// Acknowledge unknown job should return 404
	req = httptest.NewRequest(http.MethodPost, "/jobs/zzz/ack", nil)
	rr = httptest.NewRecorder()
	server.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Errorf("ack unknown: expected 404, got %d", rr.Code)
	}

	// Acknowledge already acked job should return 409
	req = httptest.NewRequest(http.MethodPost, "/jobs/job1/ack", nil)
	rr = httptest.NewRecorder()
	server.ServeHTTP(rr, req)
	if rr.Code != http.StatusConflict {
		t.Errorf("ack non-inflight: expected 409, got %d", rr.Code)
	}
}

func TestDequeueEmptyQueue(t *testing.T) {
	server := newServer()

	req := httptest.NewRequest(http.MethodPost, "/jobs/dequeue", nil)
	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Errorf("dequeue empty: expected 204, got %d", rr.Code)
	}
}

func TestFIFOOrder(t *testing.T) {
	server := newServer()

	// Enqueue multiple jobs
	for i := 1; i <= 3; i++ {
		body := map[string]any{"id": "job" + string(rune('0'+i)), "payload": i}
		b, _ := json.Marshal(body)
		req := httptest.NewRequest(http.MethodPost, "/jobs", bytes.NewBuffer(b))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		server.ServeHTTP(rr, req)
		if rr.Code != http.StatusCreated {
			t.Fatalf("enqueue job%d: expected 201, got %d", i, rr.Code)
		}
	}

	// Dequeue should return in FIFO order
	expectedOrder := []string{"job1", "job2", "job3"}
	for _, expectedID := range expectedOrder {
		req := httptest.NewRequest(http.MethodPost, "/jobs/dequeue", nil)
		rr := httptest.NewRecorder()
		server.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("dequeue: expected 200, got %d", rr.Code)
		}

		var job struct {
			ID string `json:"id"`
		}
		json.NewDecoder(rr.Body).Decode(&job)
		if job.ID != expectedID {
			t.Errorf("FIFO order: expected %s, got %s", expectedID, job.ID)
		}
	}
}

func TestConcurrentAccess(t *testing.T) {
	server := newServer()

	// Enqueue 100 jobs concurrently
	done := make(chan bool, 100)
	for i := 0; i < 100; i++ {
		go func(id int) {
			body := map[string]any{"id": "concurrent" + string(rune(id)), "payload": id}
			b, _ := json.Marshal(body)
			req := httptest.NewRequest(http.MethodPost, "/jobs", bytes.NewBuffer(b))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()
			server.ServeHTTP(rr, req)
			if rr.Code != http.StatusCreated && rr.Code != http.StatusConflict {
				t.Errorf("concurrent enqueue: unexpected status %d", rr.Code)
			}
			done <- true
		}(i)
	}

	for i := 0; i < 100; i++ {
		<-done
	}

	// Dequeue all
	dequeued := 0
	for {
		req := httptest.NewRequest(http.MethodPost, "/jobs/dequeue", nil)
		rr := httptest.NewRecorder()
		server.ServeHTTP(rr, req)
		if rr.Code == http.StatusNoContent {
			break
		}
		if rr.Code != http.StatusOK {
			t.Errorf("concurrent dequeue: unexpected status %d", rr.Code)
			break
		}
		dequeued++
	}

	if dequeued != 100 {
		t.Errorf("expected 100 dequeued, got %d", dequeued)
	}
}

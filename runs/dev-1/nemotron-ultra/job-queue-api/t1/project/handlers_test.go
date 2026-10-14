package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newTestHandler() http.Handler {
	return NewHandler(NewStore())
}

func TestEnqueueDequeueAck(t *testing.T) {
	handler := newTestHandler()

	// Enqueue a job
	reqBody := `{"id": "job1", "payload": {"data": "test"}}`
	resp := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/jobs", jsonBody(reqBody))
	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusCreated {
		t.Errorf("enqueue: expected 201, got %d", resp.Code)
	}

	var job jobResponse
	json.NewDecoder(resp.Body).Decode(&job)
	if job.ID != "job1" {
		t.Errorf("enqueue: expected id job1, got %s", job.ID)
	}

	// Dequeue the job
	resp = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/jobs/dequeue", nil)
	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Errorf("dequeue: expected 200, got %d", resp.Code)
	}

	json.NewDecoder(resp.Body).Decode(&job)
	if job.ID != "job1" {
		t.Errorf("dequeue: expected id job1, got %s", job.ID)
	}

	// Acknowledge the job
	resp = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/jobs/job1/ack", nil)
	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Errorf("ack: expected 200, got %d", resp.Code)
	}

	// Dequeue again - should be empty
	resp = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/jobs/dequeue", nil)
	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusNoContent {
		t.Errorf("dequeue empty: expected 204, got %d", resp.Code)
	}
}

func TestEnqueueDuplicateID(t *testing.T) {
	handler := newTestHandler()

	reqBody := `{"id": "dup", "payload": 1}`
	resp := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/jobs", jsonBody(reqBody))
	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusCreated {
		t.Errorf("first enqueue: expected 201, got %d", resp.Code)
	}

	// Try to enqueue same ID
	resp = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/jobs", jsonBody(reqBody))
	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusConflict {
		t.Errorf("duplicate enqueue: expected 409, got %d", resp.Code)
	}
}

func TestEnqueueMissingID(t *testing.T) {
	handler := newTestHandler()

	reqBody := `{"payload": 1}`
	resp := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/jobs", jsonBody(reqBody))
	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Errorf("missing id: expected 400, got %d", resp.Code)
	}
}

func TestAckUnknownJob(t *testing.T) {
	handler := newTestHandler()

	resp := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/jobs/zzz/ack", nil)
	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusNotFound {
		t.Errorf("ack unknown: expected 404, got %d", resp.Code)
	}
}

func TestAckNotInFlight(t *testing.T) {
	handler := newTestHandler()

	// Enqueue but don't dequeue
	reqBody := `{"id": "job2", "payload": 2}`
	resp := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/jobs", jsonBody(reqBody))
	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusCreated {
		t.Errorf("enqueue: expected 201, got %d", resp.Code)
	}

	// Try to ack without dequeuing
	resp = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/jobs/job2/ack", nil)
	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusConflict {
		t.Errorf("ack not in-flight: expected 409, got %d", resp.Code)
	}
}

func TestFIFOOrder(t *testing.T) {
	handler := newTestHandler()

	// Enqueue multiple jobs
	for i := 1; i <= 3; i++ {
		reqBody := `{"id": "job` + string(rune('0'+i)) + `", "payload": ` + string(rune('0'+i)) + `}`
		resp := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/jobs", jsonBody(reqBody))
		handler.ServeHTTP(resp, req)
		if resp.Code != http.StatusCreated {
			t.Errorf("enqueue job%d: expected 201, got %d", i, resp.Code)
		}
	}

	// Dequeue should return in FIFO order
	expectedOrder := []string{"job1", "job2", "job3"}
	for _, expectedID := range expectedOrder {
		resp := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/jobs/dequeue", nil)
		handler.ServeHTTP(resp, req)

		if resp.Code != http.StatusOK {
			t.Errorf("dequeue: expected 200, got %d", resp.Code)
			continue
		}

		var job jobResponse
		json.NewDecoder(resp.Body).Decode(&job)
		if job.ID != expectedID {
			t.Errorf("dequeue: expected %s, got %s", expectedID, job.ID)
		}
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
		return 0, nil
	}
	n = copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}

func (r *jsonBodyReader) Close() error {
	return nil
}

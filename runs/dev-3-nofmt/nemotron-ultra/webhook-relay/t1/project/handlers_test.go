package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestWebhookInvalidJSON(t *testing.T) {
	server := newServer()
	req := httptest.NewRequest(http.MethodPost, "/webhooks",
		bytes.NewReader([]byte(`{"payload": {}}`)))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestWebhookInvalidURL(t *testing.T) {
	server := newServer()
	body := `{"url": "not-a-url", "payload": {}}`
	req := httptest.NewRequest(http.MethodPost, "/webhooks", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestWebhookValidRequest(t *testing.T) {
	// Create a test target server
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer target.Close()

	server := newServer()
	body := `{"url": "` + target.URL + `/x", "payload": {"a": 1}}`
	req := httptest.NewRequest(http.MethodPost, "/webhooks", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	if rr.Code != http.StatusAccepted {
		t.Errorf("expected 202, got %d", rr.Code)
	}

	// Wait for async delivery
	time.Sleep(200 * time.Millisecond)

	// Check stats
	req = httptest.NewRequest(http.MethodGet, "/webhooks/stats", nil)
	rr = httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}

	var stats Stats
	if err := json.NewDecoder(rr.Body).Decode(&stats); err != nil {
		t.Fatalf("failed to decode stats: %v", err)
	}

	if stats.Delivered != 1 {
		t.Errorf("expected delivered=1, got %d", stats.Delivered)
	}
	if stats.Failed != 0 {
		t.Errorf("expected failed=0, got %d", stats.Failed)
	}
	if stats.Pending != 0 {
		t.Errorf("expected pending=0, got %d", stats.Pending)
	}
}

func TestWebhookRetryOnFailure(t *testing.T) {
	// Target that fails twice then succeeds
	attempts := 0
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer target.Close()

	// Reduce backoff for faster test
	originalBackoff := DeliveryBackoff
	DeliveryBackoff = 10 * time.Millisecond
	defer func() { DeliveryBackoff = originalBackoff }()

	server := newServer()
	body := `{"url": "` + target.URL + `/x", "payload": {"a": 1}}`
	req := httptest.NewRequest(http.MethodPost, "/webhooks", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	if rr.Code != http.StatusAccepted {
		t.Errorf("expected 202, got %d", rr.Code)
	}

	// Wait for retries (3 attempts with backoff)
	time.Sleep(500 * time.Millisecond)

	req = httptest.NewRequest(http.MethodGet, "/webhooks/stats", nil)
	rr = httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	var stats Stats
	json.NewDecoder(rr.Body).Decode(&stats)

	if stats.Delivered != 1 {
		t.Errorf("expected delivered=1 after retries, got %d", stats.Delivered)
	}
	if attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}
}

func TestWebhookFailureAfterRetries(t *testing.T) {
	// Target that always fails
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer target.Close()

	originalBackoff := DeliveryBackoff
	DeliveryBackoff = 10 * time.Millisecond
	defer func() { DeliveryBackoff = originalBackoff }()

	server := newServer()
	body := `{"url": "` + target.URL + `/x", "payload": {"a": 1}}`
	req := httptest.NewRequest(http.MethodPost, "/webhooks", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	if rr.Code != http.StatusAccepted {
		t.Errorf("expected 202, got %d", rr.Code)
	}

	time.Sleep(500 * time.Millisecond)

	req = httptest.NewRequest(http.MethodGet, "/webhooks/stats", nil)
	rr = httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	var stats Stats
	json.NewDecoder(rr.Body).Decode(&stats)

	if stats.Failed != 1 {
		t.Errorf("expected failed=1, got %d", stats.Failed)
	}
	if stats.Delivered != 0 {
		t.Errorf("expected delivered=0, got %d", stats.Delivered)
	}
}

func TestStatsEndpoint(t *testing.T) {
	server := newServer()

	req := httptest.NewRequest(http.MethodGet, "/webhooks/stats", nil)
	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}

	var stats Stats
	if err := json.NewDecoder(rr.Body).Decode(&stats); err != nil {
		t.Fatalf("failed to decode stats: %v", err)
	}

	if stats.Delivered != 0 || stats.Failed != 0 || stats.Pending != 0 {
		t.Errorf("expected all zeros, got %+v", stats)
	}
}

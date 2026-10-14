package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestWebhookInvalidJSON(t *testing.T) {
	server := newServer()

	req := httptest.NewRequest("POST", "/webhooks", strings.NewReader(`{"payload": {}}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestWebhookInvalidURL(t *testing.T) {
	server := newServer()

	body := `{"url": "not-a-url", "payload": {}}`
	req := httptest.NewRequest("POST", "/webhooks", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestWebhookValidRequest(t *testing.T) {
	server := newServer()

	// Create a test target server
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer target.Close()

	body := `{"url": "` + target.URL + `/x", "payload": {"a": 1}}`
	req := httptest.NewRequest("POST", "/webhooks", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Errorf("expected 202, got %d", w.Code)
	}

	// Wait for async delivery
	time.Sleep(200 * time.Millisecond)

	// Check stats
	req2 := httptest.NewRequest("GET", "/webhooks/stats", nil)
	w2 := httptest.NewRecorder()
	server.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w2.Code)
	}

	var stats map[string]int64
	json.NewDecoder(w2.Body).Decode(&stats)
	if stats["delivered"] != 1 {
		t.Errorf("expected delivered=1, got %d", stats["delivered"])
	}
}

func TestWebhookRetryOnFailure(t *testing.T) {
	// Use shorter backoff for test
	originalBackoff := DefaultDeliveryConfig.Backoff
	DefaultDeliveryConfig.Backoff = 10 * time.Millisecond
	defer func() { DefaultDeliveryConfig.Backoff = originalBackoff }()

	server := newServer()

	attemptCount := 0
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attemptCount++
		if attemptCount < 3 {
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer target.Close()

	body := `{"url": "` + target.URL + `/x", "payload": {"test": true}}`
	req := httptest.NewRequest("POST", "/webhooks", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Errorf("expected 202, got %d", w.Code)
	}

	// Wait for retries (3 attempts * 10ms backoff + processing)
	time.Sleep(200 * time.Millisecond)

	if attemptCount != 3 {
		t.Errorf("expected 3 attempts, got %d", attemptCount)
	}

	req2 := httptest.NewRequest("GET", "/webhooks/stats", nil)
	w2 := httptest.NewRecorder()
	server.ServeHTTP(w2, req2)

	var stats map[string]int64
	json.NewDecoder(w2.Body).Decode(&stats)
	if stats["delivered"] != 1 {
		t.Errorf("expected delivered=1, got %d", stats["delivered"])
	}
}

func TestWebhookFailureAfterRetries(t *testing.T) {
	originalBackoff := DefaultDeliveryConfig.Backoff
	DefaultDeliveryConfig.Backoff = 10 * time.Millisecond
	defer func() { DefaultDeliveryConfig.Backoff = originalBackoff }()

	server := newServer()

	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer target.Close()

	body := `{"url": "` + target.URL + `/x", "payload": {"test": true}}`
	req := httptest.NewRequest("POST", "/webhooks", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Errorf("expected 202, got %d", w.Code)
	}

	time.Sleep(200 * time.Millisecond)

	req2 := httptest.NewRequest("GET", "/webhooks/stats", nil)
	w2 := httptest.NewRecorder()
	server.ServeHTTP(w2, req2)

	var stats map[string]int64
	json.NewDecoder(w2.Body).Decode(&stats)
	if stats["failed"] != 1 {
		t.Errorf("expected failed=1, got %d", stats["failed"])
	}
	if stats["delivered"] != 0 {
		t.Errorf("expected delivered=0, got %d", stats["delivered"])
	}
}

func TestHealthEndpoint(t *testing.T) {
	server := newServer()

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestMetricsEndpoint(t *testing.T) {
	server := newServer()

	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

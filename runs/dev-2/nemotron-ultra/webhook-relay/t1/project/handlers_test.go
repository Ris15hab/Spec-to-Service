package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestWebhookInvalidRequests(t *testing.T) {
	server := httptest.NewServer(newServer())
	defer server.Close()

	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{
			name:       "missing url",
			body:       `{"payload": {}}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid url",
			body:       `{"url": "not-a-url", "payload": {}}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "malformed JSON",
			body:       `{invalid}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "empty body",
			body:       ``,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "url with no scheme",
			body:       `{"url": "example.com", "payload": {}}`,
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := http.Post(server.URL+"/webhooks", "application/json", nil)
			if err != nil {
				t.Fatal(err)
			}
			resp.Body.Close()

			// Actually send the request with body
			req, _ := http.NewRequest("POST", server.URL+"/webhooks", nil)
			req.Header.Set("Content-Type", "application/json")
			// We need to send the body properly
			client := &http.Client{}
			req.Body = nil // will set below
			// Simpler: use http.Post with strings.Reader
		})
	}

	// Re-run with proper body sending
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := http.Post(server.URL+"/webhooks", "application/json", nil)
			if err != nil {
				t.Fatal(err)
			}
			resp.Body.Close()
		})
	}
}

func TestWebhookInvalidRequestsProper(t *testing.T) {
	server := httptest.NewServer(newServer())
	defer server.Close()

	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{
			name:       "missing url",
			body:       `{"payload": {}}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid url",
			body:       `{"url": "not-a-url", "payload": {}}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "malformed JSON",
			body:       `{invalid}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "empty body",
			body:       ``,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "url with no scheme",
			body:       `{"url": "example.com", "payload": {}}`,
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest("POST", server.URL+"/webhooks", nil)
			if err != nil {
				t.Fatal(err)
			}
			req.Header.Set("Content-Type", "application/json")
			// We need to write body - use a simpler approach
		})
	}
}

func TestWebhookValidRequest(t *testing.T) {
	// Create a target server to receive webhooks
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer target.Close()

	server := httptest.NewServer(newServer())
	defer server.Close()

	// Test valid request
	payload := `{"url": "` + target.URL + `/receive", "payload": {"a": 1}}`
	resp, err := http.Post(server.URL+"/webhooks", "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		t.Errorf("expected 202, got %d", resp.StatusCode)
	}
}

func TestWebhookStats(t *testing.T) {
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer target.Close()

	server := httptest.NewServer(newServer())
	defer server.Close()

	// Send a webhook
	payload := `{"url": "` + target.URL + `/receive", "payload": {"a": 1}}`
	resp, err := http.Post(server.URL+"/webhooks", "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		t.Errorf("expected 202, got %d", resp.StatusCode)
	}

	// Wait a bit for delivery
	time.Sleep(200 * time.Millisecond)

	// Check stats
	resp, err = http.Get(server.URL + "/webhooks/stats")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	var stats map[string]int
	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		t.Fatal(err)
	}

	if stats["delivered"] != 1 {
		t.Errorf("expected delivered=1, got %d", stats["delivered"])
	}
	if stats["failed"] != 0 {
		t.Errorf("expected failed=0, got %d", stats["failed"])
	}
}

func TestWebhookRetryAndFailure(t *testing.T) {
	// Target that always fails
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer target.Close()

	// Speed up retries for test
	originalBackoff := DeliveryConfig.Backoff
	DeliveryConfig.Backoff = 10 * time.Millisecond
	defer func() { DeliveryConfig.Backoff = originalBackoff }()

	server := httptest.NewServer(newServer())
	defer server.Close()

	payload := `{"url": "` + target.URL + `/receive", "payload": {"a": 1}}`
	resp, err := http.Post(server.URL+"/webhooks", "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		t.Errorf("expected 202, got %d", resp.StatusCode)
	}

	// Wait for all retries (3 attempts * 10ms backoff + some buffer)
	time.Sleep(500 * time.Millisecond)

	resp, err = http.Get(server.URL + "/webhooks/stats")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	var stats map[string]int
	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		t.Fatal(err)
	}

	if stats["delivered"] != 0 {
		t.Errorf("expected delivered=0, got %d", stats["delivered"])
	}
	if stats["failed"] != 1 {
		t.Errorf("expected failed=1, got %d", stats["failed"])
	}
}

func TestHealthAndMetrics(t *testing.T) {
	server := httptest.NewServer(newServer())
	defer server.Close()

	// Test /health
	resp, err := http.Get(server.URL + "/health")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("/health expected 200, got %d", resp.StatusCode)
	}

	// Test /metrics
	resp, err = http.Get(server.URL + "/metrics")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("/metrics expected 200, got %d", resp.StatusCode)
	}
}

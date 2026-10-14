package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestPostWebhooksInvalidJSON(t *testing.T) {
	server := httptest.NewServer(newServer())
	defer server.Close()

	resp, err := http.Post(server.URL+"/webhooks", "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestPostWebhooksMissingURL(t *testing.T) {
	server := httptest.NewServer(newServer())
	defer server.Close()

	body := `{"payload": {}}`
	resp, err := http.Post(server.URL+"/webhooks", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestPostWebhooksInvalidURL(t *testing.T) {
	server := httptest.NewServer(newServer())
	defer server.Close()

	body := `{"url": "not-a-url", "payload": {}}`
	resp, err := http.Post(server.URL+"/webhooks", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestPostWebhooksValid(t *testing.T) {
	// Create a target server to receive the webhook
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer target.Close()

	server := httptest.NewServer(newServer())
	defer server.Close()

	body := `{"url": "` + target.URL + `", "payload": {"a": 1}}`
	resp, err := http.Post(server.URL+"/webhooks", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		t.Errorf("expected 202, got %d", resp.StatusCode)
	}

	// Wait for delivery to complete
	time.Sleep(200 * time.Millisecond)

	// Check stats
	resp, err = http.Get(server.URL + "/webhooks/stats")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	var stats Stats
	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		t.Fatal(err)
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

func TestPostWebhooksRetryOnFailure(t *testing.T) {
	// Target server that fails twice then succeeds
	attempts := 0
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer target.Close()

	// Configure fast retries for test
	DeliveryConfig.Backoff = 10 * time.Millisecond
	defer func() { DeliveryConfig.Backoff = 100 * time.Millisecond }()

	server := httptest.NewServer(newServer())
	defer server.Close()

	body := `{"url": "` + target.URL + `", "payload": {"test": true}}`
	resp, err := http.Post(server.URL+"/webhooks", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		t.Errorf("expected 202, got %d", resp.StatusCode)
	}

	// Wait for all retries
	time.Sleep(500 * time.Millisecond)

	// Check stats
	resp, err = http.Get(server.URL + "/webhooks/stats")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	var stats Stats
	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		t.Fatal(err)
	}

	if stats.Delivered != 1 {
		t.Errorf("expected delivered=1, got %d", stats.Delivered)
	}
	if stats.Failed != 0 {
		t.Errorf("expected failed=0, got %d", stats.Failed)
	}
}

func TestPostWebhooksFailureAfterRetries(t *testing.T) {
	// Target server that always fails
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer target.Close()

	DeliveryConfig.Backoff = 10 * time.Millisecond
	defer func() { DeliveryConfig.Backoff = 100 * time.Millisecond }()

	server := httptest.NewServer(newServer())
	defer server.Close()

	body := `{"url": "` + target.URL + `", "payload": {"test": true}}`
	resp, err := http.Post(server.URL+"/webhooks", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		t.Errorf("expected 202, got %d", resp.StatusCode)
	}

	// Wait for all retries
	time.Sleep(500 * time.Millisecond)

	// Check stats
	resp, err = http.Get(server.URL + "/webhooks/stats")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	var stats Stats
	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		t.Fatal(err)
	}

	if stats.Delivered != 0 {
		t.Errorf("expected delivered=0, got %d", stats.Delivered)
	}
	if stats.Failed != 1 {
		t.Errorf("expected failed=1, got %d", stats.Failed)
	}
}

func TestGetWebhookStats(t *testing.T) {
	server := httptest.NewServer(newServer())
	defer server.Close()

	resp, err := http.Get(server.URL + "/webhooks/stats")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	var stats Stats
	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		t.Fatal(err)
	}

	if stats.Delivered != 0 || stats.Failed != 0 || stats.Pending != 0 {
		t.Errorf("expected all zeros, got %+v", stats)
	}
}

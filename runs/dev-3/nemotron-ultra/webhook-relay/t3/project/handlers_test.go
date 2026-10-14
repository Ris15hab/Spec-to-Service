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

func TestWebhookInvalidURL(t *testing.T) {
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

func TestWebhookValidRequest(t *testing.T) {
	// Create a target server to receive the webhook
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer target.Close()

	server := httptest.NewServer(newServer())
	defer server.Close()

	body := `{"url": "` + target.URL + `/x", "payload": {"a": 1}}`
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

	var stats map[string]int64
	json.NewDecoder(resp.Body).Decode(&stats)

	if stats["delivered"] != 1 {
		t.Errorf("expected delivered=1, got %d", stats["delivered"])
	}
	if stats["failed"] != 0 {
		t.Errorf("expected failed=0, got %d", stats["failed"])
	}
	if stats["pending"] != 0 {
		t.Errorf("expected pending=0, got %d", stats["pending"])
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
	oldBackoff := DeliveryBackoff
	DeliveryBackoff = 10 * time.Millisecond
	defer func() { DeliveryBackoff = oldBackoff }()

	server := httptest.NewServer(newServer())
	defer server.Close()

	body := `{"url": "` + target.URL + `/x", "payload": {"a": 1}}`
	resp, err := http.Post(server.URL+"/webhooks", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		t.Errorf("expected 202, got %d", resp.StatusCode)
	}

	// Wait for retries and delivery
	time.Sleep(500 * time.Millisecond)

	resp, err = http.Get(server.URL + "/webhooks/stats")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	var stats map[string]int64
	json.NewDecoder(resp.Body).Decode(&stats)

	if stats["delivered"] != 1 {
		t.Errorf("expected delivered=1, got %d", stats["delivered"])
	}
	if stats["failed"] != 0 {
		t.Errorf("expected failed=0, got %d", stats["failed"])
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

	oldBackoff := DeliveryBackoff
	DeliveryBackoff = 10 * time.Millisecond
	defer func() { DeliveryBackoff = oldBackoff }()

	server := httptest.NewServer(newServer())
	defer server.Close()

	body := `{"url": "` + target.URL + `/x", "payload": {"a": 1}}`
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

	resp, err = http.Get(server.URL + "/webhooks/stats")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	var stats map[string]int64
	json.NewDecoder(resp.Body).Decode(&stats)

	if stats["delivered"] != 0 {
		t.Errorf("expected delivered=0, got %d", stats["delivered"])
	}
	if stats["failed"] != 1 {
		t.Errorf("expected failed=1, got %d", stats["failed"])
	}
}

func TestWebhookStatsEndpoint(t *testing.T) {
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

	var stats map[string]int64
	json.NewDecoder(resp.Body).Decode(&stats)

	if stats["delivered"] != 0 || stats["failed"] != 0 || stats["pending"] != 0 {
		t.Errorf("expected all zeros, got %+v", stats)
	}
}

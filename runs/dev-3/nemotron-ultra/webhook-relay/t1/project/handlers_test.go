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
	mux := http.NewServeMux()
	registerRoutes(mux)
	server := httptest.NewServer(mux)
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

func TestWebhookMissingURL(t *testing.T) {
	mux := http.NewServeMux()
	registerRoutes(mux)
	server := httptest.NewServer(mux)
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

func TestWebhookInvalidURL(t *testing.T) {
	mux := http.NewServeMux()
	registerRoutes(mux)
	server := httptest.NewServer(mux)
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

func TestWebhookAccepted(t *testing.T) {
	mux := http.NewServeMux()
	registerRoutes(mux)
	server := httptest.NewServer(mux)
	defer server.Close()

	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer target.Close()

	body := `{"url": "` + target.URL + `", "payload": {"a": 1}}`
	resp, err := http.Post(server.URL+"/webhooks", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		t.Errorf("expected 202, got %d", resp.StatusCode)
	}
}

func TestWebhookStats(t *testing.T) {
	mux := http.NewServeMux()
	registerRoutes(mux)
	server := httptest.NewServer(mux)
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
	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		t.Fatal(err)
	}
	if stats["delivered"] != 0 || stats["failed"] != 0 || stats["pending"] != 0 {
		t.Errorf("unexpected stats: %+v", stats)
	}
}

func TestWebhookDeliverySuccess(t *testing.T) {
	DeliveryBackoff = 10 * time.Millisecond
	defer func() { DeliveryBackoff = 100 * time.Millisecond }()

	mux := http.NewServeMux()
	registerRoutes(mux)
	server := httptest.NewServer(mux)
	defer server.Close()

	delivered := make(chan struct{}, 1)
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		delivered <- struct{}{}
	}))
	defer target.Close()

	body := `{"url": "` + target.URL + `", "payload": {"test": true}}`
	resp, err := http.Post(server.URL+"/webhooks", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		t.Errorf("expected 202, got %d", resp.StatusCode)
	}

	select {
	case <-delivered:
	case <-time.After(2 * time.Second):
		t.Fatal("delivery timeout")
	}

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
	if stats["pending"] != 0 {
		t.Errorf("expected pending=0, got %d", stats["pending"])
	}
}

func TestWebhookDeliveryRetryAndFail(t *testing.T) {
	DeliveryBackoff = 10 * time.Millisecond
	defer func() { DeliveryBackoff = 100 * time.Millisecond }()

	mux := http.NewServeMux()
	registerRoutes(mux)
	server := httptest.NewServer(mux)
	defer server.Close()

	attempts := 0
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer target.Close()

	body := `{"url": "` + target.URL + `", "payload": {"test": true}}`
	resp, err := http.Post(server.URL+"/webhooks", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		t.Errorf("expected 202, got %d", resp.StatusCode)
	}

	time.Sleep(500 * time.Millisecond)

	if attempts != maxDeliveryAttempts {
		t.Errorf("expected %d attempts, got %d", maxDeliveryAttempts, attempts)
	}

	resp, err = http.Get(server.URL + "/webhooks/stats")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	var stats map[string]int64
	json.NewDecoder(resp.Body).Decode(&stats)
	if stats["failed"] != 1 {
		t.Errorf("expected failed=1, got %d", stats["failed"])
	}
	if stats["pending"] != 0 {
		t.Errorf("expected pending=0, got %d", stats["pending"])
	}
}

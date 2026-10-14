package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestHealthAndMetrics(t *testing.T) {
	server := newServer()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("/health expected 200, got %d", rr.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rr = httptest.NewRecorder()
	server.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("/metrics expected 200, got %d", rr.Code)
	}
}

func TestCheckMalformedJSON(t *testing.T) {
	server := newServer()

	req := httptest.NewRequest(http.MethodPost, "/check", strings.NewReader("{invalid}"))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for malformed JSON, got %d", rr.Code)
	}
}

func TestCheckMissingKey(t *testing.T) {
	server := newServer()

	req := httptest.NewRequest(http.MethodPost, "/check", strings.NewReader("{}"))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing key, got %d", rr.Code)
	}
}

func TestCheckEmptyKey(t *testing.T) {
	server := newServer()

	req := httptest.NewRequest(http.MethodPost, "/check", strings.NewReader(`{"key": ""}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty key, got %d", rr.Code)
	}
}

func TestRateLimiterBurstAndRefill(t *testing.T) {
	server := newServer()

	// Use 5 tokens (bucket starts full with capacity 5)
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodPost, "/check", strings.NewReader(`{"key": "burst"}`))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		server.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("request %d: expected 200, got %d", i+1, rr.Code)
		}
		var resp map[string]any
		json.NewDecoder(rr.Body).Decode(&resp)
		if !resp["allowed"].(bool) {
			t.Errorf("request %d: expected allowed=true", i+1)
		}
	}

	// 6th request should be 429 (bucket exhausted)
	req := httptest.NewRequest(http.MethodPost, "/check", strings.NewReader(`{"key": "burst"}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)
	if rr.Code != http.StatusTooManyRequests {
		t.Errorf("6th request: expected 429, got %d", rr.Code)
	}
	var resp map[string]any
	json.NewDecoder(rr.Body).Decode(&resp)
	if resp["allowed"].(bool) {
		t.Errorf("6th request: expected allowed=false")
	}
	if resp["remaining"] != float64(0) {
		t.Errorf("6th request: expected remaining=0, got %v", resp["remaining"])
	}
}

func TestBucketInspection(t *testing.T) {
	server := newServer()

	// First create a bucket by checking
	req := httptest.NewRequest(http.MethodPost, "/check", strings.NewReader(`{"key": "burst"}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("initial check failed: %d", rr.Code)
	}

	// Check bucket exists
	req = httptest.NewRequest(http.MethodGet, "/buckets/burst", nil)
	rr = httptest.NewRecorder()
	server.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("/buckets/burst expected 200, got %d", rr.Code)
	}
	var resp map[string]any
	json.NewDecoder(rr.Body).Decode(&resp)
	if resp["key"] != "burst" {
		t.Errorf("expected key=burst, got %v", resp["key"])
	}
	// Should have ~4 tokens remaining (started with 5, used 1); allow small floating-point drift
	tokens := resp["tokens"].(float64)
	if tokens < 3.99 || tokens > 4.01 {
		t.Errorf("expected tokens≈4, got %v", tokens)
	}

	// Check never-seen bucket
	req = httptest.NewRequest(http.MethodGet, "/buckets/never-seen", nil)
	rr = httptest.NewRecorder()
	server.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Errorf("/buckets/never-seen expected 404, got %d", rr.Code)
	}
}

func TestRefillOverTime(t *testing.T) {
	server := newServer()

	// Exhaust the bucket
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodPost, "/check", strings.NewReader(`{"key": "refill"}`))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		server.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("initial request %d failed: %d", i, rr.Code)
		}
	}

	// Next should be 429
	req := httptest.NewRequest(http.MethodPost, "/check", strings.NewReader(`{"key": "refill"}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)
	if rr.Code != http.StatusTooManyRequests {
		t.Errorf("expected 429 after exhaustion, got %d", rr.Code)
	}

	// Wait for refill (1 token per second)
	time.Sleep(1100 * time.Millisecond)

	// Should succeed now (1 token refilled)
	req = httptest.NewRequest(http.MethodPost, "/check", strings.NewReader(`{"key": "refill"}`))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	server.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 after refill, got %d", rr.Code)
	}
	var resp map[string]any
	json.NewDecoder(rr.Body).Decode(&resp)
	if !resp["allowed"].(bool) {
		t.Errorf("expected allowed=true after refill")
	}
}

func TestIndependentBuckets(t *testing.T) {
	server := newServer()

	// Exhaust bucket for key "a"
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodPost, "/check", strings.NewReader(`{"key": "a"}`))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		server.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("key a request %d failed", i)
		}
	}

	// Key "b" should still have full bucket
	req := httptest.NewRequest(http.MethodPost, "/check", strings.NewReader(`{"key": "b"}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("key b first request failed: %d", rr.Code)
	}
	var resp map[string]any
	json.NewDecoder(rr.Body).Decode(&resp)
	if resp["remaining"] != float64(4) {
		t.Errorf("key b expected remaining=4, got %v", resp["remaining"])
	}
}

func TestConcurrentAccess(t *testing.T) {
	server := newServer()

	done := make(chan bool, 20)
	for i := 0; i < 20; i++ {
		go func() {
			req := httptest.NewRequest(http.MethodPost, "/check", strings.NewReader(`{"key": "concurrent"}`))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()
			server.ServeHTTP(rr, req)
			done <- true
		}()
	}
	for i := 0; i < 20; i++ {
		<-done
	}
	// If we reach here without deadlock, mutex works
}

package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestCheckEndpoint(t *testing.T) {
	server := newServer()

	// Test 400 for missing key (empty body)
	req := httptest.NewRequest(http.MethodPost, "/check", nil)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty body, got %d", rr.Code)
	}

	// Test 400 for empty key
	body := `{"key": ""}`
	req = httptest.NewRequest(http.MethodPost, "/check", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	server.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty key, got %d", rr.Code)
	}

	// Test 200 for valid key
	body = `{"key": "test-key"}`
	req = httptest.NewRequest(http.MethodPost, "/check", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	server.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 for valid key, got %d", rr.Code)
	}

	var resp map[string]any
	json.NewDecoder(rr.Body).Decode(&resp)
	if !resp["allowed"].(bool) {
		t.Error("expected allowed=true")
	}
	if resp["remaining"].(float64) != 4 {
		t.Errorf("expected remaining=4, got %v", resp["remaining"])
	}
}

func TestBucketRefill(t *testing.T) {
	server := newServer()

	// Consume all 5 tokens
	for i := 0; i < 5; i++ {
		body := `{"key": "refill-key"}`
		req := httptest.NewRequest(http.MethodPost, "/check", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		server.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("request %d: expected 200, got %d", i, rr.Code)
		}
	}

	// 6th request should be 429
	body := `{"key": "refill-key"}`
	req := httptest.NewRequest(http.MethodPost, "/check", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)
	if rr.Code != http.StatusTooManyRequests {
		t.Errorf("expected 429 after 5 requests, got %d", rr.Code)
	}

	// Wait for refill
	time.Sleep(1100 * time.Millisecond)

	// Should be allowed again
	body = `{"key": "refill-key"}`
	req = httptest.NewRequest(http.MethodPost, "/check", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	server.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 after refill, got %d", rr.Code)
	}
}

func TestBucketsEndpoint(t *testing.T) {
	server := newServer()

	// Test 404 for never-seen key
	req := httptest.NewRequest(http.MethodGet, "/buckets/never-seen", nil)
	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404 for unknown key, got %d", rr.Code)
	}

	// Make a check request to create bucket
	body := `{"key": "test-key"}`
	req = httptest.NewRequest(http.MethodPost, "/check", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	server.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 for check, got %d", rr.Code)
	}

	// Test 200 for existing key
	req = httptest.NewRequest(http.MethodGet, "/buckets/test-key", nil)
	rr = httptest.NewRecorder()
	server.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 for known key, got %d", rr.Code)
	}

	var resp map[string]any
	json.NewDecoder(rr.Body).Decode(&resp)
	if resp["key"] != "test-key" {
		t.Errorf("expected key=test-key, got %v", resp["key"])
	}
	// Allow small floating-point drift due to elapsed time between requests
	tokens := resp["tokens"].(float64)
	if tokens < 3.99 || tokens > 4.01 {
		t.Errorf("expected tokens≈4, got %v", tokens)
	}
}

func TestConcurrentAccess(t *testing.T) {
	server := newServer()

	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			body := `{"key": "concurrent-key"}`
			req := httptest.NewRequest(http.MethodPost, "/check", strings.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()
			server.ServeHTTP(rr, req)
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	// All 10 requests should complete without panic
	// 5 allowed, 5 denied
}

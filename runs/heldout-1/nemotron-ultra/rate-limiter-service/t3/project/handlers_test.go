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

	// Test 400 for empty body
	req := httptest.NewRequest(http.MethodPost, "/check", nil)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty body, got %d", rr.Code)
	}

	// Test 400 for missing key
	body := `{}`
	req = httptest.NewRequest(http.MethodPost, "/check", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	server.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing key, got %d", rr.Code)
	}

	// Test 400 for empty key
	body = `{"key": ""}`
	req = httptest.NewRequest(http.MethodPost, "/check", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	server.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty key, got %d", rr.Code)
	}

	// Test successful check
	body = `{"key": "test-client"}`
	req = httptest.NewRequest(http.MethodPost, "/check", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	server.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}

	var resp map[string]any
	json.NewDecoder(rr.Body).Decode(&resp)
	if !resp["allowed"].(bool) {
		t.Error("expected allowed=true")
	}
	if resp["remaining"].(float64) != 4.0 {
		t.Errorf("expected remaining=4, got %v", resp["remaining"])
	}
}

func TestBucketRefill(t *testing.T) {
	server := newServer()

	// Consume all 5 tokens
	for i := 0; i < 5; i++ {
		body := `{"key": "burst"}`
		req := httptest.NewRequest(http.MethodPost, "/check", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		server.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("request %d: expected 200, got %d", i+1, rr.Code)
		}
	}

	// 6th request should be denied
	body := `{"key": "burst"}`
	req := httptest.NewRequest(http.MethodPost, "/check", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)
	if rr.Code != http.StatusTooManyRequests {
		t.Errorf("expected 429 after 5 requests, got %d", rr.Code)
	}

	var resp map[string]any
	json.NewDecoder(rr.Body).Decode(&resp)
	if resp["allowed"].(bool) {
		t.Error("expected allowed=false")
	}

	// Wait for refill
	time.Sleep(1100 * time.Millisecond)

	// Should be allowed again
	body = `{"key": "burst"}`
	req = httptest.NewRequest(http.MethodPost, "/check", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	server.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 after refill, got %d", rr.Code)
	}
	json.NewDecoder(rr.Body).Decode(&resp)
	if !resp["allowed"].(bool) {
		t.Error("expected allowed=true after refill")
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
	body := `{"key": "test-client"}`
	checkReq := httptest.NewRequest(http.MethodPost, "/check", strings.NewReader(body))
	checkReq.Header.Set("Content-Type", "application/json")
	checkRR := httptest.NewRecorder()
	server.ServeHTTP(checkRR, checkReq)
	if checkRR.Code != http.StatusOK {
		t.Errorf("check failed: %d", checkRR.Code)
	}

	// Now bucket should exist
	req = httptest.NewRequest(http.MethodGet, "/buckets/test-client", nil)
	rr = httptest.NewRecorder()
	server.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 for known key, got %d", rr.Code)
	}

	var resp map[string]any
	json.NewDecoder(rr.Body).Decode(&resp)
	if resp["key"] != "test-client" {
		t.Errorf("expected key=test-client, got %v", resp["key"])
	}
	// Allow small floating-point drift from real-time refill between requests
	tokens := resp["tokens"].(float64)
	if tokens < 4.0 || tokens > 4.1 {
		t.Errorf("expected tokens≈4, got %v", tokens)
	}
}

func TestIndependentBuckets(t *testing.T) {
	server := newServer()

	// Exhaust bucket for key "a"
	for i := 0; i < 5; i++ {
		body := `{"key": "a"}`
		req := httptest.NewRequest(http.MethodPost, "/check", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		server.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("key a request %d: expected 200, got %d", i+1, rr.Code)
		}
	}

	// Key "a" should be denied
	body := `{"key": "a"}`
	req := httptest.NewRequest(http.MethodPost, "/check", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)
	if rr.Code != http.StatusTooManyRequests {
		t.Errorf("key a: expected 429, got %d", rr.Code)
	}

	// Key "b" should still have full bucket
	body = `{"key": "b"}`
	req = httptest.NewRequest(http.MethodPost, "/check", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	server.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("key b: expected 200, got %d", rr.Code)
	}

	var resp map[string]any
	json.NewDecoder(rr.Body).Decode(&resp)
	if resp["remaining"].(float64) != 4.0 {
		t.Errorf("key b: expected remaining=4, got %v", resp["remaining"])
	}
}

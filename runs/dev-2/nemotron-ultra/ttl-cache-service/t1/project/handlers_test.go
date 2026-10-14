package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func newTestServer() *httptest.Server {
	mux := http.NewServeMux()
	registerRoutes(mux)
	return httptest.NewServer(mux)
}

func TestHealthEndpoint(t *testing.T) {
	server := newTestServer()
	defer server.Close()

	resp, err := http.Get(server.URL + "/health")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
}

func TestCachePutAndGet(t *testing.T) {
	server := newTestServer()
	defer server.Close()

	putBody := `{"value": "test-value", "ttl_seconds": 60}`
	req, _ := http.NewRequest(http.MethodPut, server.URL+"/cache/test-key", nil)
	req.Body = jsonBody(putBody)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("PUT request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("expected status 201, got %d", resp.StatusCode)
	}

	resp, err = http.Get(server.URL + "/cache/test-key")
	if err != nil {
		t.Fatalf("GET request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	var result map[string]string
	json.NewDecoder(resp.Body).Decode(&result)
	if result["value"] != "test-value" {
		t.Errorf("expected value 'test-value', got '%s'", result["value"])
	}
}

func TestCacheMiss(t *testing.T) {
	server := newTestServer()
	defer server.Close()

	resp, err := http.Get(server.URL + "/cache/nonexistent")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", resp.StatusCode)
	}
}

func TestCachePutInvalidTTL(t *testing.T) {
	server := newTestServer()
	defer server.Close()

	testCases := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{"missing value", `{"ttl_seconds": 5}`, http.StatusBadRequest},
		{"zero ttl", `{"value": "v", "ttl_seconds": 0}`, http.StatusBadRequest},
		{"negative ttl", `{"value": "v", "ttl_seconds": -1}`, http.StatusBadRequest},
		{"malformed json", `{invalid}`, http.StatusBadRequest},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req, _ := http.NewRequest(http.MethodPut, server.URL+"/cache/test", jsonBody(tc.body))
			req.Header.Set("Content-Type", "application/json")

			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tc.wantStatus {
				t.Errorf("expected status %d, got %d", tc.wantStatus, resp.StatusCode)
			}
		})
	}
}

func TestCacheDelete(t *testing.T) {
	server := newTestServer()
	defer server.Close()

	putBody := `{"value": "test-value", "ttl_seconds": 60}`
	req, _ := http.NewRequest(http.MethodPut, server.URL+"/cache/delete-key", jsonBody(putBody))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("PUT request failed: %v", err)
	}
	defer resp.Body.Close()

	req, _ = http.NewRequest(http.MethodDelete, server.URL+"/cache/delete-key", nil)
	resp, err = client.Do(req)
	if err != nil {
		t.Fatalf("DELETE request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("expected status 204, got %d", resp.StatusCode)
	}

	resp, err = http.Get(server.URL + "/cache/delete-key")
	if err != nil {
		t.Fatalf("GET request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected status 404 after delete, got %d", resp.StatusCode)
	}
}

func TestCacheExpiry(t *testing.T) {
	server := newTestServer()
	defer server.Close()

	putBody := `{"value": "expire-me", "ttl_seconds": 1}`
	req, _ := http.NewRequest(http.MethodPut, server.URL+"/cache/expire-key", jsonBody(putBody))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("PUT request failed: %v", err)
	}
	defer resp.Body.Close()

	resp, err = http.Get(server.URL + "/cache/expire-key")
	if err != nil {
		t.Fatalf("GET request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200 before expiry, got %d", resp.StatusCode)
	}

	time.Sleep(1500 * time.Millisecond)

	resp, err = http.Get(server.URL + "/cache/expire-key")
	if err != nil {
		t.Fatalf("GET request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected status 404 after expiry, got %d", resp.StatusCode)
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

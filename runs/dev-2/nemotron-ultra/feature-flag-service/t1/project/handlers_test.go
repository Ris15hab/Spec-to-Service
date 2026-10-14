package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPutAndGetFlag(t *testing.T) {
	server := newServer()

	// PUT a new flag
	req := httptest.NewRequest(http.MethodPut, "/flags/beta", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Body = jsonBody(`{"enabled": true, "rollout_percent": 100}`)
	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated && rr.Code != http.StatusOK {
		t.Fatalf("PUT /flags/beta: expected 201 or 200, got %d", rr.Code)
	}

	// GET the flag
	req = httptest.NewRequest(http.MethodGet, "/flags/beta", nil)
	rr = httptest.NewRecorder()
	server.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("GET /flags/beta: expected 200, got %d", rr.Code)
	}

	var flag Flag
	if err := json.NewDecoder(rr.Body).Decode(&flag); err != nil {
		t.Fatalf("decode flag: %v", err)
	}
	if flag.Name != "beta" || !flag.Enabled || flag.RolloutPercent != 100 {
		t.Fatalf("unexpected flag: %+v", flag)
	}
}

func TestEvaluateFlag(t *testing.T) {
	server := newServer()

	// Create flag with 50% rollout
	req := httptest.NewRequest(http.MethodPut, "/flags/rollout50", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Body = jsonBody(`{"enabled": true, "rollout_percent": 50}`)
	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated && rr.Code != http.StatusOK {
		t.Fatalf("PUT flag: %d", rr.Code)
	}

	// Evaluate for a user
	req = httptest.NewRequest(http.MethodGet, "/evaluate?flag=rollout50&user=user123", nil)
	rr = httptest.NewRecorder()
	server.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("GET /evaluate: expected 200, got %d", rr.Code)
	}

	var result map[string]bool
	if err := json.NewDecoder(rr.Body).Decode(&result); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if _, ok := result["enabled"]; !ok {
		t.Fatalf("missing enabled field: %+v", result)
	}

	// Same user should get same result (deterministic)
	req = httptest.NewRequest(http.MethodGet, "/evaluate?flag=rollout50&user=user123", nil)
	rr = httptest.NewRecorder()
	server.ServeHTTP(rr, req)
	var result2 map[string]bool
	json.NewDecoder(rr.Body).Decode(&result2)
	if result["enabled"] != result2["enabled"] {
		t.Fatalf("evaluation not deterministic: %v vs %v", result, result2)
	}
}

func TestEvaluateErrors(t *testing.T) {
	server := newServer()

	// Missing flag param
	req := httptest.NewRequest(http.MethodGet, "/evaluate?user=u1", nil)
	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("missing flag: expected 400, got %d", rr.Code)
	}

	// Missing user param
	req = httptest.NewRequest(http.MethodGet, "/evaluate?flag=beta", nil)
	rr = httptest.NewRecorder()
	server.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("missing user: expected 400, got %d", rr.Code)
	}

	// Unknown flag
	req = httptest.NewRequest(http.MethodGet, "/evaluate?flag=unknown&user=u1", nil)
	rr = httptest.NewRecorder()
	server.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("unknown flag: expected 404, got %d", rr.Code)
	}
}

func TestListFlags(t *testing.T) {
	server := newServer()

	// Create two flags
	for _, name := range []string{"flag1", "flag2"} {
		req := httptest.NewRequest(http.MethodPut, "/flags/"+name, nil)
		req.Header.Set("Content-Type", "application/json")
		req.Body = jsonBody(`{"enabled": true, "rollout_percent": 100}`)
		rr := httptest.NewRecorder()
		server.ServeHTTP(rr, req)
		if rr.Code != http.StatusCreated && rr.Code != http.StatusOK {
			t.Fatalf("PUT %s: %d", name, rr.Code)
		}
	}

	// List flags
	req := httptest.NewRequest(http.MethodGet, "/flags", nil)
	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("GET /flags: expected 200, got %d", rr.Code)
	}

	var flags []Flag
	if err := json.NewDecoder(rr.Body).Decode(&flags); err != nil {
		t.Fatalf("decode flags: %v", err)
	}
	if len(flags) != 2 {
		t.Fatalf("expected 2 flags, got %d", len(flags))
	}
}

func TestInvalidRolloutPercent(t *testing.T) {
	server := newServer()

	req := httptest.NewRequest(http.MethodPut, "/flags/bad", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Body = jsonBody(`{"enabled": true, "rollout_percent": 150}`)
	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for rollout_percent > 100, got %d", rr.Code)
	}
}

func jsonBody(s string) *stringsReader {
	return &stringsReader{s: s}
}

type stringsReader struct {
	s string
	i int
}

func (r *stringsReader) Read(p []byte) (n int, err error) {
	if r.i >= len(r.s) {
		return 0, io.EOF
	}
	n = copy(p, r.s[r.i:])
	r.i += n
	return n, nil
}

func (r *stringsReader) Close() error { return nil }

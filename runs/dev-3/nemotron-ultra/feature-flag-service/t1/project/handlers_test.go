package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPutAndGetFlag(t *testing.T) {
	server := newServer()

	// Create a flag
	req := httptest.NewRequest(http.MethodPut, "/flags/beta", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Body = jsonBody(t, map[string]any{
		"enabled":         true,
		"rollout_percent": 50,
	})
	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated && rr.Code != http.StatusOK {
		t.Fatalf("expected 201 or 200, got %d", rr.Code)
	}

	// Get the flag
	req = httptest.NewRequest(http.MethodGet, "/flags/beta", nil)
	rr = httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var flag Flag
	if err := json.NewDecoder(rr.Body).Decode(&flag); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if flag.Name != "beta" || !flag.Enabled || flag.RolloutPercent != 50 {
		t.Errorf("unexpected flag: %+v", flag)
	}
}

func TestEvaluateFlag(t *testing.T) {
	server := newServer()

	// Create a flag with 100% rollout
	req := httptest.NewRequest(http.MethodPut, "/flags/beta", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Body = jsonBody(t, map[string]any{
		"enabled":         true,
		"rollout_percent": 100,
	})
	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated && rr.Code != http.StatusOK {
		t.Fatalf("create flag failed: %d", rr.Code)
	}

	// Evaluate for a user - should be enabled at 100%
	req = httptest.NewRequest(http.MethodGet, "/evaluate?flag=beta&user=u1", nil)
	rr = httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var result map[string]bool
	if err := json.NewDecoder(rr.Body).Decode(&result); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if !result["enabled"] {
		t.Error("expected enabled=true at 100% rollout")
	}

	// Test deterministic evaluation - same user should get same result
	req = httptest.NewRequest(http.MethodGet, "/evaluate?flag=beta&user=u1", nil)
	rr = httptest.NewRecorder()
	server.ServeHTTP(rr, req)
	if err := json.NewDecoder(rr.Body).Decode(&result); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if !result["enabled"] {
		t.Error("expected deterministic evaluation")
	}
}

func TestEvaluateMissingFlag(t *testing.T) {
	server := newServer()

	req := httptest.NewRequest(http.MethodGet, "/evaluate?flag=nope&user=u1", nil)
	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}
}

func TestEvaluateMissingParams(t *testing.T) {
	server := newServer()

	// Missing user
	req := httptest.NewRequest(http.MethodGet, "/evaluate?flag=beta", nil)
	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}

	// Missing flag
	req = httptest.NewRequest(http.MethodGet, "/evaluate?user=u1", nil)
	rr = httptest.NewRecorder()
	server.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestInvalidRolloutPercent(t *testing.T) {
	server := newServer()

	req := httptest.NewRequest(http.MethodPut, "/flags/bad", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Body = jsonBody(t, map[string]any{
		"enabled":         true,
		"rollout_percent": 150,
	})
	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestListFlags(t *testing.T) {
	server := newServer()

	// Create two flags
	for _, name := range []string{"flag1", "flag2"} {
		req := httptest.NewRequest(http.MethodPut, "/flags/"+name, nil)
		req.Header.Set("Content-Type", "application/json")
		req.Body = jsonBody(t, map[string]any{
			"enabled":         true,
			"rollout_percent": 50,
		})
		rr := httptest.NewRecorder()
		server.ServeHTTP(rr, req)
		if rr.Code != http.StatusCreated && rr.Code != http.StatusOK {
			t.Fatalf("create flag failed: %d", rr.Code)
		}
	}

	// List flags
	req := httptest.NewRequest(http.MethodGet, "/flags", nil)
	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var flags []Flag
	if err := json.NewDecoder(rr.Body).Decode(&flags); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if len(flags) != 2 {
		t.Errorf("expected 2 flags, got %d", len(flags))
	}
}

func jsonBody(t *testing.T, v any) *bytesReader {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	return &bytesReader{data: data}
}

type bytesReader struct {
	data []byte
	pos  int
}

func (b *bytesReader) Read(p []byte) (n int, err error) {
	if b.pos >= len(b.data) {
		return 0, errors.New("EOF")
	}
	n = copy(p, b.data[b.pos:])
	b.pos += n
	return n, nil
}

func (b *bytesReader) Close() error {
	return nil
}

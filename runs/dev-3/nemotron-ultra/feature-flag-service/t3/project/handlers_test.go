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

	// Create a flag
	req := httptest.NewRequest(http.MethodPut, "/flags/beta", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Body = jsonBody(t, map[string]any{
		"enabled":         true,
		"rollout_percent": 100,
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
		t.Fatalf("failed to decode response: %v", err)
	}

	if flag.Name != "beta" || !flag.Enabled || flag.RolloutPercent != 100 {
		t.Errorf("unexpected flag: %+v", flag)
	}
}

func TestEvaluateFlag(t *testing.T) {
	server := newServer()

	// Create a flag with 50% rollout
	req := httptest.NewRequest(http.MethodPut, "/flags/beta", nil)
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

	// Evaluate for a user
	req = httptest.NewRequest(http.MethodGet, "/evaluate?flag=beta&user=u1", nil)
	rr = httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var result map[string]bool
	if err := json.NewDecoder(rr.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if _, ok := result["enabled"]; !ok {
		t.Errorf("response missing 'enabled' field: %+v", result)
	}

	// Same user should get same result (deterministic)
	req = httptest.NewRequest(http.MethodGet, "/evaluate?flag=beta&user=u1", nil)
	rr = httptest.NewRecorder()
	server.ServeHTTP(rr, req)
	var result2 map[string]bool
	json.NewDecoder(rr.Body).Decode(&result2)
	if result["enabled"] != result2["enabled"] {
		t.Error("evaluation not deterministic for same user")
	}
}

func TestEvaluateErrors(t *testing.T) {
	server := newServer()

	// Missing flag param
	req := httptest.NewRequest(http.MethodGet, "/evaluate?user=u1", nil)
	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing flag, got %d", rr.Code)
	}

	// Missing user param
	req = httptest.NewRequest(http.MethodGet, "/evaluate?flag=beta", nil)
	rr = httptest.NewRecorder()
	server.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing user, got %d", rr.Code)
	}

	// Unknown flag
	req = httptest.NewRequest(http.MethodGet, "/evaluate?flag=nope&user=u1", nil)
	rr = httptest.NewRecorder()
	server.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404 for unknown flag, got %d", rr.Code)
	}
}

func TestInvalidFlagInput(t *testing.T) {
	server := newServer()

	// rollout_percent out of range
	req := httptest.NewRequest(http.MethodPut, "/flags/bad", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Body = jsonBody(t, map[string]any{
		"enabled":         true,
		"rollout_percent": 150,
	})
	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for rollout_percent > 100, got %d", rr.Code)
	}

	// missing enabled
	req = httptest.NewRequest(http.MethodPut, "/flags/bad2", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Body = jsonBody(t, map[string]any{
		"rollout_percent": 50,
	})
	rr = httptest.NewRecorder()
	server.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing enabled, got %d", rr.Code)
	}

	// malformed JSON
	req = httptest.NewRequest(http.MethodPut, "/flags/bad3", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Body = jsonBody(t, "not json")
	rr = httptest.NewRecorder()
	server.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for malformed JSON, got %d", rr.Code)
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
			"rollout_percent": 100,
		})
		rr := httptest.NewRecorder()
		server.ServeHTTP(rr, req)
		if rr.Code != http.StatusCreated && rr.Code != http.StatusOK {
			t.Fatalf("create %s failed: %d", name, rr.Code)
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
		t.Fatalf("failed to decode: %v", err)
	}

	if len(flags) != 2 {
		t.Errorf("expected 2 flags, got %d", len(flags))
	}
}

func jsonBody(t *testing.T, v any) *stringsReader {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return &stringsReader{data: data}
}

type stringsReader struct {
	data []byte
	pos  int
}

func (r *stringsReader) Read(p []byte) (n int, err error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n = copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}

func (r *stringsReader) Close() error { return nil }

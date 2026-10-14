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
	req.Body = jsonBody(t, map[string]any{
		"enabled":         true,
		"rollout_percent": 100,
	})
	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d", rr.Code)
	}

	// GET the flag
	req = httptest.NewRequest(http.MethodGet, "/flags/beta", nil)
	rr = httptest.NewRecorder()
	server.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	var flag Flag
	json.Unmarshal(rr.Body.Bytes(), &flag)
	if flag.Name != "beta" || !flag.Enabled || flag.RolloutPercent != 100 {
		t.Errorf("unexpected flag: %+v", flag)
	}
}

func TestPutFlagValidation(t *testing.T) {
	server := newServer()

	tests := []struct {
		name       string
		body       map[string]any
		wantStatus int
	}{
		{"rollout too high", map[string]any{"enabled": true, "rollout_percent": 150}, http.StatusBadRequest},
		{"rollout negative", map[string]any{"enabled": true, "rollout_percent": -1}, http.StatusBadRequest},
		{"missing enabled", map[string]any{"rollout_percent": 50}, http.StatusBadRequest},
		{"malformed JSON", nil, http.StatusBadRequest},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPut, "/flags/test", nil)
			req.Header.Set("Content-Type", "application/json")
			if tc.body != nil {
				req.Body = jsonBody(t, tc.body)
			} else {
				req.Body = jsonBody(t, "not json")
			}
			rr := httptest.NewRecorder()
			server.ServeHTTP(rr, req)
			if rr.Code != tc.wantStatus {
				t.Errorf("expected %d, got %d", tc.wantStatus, rr.Code)
			}
		})
	}
}

func TestEvaluateFlag(t *testing.T) {
	server := newServer()

	// Create flag with 50% rollout
	req := httptest.NewRequest(http.MethodPut, "/flags/beta", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Body = jsonBody(t, map[string]any{
		"enabled":         true,
		"rollout_percent": 50,
	})
	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("setup failed: %d", rr.Code)
	}

	// Evaluate for a user
	req = httptest.NewRequest(http.MethodGet, "/evaluate?flag=beta&user=u1", nil)
	rr = httptest.NewRecorder()
	server.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	var result map[string]bool
	json.Unmarshal(rr.Body.Bytes(), &result)
	if _, ok := result["enabled"]; !ok {
		t.Errorf("missing enabled in response: %+v", result)
	}

	// Deterministic: same user should get same result
	req = httptest.NewRequest(http.MethodGet, "/evaluate?flag=beta&user=u1", nil)
	rr = httptest.NewRecorder()
	server.ServeHTTP(rr, req)
	var result2 map[string]bool
	json.Unmarshal(rr.Body.Bytes(), &result2)
	if result["enabled"] != result2["enabled"] {
		t.Errorf("evaluation not deterministic: %v vs %v", result, result2)
	}

	// Unknown flag
	req = httptest.NewRequest(http.MethodGet, "/evaluate?flag=nope&user=u1", nil)
	rr = httptest.NewRecorder()
	server.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}

	// Missing params
	req = httptest.NewRequest(http.MethodGet, "/evaluate?flag=beta", nil)
	rr = httptest.NewRecorder()
	server.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestListFlags(t *testing.T) {
	server := newServer()

	// Add a couple flags
	for _, name := range []string{"flag1", "flag2"} {
		req := httptest.NewRequest(http.MethodPut, "/flags/"+name, nil)
		req.Header.Set("Content-Type", "application/json")
		req.Body = jsonBody(t, map[string]any{"enabled": true, "rollout_percent": 100})
		rr := httptest.NewRecorder()
		server.ServeHTTP(rr, req)
		if rr.Code != http.StatusCreated {
			t.Fatalf("setup failed: %d", rr.Code)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/flags", nil)
	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	var flags []Flag
	json.Unmarshal(rr.Body.Bytes(), &flags)
	if len(flags) != 2 {
		t.Errorf("expected 2 flags, got %d", len(flags))
	}
}

func jsonBody(t *testing.T, v any) *bytesReader {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return &bytesReader{bytes: b}
}

type bytesReader struct {
	bytes []byte
	pos   int
}

func (r *bytesReader) Read(p []byte) (n int, err error) {
	if r.pos >= len(r.bytes) {
		return 0, io.EOF
	}
	n = copy(p, r.bytes[r.pos:])
	r.pos += n
	return n, nil
}

func (r *bytesReader) Close() error { return nil }

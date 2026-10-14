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
	reqBody := `{"enabled": true, "rollout_percent": 50}`
	req := httptest.NewRequest(http.MethodPut, "/flags/beta", nil)
	req.Body = jsonBody(reqBody)
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d", w.Code)
	}

	var flag Flag
	if err := json.NewDecoder(w.Body).Decode(&flag); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if flag.Name != "beta" || !flag.Enabled || flag.RolloutPercent != 50 {
		t.Errorf("unexpected flag: %+v", flag)
	}

	// Get the flag
	req = httptest.NewRequest(http.MethodGet, "/flags/beta", nil)
	w = httptest.NewRecorder()
	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if err := json.NewDecoder(w.Body).Decode(&flag); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if flag.Name != "beta" || !flag.Enabled || flag.RolloutPercent != 50 {
		t.Errorf("unexpected flag: %+v", flag)
	}
}

func TestEvaluateFlag(t *testing.T) {
	server := newServer()

	// Create a flag with 100% rollout
	reqBody := `{"enabled": true, "rollout_percent": 100}`
	req := httptest.NewRequest(http.MethodPut, "/flags/beta", nil)
	req.Body = jsonBody(reqBody)
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create flag: expected 201, got %d", w.Code)
	}

	// Evaluate for a user - should be enabled at 100%
	req = httptest.NewRequest(http.MethodGet, "/evaluate?flag=beta&user=u1", nil)
	w = httptest.NewRecorder()
	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var result map[string]bool
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !result["enabled"] {
		t.Errorf("expected enabled=true at 100%% rollout")
	}

	// Test unknown flag
	req = httptest.NewRequest(http.MethodGet, "/evaluate?flag=nope&user=u1", nil)
	w = httptest.NewRecorder()
	server.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for unknown flag, got %d", w.Code)
	}

	// Test missing query params
	req = httptest.NewRequest(http.MethodGet, "/evaluate?flag=beta", nil)
	w = httptest.NewRecorder()
	server.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing user, got %d", w.Code)
	}
}

func TestListFlags(t *testing.T) {
	server := newServer()

	// Create multiple flags
	for _, name := range []string{"alpha", "beta", "gamma"} {
		reqBody := `{"enabled": true, "rollout_percent": 10}`
		req := httptest.NewRequest(http.MethodPut, "/flags/"+name, nil)
		req.Body = jsonBody(reqBody)
		w := httptest.NewRecorder()
		server.ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("create %s: expected 201, got %d", name, w.Code)
		}
	}

	// List flags
	req := httptest.NewRequest(http.MethodGet, "/flags", nil)
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var flags []Flag
	if err := json.NewDecoder(w.Body).Decode(&flags); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(flags) != 3 {
		t.Errorf("expected 3 flags, got %d", len(flags))
	}
}

func TestValidationErrors(t *testing.T) {
	server := newServer()

	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{"rollout too high", `{"enabled": true, "rollout_percent": 150}`, http.StatusBadRequest},
		{"rollout negative", `{"enabled": true, "rollout_percent": -1}`, http.StatusBadRequest},
		{"missing enabled", `{"rollout_percent": 50}`, http.StatusBadRequest},
		{"malformed JSON", `{invalid}`, http.StatusBadRequest},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPut, "/flags/test", nil)
			req.Body = jsonBody(tc.body)
			w := httptest.NewRecorder()
			server.ServeHTTP(w, req)
			if w.Code != tc.wantStatus {
				t.Errorf("expected %d, got %d", tc.wantStatus, w.Code)
			}
		})
	}
}

func TestDeterministicEvaluation(t *testing.T) {
	server := newServer()

	// Create flag with 50% rollout
	reqBody := `{"enabled": true, "rollout_percent": 50}`
	req := httptest.NewRequest(http.MethodPut, "/flags/rollout50", nil)
	req.Body = jsonBody(reqBody)
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create flag: %d", w.Code)
	}

	// Same user should always get same result
	user := "deterministic-user"
	var results []bool
	for i := 0; i < 10; i++ {
		req = httptest.NewRequest(http.MethodGet, "/evaluate?flag=rollout50&user="+user, nil)
		w = httptest.NewRecorder()
		server.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("evaluate: %d", w.Code)
		}
		var result map[string]bool
		json.NewDecoder(w.Body).Decode(&result)
		results = append(results, result["enabled"])
	}

	// All results should be identical
	for i := 1; i < len(results); i++ {
		if results[i] != results[0] {
			t.Errorf("evaluation not deterministic: got %v", results)
			break
		}
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
		return 0, errors.New("EOF")
	}
	n = copy(p, r.s[r.i:])
	r.i += n
	return n, nil
}

func (r *stringsReader) Close() error {
	return nil
}

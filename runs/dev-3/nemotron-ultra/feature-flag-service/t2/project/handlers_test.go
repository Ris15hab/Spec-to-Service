package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPutAndGetFlag(t *testing.T) {
	server := newServer()

	// Create a flag
	req := httptest.NewRequest(http.MethodPut, "/flags/beta", strings.NewReader(`{"enabled": true, "rollout_percent": 50}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d", rr.Code)
	}

	var flag Flag
	json.Unmarshal(rr.Body.Bytes(), &flag)
	if flag.Name != "beta" || !flag.Enabled || flag.RolloutPercent != 50 {
		t.Errorf("unexpected flag: %+v", flag)
	}

	// Get the flag
	req = httptest.NewRequest(http.MethodGet, "/flags/beta", nil)
	rr = httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}

	json.Unmarshal(rr.Body.Bytes(), &flag)
	if flag.Name != "beta" || !flag.Enabled || flag.RolloutPercent != 50 {
		t.Errorf("unexpected flag: %+v", flag)
	}
}

func TestEvaluateFlag(t *testing.T) {
	server := newServer()

	// Create a flag with 100% rollout
	req := httptest.NewRequest(http.MethodPut, "/flags/beta", strings.NewReader(`{"enabled": true, "rollout_percent": 100}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rr.Code)
	}

	// Evaluate for a user - should be enabled at 100%
	req = httptest.NewRequest(http.MethodGet, "/evaluate?flag=beta&user=u1", nil)
	rr = httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}

	var result map[string]bool
	json.Unmarshal(rr.Body.Bytes(), &result)
	if !result["enabled"] {
		t.Errorf("expected enabled=true at 100%% rollout")
	}

	// Test deterministic: same user should get same result
	req = httptest.NewRequest(http.MethodGet, "/evaluate?flag=beta&user=u1", nil)
	rr = httptest.NewRecorder()
	server.ServeHTTP(rr, req)
	json.Unmarshal(rr.Body.Bytes(), &result)
	if !result["enabled"] {
		t.Errorf("expected deterministic result for same user")
	}

	// Test unknown flag
	req = httptest.NewRequest(http.MethodGet, "/evaluate?flag=nope&user=u1", nil)
	rr = httptest.NewRecorder()
	server.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404 for unknown flag, got %d", rr.Code)
	}

	// Test missing query params
	req = httptest.NewRequest(http.MethodGet, "/evaluate?flag=beta", nil)
	rr = httptest.NewRecorder()
	server.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing user param, got %d", rr.Code)
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
		{"malformed JSON", `not json`, http.StatusBadRequest},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPut, "/flags/test", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()
			server.ServeHTTP(rr, req)
			if rr.Code != tc.wantStatus {
				t.Errorf("expected %d, got %d", tc.wantStatus, rr.Code)
			}
		})
	}
}

func TestListFlags(t *testing.T) {
	server := newServer()

	// Create two flags
	for _, name := range []string{"flag1", "flag2"} {
		req := httptest.NewRequest(http.MethodPut, "/flags/"+name, strings.NewReader(`{"enabled": true, "rollout_percent": 50}`))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		server.ServeHTTP(rr, req)
		if rr.Code != http.StatusCreated {
			t.Fatalf("failed to create %s: %d", name, rr.Code)
		}
	}

	// List flags
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

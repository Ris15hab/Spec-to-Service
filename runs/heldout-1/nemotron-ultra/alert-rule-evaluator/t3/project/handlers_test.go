package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreateAndListRules(t *testing.T) {
	server := newServer()

	// Create a rule
	reqBody := `{"name": "hot", "metric": "temp", "op": ">", "threshold": 80}`
	req := httptest.NewRequest(http.MethodPost, "/rules", nil)
	req.Body = jsonBody(reqBody)
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d", w.Code)
	}

	// List rules
	req = httptest.NewRequest(http.MethodGet, "/rules", nil)
	w = httptest.NewRecorder()
	server.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var rules []Rule
	if err := json.NewDecoder(w.Body).Decode(&rules); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if len(rules) != 1 || rules[0].Name != "hot" {
		t.Errorf("unexpected rules: %+v", rules)
	}
}

func TestCreateRuleValidation(t *testing.T) {
	server := newServer()

	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{"invalid op", `{"name": "bad", "metric": "temp", "op": "=", "threshold": 1}`, http.StatusBadRequest},
		{"empty name", `{"name": "", "metric": "temp", "op": ">", "threshold": 80}`, http.StatusBadRequest},
		{"empty metric", `{"name": "cold", "metric": "", "op": "<", "threshold": 10}`, http.StatusBadRequest},
		{"malformed JSON", `{invalid}`, http.StatusBadRequest},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/rules", nil)
			req.Body = jsonBody(tc.body)
			w := httptest.NewRecorder()
			server.ServeHTTP(w, req)
			if w.Code != tc.wantStatus {
				t.Errorf("expected %d, got %d", tc.wantStatus, w.Code)
			}
		})
	}
}

func TestDuplicateRuleConflict(t *testing.T) {
	server := newServer()

	body := `{"name": "hot", "metric": "temp", "op": ">", "threshold": 80}`

	// First creation
	req := httptest.NewRequest(http.MethodPost, "/rules", nil)
	req.Body = jsonBody(body)
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Errorf("first create: expected 201, got %d", w.Code)
	}

	// Duplicate creation
	req = httptest.NewRequest(http.MethodPost, "/rules", nil)
	req.Body = jsonBody(body)
	w = httptest.NewRecorder()
	server.ServeHTTP(w, req)
	if w.Code != http.StatusConflict {
		t.Errorf("duplicate create: expected 409, got %d", w.Code)
	}
}

func TestEvaluateRules(t *testing.T) {
	server := newServer()

	// Create rules
	rules := []string{
		`{"name": "hot", "metric": "temp", "op": ">", "threshold": 80}`,
		`{"name": "warm", "metric": "temp", "op": ">", "threshold": 60}`,
		`{"name": "cold", "metric": "temp", "op": "<", "threshold": 10}`,
		`{"name": "high_cpu", "metric": "cpu", "op": ">", "threshold": 90}`,
	}
	for _, r := range rules {
		req := httptest.NewRequest(http.MethodPost, "/rules", nil)
		req.Body = jsonBody(r)
		w := httptest.NewRecorder()
		server.ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("rule create failed: %d", w.Code)
		}
	}

	// Evaluate temp=95 (should fire hot and warm)
	req := httptest.NewRequest(http.MethodPost, "/evaluate", nil)
	req.Body = jsonBody(`{"metric": "temp", "value": 95}`)
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("evaluate: expected 200, got %d", w.Code)
	}

	var resp EvaluateResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	expected := []string{"hot", "warm"}
	if len(resp.Firing) != len(expected) {
		t.Errorf("expected %v, got %v", expected, resp.Firing)
	}
	for i := range expected {
		if resp.Firing[i] != expected[i] {
			t.Errorf("expected %v, got %v", expected, resp.Firing)
		}
	}

	// Evaluate temp=5 (should fire cold)
	req = httptest.NewRequest(http.MethodPost, "/evaluate", nil)
	req.Body = jsonBody(`{"metric": "temp", "value": 5}`)
	w = httptest.NewRecorder()
	server.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("evaluate: expected 200, got %d", w.Code)
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if len(resp.Firing) != 1 || resp.Firing[0] != "cold" {
		t.Errorf("expected [cold], got %v", resp.Firing)
	}

	// Evaluate cpu=95 (should fire high_cpu, not temp rules)
	req = httptest.NewRequest(http.MethodPost, "/evaluate", nil)
	req.Body = jsonBody(`{"metric": "cpu", "value": 95}`)
	w = httptest.NewRecorder()
	server.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("evaluate: expected 200, got %d", w.Code)
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if len(resp.Firing) != 1 || resp.Firing[0] != "high_cpu" {
		t.Errorf("expected [high_cpu], got %v", resp.Firing)
	}
}

func TestEvaluateValidation(t *testing.T) {
	server := newServer()

	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{"missing metric", `{"value": 95}`, http.StatusBadRequest},
		{"missing value", `{"metric": "temp"}`, http.StatusBadRequest},
		{"malformed JSON", `{invalid}`, http.StatusBadRequest},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/evaluate", nil)
			req.Body = jsonBody(tc.body)
			w := httptest.NewRecorder()
			server.ServeHTTP(w, req)
			if w.Code != tc.wantStatus {
				t.Errorf("expected %d, got %d", tc.wantStatus, w.Code)
			}
		})
	}
}

func jsonBody(s string) *stringsReader {
	return &stringsReader{s: s}
}

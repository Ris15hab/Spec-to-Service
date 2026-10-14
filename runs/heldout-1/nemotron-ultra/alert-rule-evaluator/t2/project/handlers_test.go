package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreateAndListRules(t *testing.T) {
	srv := newServer()

	// Create a rule
	reqBody := `{"name": "hot", "metric": "temp", "op": ">", "threshold": 80}`
	req := httptest.NewRequest(http.MethodPost, "/rules", nil)
	req.Body = jsonBody(reqBody)
	rr := httptest.NewRecorder()
	srv.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d", rr.Code)
	}

	// Duplicate name should return 409
	req = httptest.NewRequest(http.MethodPost, "/rules", nil)
	req.Body = jsonBody(reqBody)
	rr = httptest.NewRecorder()
	srv.ServeHTTP(rr, req)
	if rr.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d", rr.Code)
	}

	// Invalid op should return 400
	badBody := `{"name": "bad", "metric": "temp", "op": "=", "threshold": 1}`
	req = httptest.NewRequest(http.MethodPost, "/rules", nil)
	req.Body = jsonBody(badBody)
	rr = httptest.NewRecorder()
	srv.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}

	// List rules
	req = httptest.NewRequest(http.MethodGet, "/rules", nil)
	rr = httptest.NewRecorder()
	srv.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	var rules []Rule
	json.NewDecoder(rr.Body).Decode(&rules)
	if len(rules) != 1 || rules[0].Name != "hot" {
		t.Errorf("unexpected rules: %+v", rules)
	}
}

func TestEvaluate(t *testing.T) {
	srv := newServer()

	// Add rule first
	reqBody := `{"name": "hot", "metric": "temp", "op": ">", "threshold": 80}`
	req := httptest.NewRequest(http.MethodPost, "/rules", nil)
	req.Body = jsonBody(reqBody)
	rr := httptest.NewRecorder()
	srv.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("setup failed: %d", rr.Code)
	}

	// Evaluate with value that fires
	evalBody := `{"metric": "temp", "value": 95}`
	req = httptest.NewRequest(http.MethodPost, "/evaluate", nil)
	req.Body = jsonBody(evalBody)
	rr = httptest.NewRecorder()
	srv.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	var result map[string][]string
	json.NewDecoder(rr.Body).Decode(&result)
	if len(result["firing"]) != 1 || result["firing"][0] != "hot" {
		t.Errorf("expected firing [hot], got %v", result["firing"])
	}

	// Evaluate with value that doesn't fire
	evalBody = `{"metric": "temp", "value": 70}`
	req = httptest.NewRequest(http.MethodPost, "/evaluate", nil)
	req.Body = jsonBody(evalBody)
	rr = httptest.NewRecorder()
	srv.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	json.NewDecoder(rr.Body).Decode(&result)
	if len(result["firing"]) != 0 {
		t.Errorf("expected empty firing, got %v", result["firing"])
	}

	// Missing value should return 400
	evalBody = `{"metric": "temp"}`
	req = httptest.NewRequest(http.MethodPost, "/evaluate", nil)
	req.Body = jsonBody(evalBody)
	rr = httptest.NewRecorder()
	srv.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}

	// Missing metric should return 400
	evalBody = `{"value": 95}`
	req = httptest.NewRequest(http.MethodPost, "/evaluate", nil)
	req.Body = jsonBody(evalBody)
	rr = httptest.NewRecorder()
	srv.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestEvaluateIgnoresOtherMetrics(t *testing.T) {
	srv := newServer()

	// Add rules for different metrics
	rules := []string{
		`{"name": "hot", "metric": "temp", "op": ">", "threshold": 80}`,
		`{"name": "cold", "metric": "temp", "op": "<", "threshold": 10}`,
		`{"name": "high-pressure", "metric": "pressure", "op": ">", "threshold": 100}`,
	}
	for _, rb := range rules {
		req := httptest.NewRequest(http.MethodPost, "/rules", nil)
		req.Body = jsonBody(rb)
		rr := httptest.NewRecorder()
		srv.ServeHTTP(rr, req)
		if rr.Code != http.StatusCreated {
			t.Fatalf("setup failed: %d", rr.Code)
		}
	}

	// Evaluate temp=95 should only fire "hot", not "high-pressure"
	evalBody := `{"metric": "temp", "value": 95}`
	req := httptest.NewRequest(http.MethodPost, "/evaluate", nil)
	req.Body = jsonBody(evalBody)
	rr := httptest.NewRecorder()
	srv.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	var result map[string][]string
	json.NewDecoder(rr.Body).Decode(&result)
	if len(result["firing"]) != 1 || result["firing"][0] != "hot" {
		t.Errorf("expected firing [hot], got %v", result["firing"])
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

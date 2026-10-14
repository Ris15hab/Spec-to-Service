package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCreateAndListRules(t *testing.T) {
	srv := newServer()

	body := `{"name": "hot", "metric": "temp", "op": ">", "threshold": 80}`
	req := httptest.NewRequest(http.MethodPost, "/rules", strings.NewReader(body))
	rr := httptest.NewRecorder()
	srv.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d", rr.Code)
	}

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

func TestDuplicateRuleReturns409(t *testing.T) {
	srv := newServer()

	body := `{"name": "hot", "metric": "temp", "op": ">", "threshold": 80}`
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodPost, "/rules", strings.NewReader(body))
		rr := httptest.NewRecorder()
		srv.ServeHTTP(rr, req)
		if i == 0 {
			if rr.Code != http.StatusCreated {
				t.Errorf("first create: expected 201, got %d", rr.Code)
			}
		} else {
			if rr.Code != http.StatusConflict {
				t.Errorf("second create: expected 409, got %d", rr.Code)
			}
		}
	}
}

func TestEvaluateMetric(t *testing.T) {
	srv := newServer()

	rules := []string{
		`{"name": "hot", "metric": "temp", "op": ">", "threshold": 80}`,
		`{"name": "cold", "metric": "temp", "op": "<", "threshold": 10}`,
	}
	for _, r := range rules {
		req := httptest.NewRequest(http.MethodPost, "/rules", strings.NewReader(r))
		rr := httptest.NewRecorder()
		srv.ServeHTTP(rr, req)
		if rr.Code != http.StatusCreated {
			t.Errorf("create rule failed: %d", rr.Code)
		}
	}

	req := httptest.NewRequest(http.MethodPost, "/evaluate", strings.NewReader(`{"metric": "temp", "value": 95}`))
	rr := httptest.NewRecorder()
	srv.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("evaluate: expected 200, got %d", rr.Code)
	}
	var resp map[string][]string
	json.NewDecoder(rr.Body).Decode(&resp)
	if len(resp["firing"]) != 1 || resp["firing"][0] != "hot" {
		t.Errorf("unexpected firing: %+v", resp)
	}

	req = httptest.NewRequest(http.MethodPost, "/evaluate", strings.NewReader(`{"metric": "temp", "value": 5}`))
	rr = httptest.NewRecorder()
	srv.ServeHTTP(rr, req)
	json.NewDecoder(rr.Body).Decode(&resp)
	if len(resp["firing"]) != 1 || resp["firing"][0] != "cold" {
		t.Errorf("unexpected firing: %+v", resp)
	}

	req = httptest.NewRequest(http.MethodPost, "/evaluate", strings.NewReader(`{"metric": "temp", "value": 50}`))
	rr = httptest.NewRecorder()
	srv.ServeHTTP(rr, req)
	json.NewDecoder(rr.Body).Decode(&resp)
	if len(resp["firing"]) != 0 {
		t.Errorf("expected no firing, got %+v", resp)
	}
}

func TestEvaluateMissingFieldsReturns400(t *testing.T) {
	srv := newServer()

	req := httptest.NewRequest(http.MethodPost, "/evaluate", strings.NewReader(`{"metric": "temp"}`))
	rr := httptest.NewRecorder()
	srv.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}

	req = httptest.NewRequest(http.MethodPost, "/evaluate", strings.NewReader(`{"value": 10}`))
	rr = httptest.NewRecorder()
	srv.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestInvalidRuleReturns400(t *testing.T) {
	srv := newServer()

	invalidRules := []string{
		`{"name": "", "metric": "temp", "op": ">", "threshold": 80}`,
		`{"name": "hot", "metric": "", "op": ">", "threshold": 80}`,
		`{"name": "hot", "metric": "temp", "op": "=", "threshold": 80}`,
		`{"name": "hot", "metric": "temp", "op": ">", "threshold": "not a number"}`,
		`not json`,
	}
	for _, body := range invalidRules {
		req := httptest.NewRequest(http.MethodPost, "/rules", strings.NewReader(body))
		rr := httptest.NewRecorder()
		srv.ServeHTTP(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("body %q: expected 400, got %d", body, rr.Code)
		}
	}
}

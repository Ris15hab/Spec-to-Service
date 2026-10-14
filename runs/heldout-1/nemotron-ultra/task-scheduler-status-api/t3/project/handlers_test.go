package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

func TestCreateSchedule(t *testing.T) {
	server := newServer()

	tests := []struct {
		name           string
		body           string
		expectedStatus int
		checkBody      func(t *testing.T, body []byte)
	}{
		{
			name:           "valid schedule */15",
			body:           `{"name": "s1", "cron": "*/15 * * * *"}`,
			expectedStatus: http.StatusCreated,
			checkBody: func(t *testing.T, body []byte) {
				var resp map[string]string
				json.Unmarshal(body, &resp)
				if resp["name"] != "s1" || resp["cron"] != "*/15 * * * *" {
					t.Errorf("unexpected response: %s", string(body))
				}
			},
		},
		{
			name:           "valid schedule specific minute",
			body:           `{"name": "s2", "cron": "5 * * * *"}`,
			expectedStatus: http.StatusCreated,
			checkBody: func(t *testing.T, body []byte) {
				var resp map[string]string
				json.Unmarshal(body, &resp)
				if resp["name"] != "s2" || resp["cron"] != "5 * * * *" {
					t.Errorf("unexpected response: %s", string(body))
				}
			},
		},
		{
			name:           "invalid cron bogus",
			body:           `{"name": "s3", "cron": "bogus"}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid cron minute 61",
			body:           `{"name": "s4", "cron": "61 * * * *"}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "missing name",
			body:           `{"cron": "*/15 * * * *"}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "duplicate name",
			body:           `{"name": "s1", "cron": "5 * * * *"}`,
			expectedStatus: http.StatusConflict,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/schedules", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			server.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d: %s", tt.expectedStatus, w.Code, w.Body.String())
			}
			if tt.checkBody != nil {
				tt.checkBody(t, w.Body.Bytes())
			}
		})
	}
}

func TestGetNextRun(t *testing.T) {
	server := newServer()

	// First create a schedule
	req := httptest.NewRequest(http.MethodPost, "/schedules", strings.NewReader(`{"name": "s1", "cron": "*/15 * * * *"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("failed to create schedule: %s", w.Body.String())
	}

	tests := []struct {
		name           string
		path           string
		expectedStatus int
		checkBody      func(t *testing.T, body []byte)
	}{
		{
			name:           "valid from parameter */15",
			path:           "/schedules/s1/next?from=2026-01-01T00:07:00Z",
			expectedStatus: http.StatusOK,
			checkBody: func(t *testing.T, body []byte) {
				var resp map[string]string
				json.Unmarshal(body, &resp)
				expected := "2026-01-01T00:15:00Z"
				if resp["next"] != expected {
					t.Errorf("expected next=%s, got %s", expected, resp["next"])
				}
			},
		},
		{
			name:           "valid from parameter specific minute",
			path:           "/schedules/s1/next?from=2026-01-01T00:05:00Z",
			expectedStatus: http.StatusOK,
			checkBody: func(t *testing.T, body []byte) {
				var resp map[string]string
				json.Unmarshal(body, &resp)
				// s1 is */15, so from 00:05 next is 00:15
				expected := "2026-01-01T00:15:00Z"
				if resp["next"] != expected {
					t.Errorf("expected next=%s, got %s", expected, resp["next"])
				}
			},
		},
		{
			name:           "missing from parameter",
			path:           "/schedules/s1/next",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid from parameter",
			path:           "/schedules/s1/next?from=invalid",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "unknown schedule",
			path:           "/schedules/zzz/next?from=2026-01-01T00:07:00Z",
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			w := httptest.NewRecorder()
			server.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d: %s", tt.expectedStatus, w.Code, w.Body.String())
			}
			if tt.checkBody != nil {
				tt.checkBody(t, w.Body.Bytes())
			}
		})
	}
}

func TestListSchedules(t *testing.T) {
	server := newServer()

	var req *http.Request
	var w *httptest.ResponseRecorder

	// Create a couple schedules
	for _, body := range []string{
		`{"name": "s1", "cron": "*/15 * * * *"}`,
		`{"name": "s2", "cron": "5 * * * *"}`,
	} {
		req = httptest.NewRequest(http.MethodPost, "/schedules", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		server.ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("failed to create schedule: %s", w.Body.String())
		}
	}

	// List schedules
	req = httptest.NewRequest(http.MethodGet, "/schedules", nil)
	w = httptest.NewRecorder()
	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var schedules []map[string]string
	json.Unmarshal(w.Body.Bytes(), &schedules)
	if len(schedules) != 2 {
		t.Errorf("expected 2 schedules, got %d", len(schedules))
	}

	names := make(map[string]bool)
	for _, s := range schedules {
		names[s["name"]] = true
	}
	if !names["s1"] || !names["s2"] {
		t.Errorf("missing expected schedules: %v", schedules)
	}
}

func TestSpecificMinuteCron(t *testing.T) {
	server := newServer()

	// Create schedule with specific minute
	req := httptest.NewRequest(http.MethodPost, "/schedules", strings.NewReader(`{"name": "specific", "cron": "5 * * * *"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("failed to create schedule: %s", w.Body.String())
	}

	// Test: from 00:05 should give 01:05 (strictly after)
	req = httptest.NewRequest(http.MethodGet, "/schedules/specific/next?from=2026-01-01T00:05:00Z", nil)
	w = httptest.NewRecorder()
	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	expected := "2026-01-01T01:05:00Z"
	if resp["next"] != expected {
		t.Errorf("expected next=%s, got %s", expected, resp["next"])
	}

	// Test: from 00:04 should give 00:05
	req = httptest.NewRequest(http.MethodGet, "/schedules/specific/next?from=2026-01-01T00:04:00Z", nil)
	w = httptest.NewRecorder()
	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	json.Unmarshal(w.Body.Bytes(), &resp)
	expected = "2026-01-01T00:05:00Z"
	if resp["next"] != expected {
		t.Errorf("expected next=%s, got %s", expected, resp["next"])
	}
}

func TestStarCron(t *testing.T) {
	server := newServer()

	req := httptest.NewRequest(http.MethodPost, "/schedules", strings.NewReader(`{"name": "every", "cron": "* * * * *"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("failed to create schedule: %s", w.Body.String())
	}

	// Every minute - from 00:07:30 (truncated to 00:07) -> next is 00:08
	req = httptest.NewRequest(http.MethodGet, "/schedules/every/next?from=2026-01-01T00:07:30Z", nil)
	w = httptest.NewRecorder()
	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	expected := "2026-01-01T00:08:00Z"
	if resp["next"] != expected {
		t.Errorf("expected next=%s, got %s", expected, resp["next"])
	}
}

func TestConcurrentAccess(t *testing.T) {
	server := newServer()

	// Create initial schedule
	req := httptest.NewRequest(http.MethodPost, "/schedules", strings.NewReader(`{"name": "init", "cron": "* * * * *"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("failed to create initial schedule: %s", w.Body.String())
	}

	done := make(chan bool, 100)
	for i := 0; i < 50; i++ {
		go func(id int) {
			name := "concurrent" + strconv.Itoa(id)
			body := `{"name": "` + name + `", "cron": "*/5 * * * *"}`
			req := httptest.NewRequest(http.MethodPost, "/schedules", strings.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			server.ServeHTTP(w, req)
			if w.Code != http.StatusCreated && w.Code != http.StatusConflict {
				t.Errorf("unexpected status for %s: %d", name, w.Code)
			}
			done <- true
		}(i)
	}

	for i := 0; i < 50; i++ {
		go func(id int) {
			req := httptest.NewRequest(http.MethodGet, "/schedules/init/next?from=2026-01-01T00:00:00Z", nil)
			w := httptest.NewRecorder()
			server.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("unexpected status for read: %d", w.Code)
			}
			done <- true
		}(i)
	}

	for i := 0; i < 100; i++ {
		<-done
	}
}

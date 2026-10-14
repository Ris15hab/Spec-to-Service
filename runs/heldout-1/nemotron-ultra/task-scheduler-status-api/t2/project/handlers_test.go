package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCreateAndListSchedules(t *testing.T) {
	server := newServer()

	// Create a valid schedule
	reqBody := `{"name": "test-schedule", "cron": "*/15 * * * *"}`
	req := httptest.NewRequest(http.MethodPost, "/schedules", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var schedule Schedule
	if err := json.Unmarshal(w.Body.Bytes(), &schedule); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if schedule.Name != "test-schedule" || schedule.Cron != "*/15 * * * *" {
		t.Errorf("unexpected schedule: %+v", schedule)
	}

	// List schedules
	req = httptest.NewRequest(http.MethodGet, "/schedules", nil)
	w = httptest.NewRecorder()
	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var schedules []Schedule
	if err := json.Unmarshal(w.Body.Bytes(), &schedules); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(schedules) != 1 {
		t.Errorf("expected 1 schedule, got %d", len(schedules))
	}
}

func TestCreateScheduleValidation(t *testing.T) {
	server := newServer()

	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{"missing name", `{"name": "", "cron": "*/15 * * * *"}`, http.StatusBadRequest},
		{"invalid cron field count", `{"name": "s1", "cron": "* * * *"}`, http.StatusBadRequest},
		{"invalid minute field", `{"name": "s2", "cron": "bogus * * * *"}`, http.StatusBadRequest},
		{"minute out of range", `{"name": "s3", "cron": "61 * * * *"}`, http.StatusBadRequest},
		{"non-wildcard hour field", `{"name": "s4", "cron": "* 5 * * *"}`, http.StatusBadRequest},
		{"valid step", `{"name": "s5", "cron": "*/30 * * * *"}`, http.StatusCreated},
		{"valid specific minute", `{"name": "s6", "cron": "5 * * * *"}`, http.StatusCreated},
		{"valid wildcard minute", `{"name": "s7", "cron": "* * * * *"}`, http.StatusCreated},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/schedules", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			server.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("expected %d, got %d: %s", tt.wantStatus, w.Code, w.Body.String())
			}
		})
	}
}

func TestDuplicateSchedule(t *testing.T) {
	server := newServer()

	body := `{"name": "dup", "cron": "*/10 * * * *"}`

	// First creation
	req := httptest.NewRequest(http.MethodPost, "/schedules", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Errorf("first create: expected 201, got %d", w.Code)
	}

	// Duplicate creation
	req = httptest.NewRequest(http.MethodPost, "/schedules", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	server.ServeHTTP(w, req)
	if w.Code != http.StatusConflict {
		t.Errorf("duplicate create: expected 409, got %d", w.Code)
	}
}

func TestNextRunComputation(t *testing.T) {
	tests := []struct {
		name       string
		cron       string
		from       string
		wantStatus int
		wantNext   string
	}{
		{
			name:       "step 15 from 00:07",
			cron:       "*/15 * * * *",
			from:       "2026-01-01T00:07:00Z",
			wantStatus: http.StatusOK,
			wantNext:   "2026-01-01T00:15:00Z",
		},
		{
			name:       "specific minute 5 from 00:05",
			cron:       "5 * * * *",
			from:       "2026-01-01T00:05:00Z",
			wantStatus: http.StatusOK,
			wantNext:   "2026-01-01T01:05:00Z",
		},
		{
			name:       "missing from param",
			cron:       "*/15 * * * *",
			from:       "",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid from param",
			cron:       "*/15 * * * *",
			from:       "not-a-date",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "unknown schedule",
			cron:       "*/15 * * * *",
			from:       "2026-01-01T00:07:00Z",
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := newServer()

			// Create schedule for this test case
			if tt.cron != "" {
				body := `{"name": "s1", "cron": "` + tt.cron + `"}`
				req := httptest.NewRequest(http.MethodPost, "/schedules", strings.NewReader(body))
				req.Header.Set("Content-Type", "application/json")
				w := httptest.NewRecorder()
				server.ServeHTTP(w, req)
				if w.Code != http.StatusCreated {
					t.Fatalf("create failed: %d", w.Code)
				}
			}

			path := "/schedules/s1/next"
			if tt.name == "unknown schedule" {
				path = "/schedules/zzz/next"
			}
			if tt.from != "" {
				path += "?from=" + tt.from
			}

			req := httptest.NewRequest(http.MethodGet, path, nil)
			w := httptest.NewRecorder()
			server.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("expected %d, got %d: %s", tt.wantStatus, w.Code, w.Body.String())
				return
			}

			if tt.wantNext != "" {
				var resp struct {
					Next string `json:"next"`
				}
				if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}
				if resp.Next != tt.wantNext {
					t.Errorf("expected next=%s, got %s", tt.wantNext, resp.Next)
				}
			}
		})
	}
}

func TestMetricsIncrement(t *testing.T) {
	server := newServer()

	// Create schedule
	body := `{"name": "metrics-test", "cron": "5 * * * *"}`
	req := httptest.NewRequest(http.MethodPost, "/schedules", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create failed: %d", w.Code)
	}

	// Query next run
	req = httptest.NewRequest(http.MethodGet, "/schedules/metrics-test/next?from=2026-01-01T00:00:00Z", nil)
	w = httptest.NewRecorder()
	server.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("next run failed: %d", w.Code)
	}

	// Query again
	req = httptest.NewRequest(http.MethodGet, "/schedules/metrics-test/next?from=2026-01-01T01:00:00Z", nil)
	w = httptest.NewRecorder()
	server.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("second next run failed: %d", w.Code)
	}

	// Check metrics endpoint
	req = httptest.NewRequest(http.MethodGet, "/metrics", nil)
	w = httptest.NewRecorder()
	server.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("metrics failed: %d", w.Code)
	}

	metricsOutput := w.Body.String()
	if !strings.Contains(metricsOutput, "schedule_queries_total") {
		t.Errorf("schedule_queries_total not found in metrics")
	}
	if !strings.Contains(metricsOutput, "schedules_total") {
		t.Errorf("schedules_total not found in metrics")
	}
}

func TestHealthEndpoint(t *testing.T) {
	server := newServer()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

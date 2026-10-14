package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCreateSchedule(t *testing.T) {
	server := newServer()

	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{"valid schedule", `{"name": "s1", "cron": "*/15 * * * *"}`, http.StatusCreated},
		{"invalid cron", `{"name": "s2", "cron": "bogus"}`, http.StatusBadRequest},
		{"minute out of range", `{"name": "s3", "cron": "61 * * * *"}`, http.StatusBadRequest},
		{"missing name", `{"name": "", "cron": "5 * * * *"}`, http.StatusBadRequest},
		{"duplicate name", `{"name": "s1", "cron": "5 * * * *"}`, http.StatusConflict},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/schedules", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()
			server.ServeHTTP(rr, req)
			if rr.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rr.Code, tt.wantStatus)
			}
		})
	}
}

func TestListSchedules(t *testing.T) {
	server := newServer()

	req := httptest.NewRequest(http.MethodPost, "/schedules", strings.NewReader(`{"name": "s1", "cron": "*/15 * * * *"}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("create failed: %d", rr.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/schedules", nil)
	rr = httptest.NewRecorder()
	server.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusOK)
	}

	var schedules []Schedule
	if err := json.NewDecoder(rr.Body).Decode(&schedules); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(schedules) != 1 || schedules[0].Name != "s1" {
		t.Errorf("unexpected schedules: %+v", schedules)
	}
}

func TestNextRun(t *testing.T) {
	server := newServer()

	req := httptest.NewRequest(http.MethodPost, "/schedules", strings.NewReader(`{"name": "s1", "cron": "*/15 * * * *"}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("create failed: %d", rr.Code)
	}

	tests := []struct {
		name       string
		from       string
		wantStatus int
		wantNext   string
	}{
		{"valid from", "2026-01-01T00:07:00Z", http.StatusOK, "2026-01-01T00:15:00Z"},
		{"missing from", "", http.StatusBadRequest, ""},
		{"unknown schedule", "2026-01-01T00:07:00Z", http.StatusNotFound, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := "/schedules/s1/next"
			if tt.name == "unknown schedule" {
				path = "/schedules/zzz/next"
			}
			if tt.from != "" {
				path += "?from=" + tt.from
			}
			req := httptest.NewRequest(http.MethodGet, path, nil)
			rr := httptest.NewRecorder()
			server.ServeHTTP(rr, req)
			if rr.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rr.Code, tt.wantStatus)
			}
			if tt.wantNext != "" {
				var resp struct {
					Next string `json:"next"`
				}
				if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
					t.Fatalf("decode: %v", err)
				}
				if resp.Next != tt.wantNext {
					t.Errorf("next = %s, want %s", resp.Next, tt.wantNext)
				}
			}
		})
	}
}

func TestNextRunExactMinute(t *testing.T) {
	server := newServer()

	req := httptest.NewRequest(http.MethodPost, "/schedules", strings.NewReader(`{"name": "exact", "cron": "5 * * * *"}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("create failed: %d", rr.Code)
	}

	from := "2026-01-01T00:05:00Z"
	req = httptest.NewRequest(http.MethodGet, "/schedules/exact/next?from="+from, nil)
	rr = httptest.NewRecorder()
	server.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	var resp struct {
		Next string `json:"next"`
	}
	json.NewDecoder(rr.Body).Decode(&resp)
	if resp.Next != "2026-01-01T01:05:00Z" {
		t.Errorf("next = %s, want 2026-01-01T01:05:00Z", resp.Next)
	}
}

func TestMetricsIncrement(t *testing.T) {
	server := newServer()

	req := httptest.NewRequest(http.MethodPost, "/schedules", strings.NewReader(`{"name": "m1", "cron": "* * * * *"}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("create failed: %d", rr.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/schedules/m1/next?from=2026-01-01T00:00:00Z", nil)
	rr = httptest.NewRecorder()
	server.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("status = %d", rr.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rr = httptest.NewRecorder()
	server.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("metrics status = %d", rr.Code)
	}
	body := rr.Body.String()
	if !strings.Contains(body, "schedules_total") || !strings.Contains(body, "schedule_queries_total") {
		t.Errorf("metrics missing expected names: %s", body)
	}
}

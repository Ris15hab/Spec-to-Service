package main

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Schedule struct {
	Name string `json:"name"`
	Cron string `json:"cron"`
}

type Store struct {
	mu        sync.Mutex
	schedules map[string]Schedule
}

func NewStore() *Store {
	return &Store{
		schedules: make(map[string]Schedule),
	}
}

func (s *Store) Create(name, cron string) (Schedule, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.schedules[name]; exists {
		return Schedule{}, ErrConflict
	}

	schedule := Schedule{Name: name, Cron: cron}
	s.schedules[name] = schedule
	schedulesTotal.Set(float64(len(s.schedules)))
	return schedule, nil
}

func (s *Store) Get(name string) (Schedule, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	schedule, ok := s.schedules[name]
	return schedule, ok
}

func (s *Store) List() []Schedule {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := make([]Schedule, 0, len(s.schedules))
	for _, schedule := range s.schedules {
		result = append(result, schedule)
	}
	return result
}

var (
	ErrConflict   = &Error{Code: http.StatusConflict, Message: "schedule already exists"}
	ErrNotFound   = &Error{Code: http.StatusNotFound, Message: "schedule not found"}
	ErrBadRequest = &Error{Code: http.StatusBadRequest, Message: "bad request"}
)

type Error struct {
	Code    int
	Message string
}

func (e *Error) Error() string {
	return e.Message
}

func registerRoutes(mux *http.ServeMux) {
	store := NewStore()

	mux.HandleFunc("POST /schedules", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Name string `json:"name"`
			Cron string `json:"cron"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, ErrBadRequest.Message, ErrBadRequest.Code)
			return
		}

		if req.Name == "" {
			http.Error(w, "name is required", http.StatusBadRequest)
			return
		}

		if !isValidCron(req.Cron) {
			http.Error(w, "invalid cron expression", http.StatusBadRequest)
			return
		}

		schedule, err := store.Create(req.Name, req.Cron)
		if err != nil {
			if e, ok := err.(*Error); ok {
				http.Error(w, e.Message, e.Code)
			} else {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(schedule)
	})

	mux.HandleFunc("GET /schedules", func(w http.ResponseWriter, r *http.Request) {
		schedules := store.List()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(schedules)
	})

	mux.HandleFunc("GET /schedules/{name}/next", func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")
		fromStr := r.URL.Query().Get("from")

		if fromStr == "" {
			http.Error(w, "from query parameter is required", http.StatusBadRequest)
			return
		}

		from, err := time.Parse(time.RFC3339, fromStr)
		if err != nil {
			http.Error(w, "invalid from parameter, must be RFC3339", http.StatusBadRequest)
			return
		}

		schedule, ok := store.Get(name)
		if !ok {
			http.Error(w, ErrNotFound.Message, ErrNotFound.Code)
			return
		}

		next := computeNextRun(schedule.Cron, from)
		scheduleQueriesTotal.Inc()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"next": next.Format(time.RFC3339),
		})
	})
}

func isValidCron(cron string) bool {
	fields := strings.Fields(cron)
	if len(fields) != 5 {
		return false
	}

	// Fields 2-5 (hour, day of month, month, day of week) must be '*'
	for i := 1; i < 5; i++ {
		if fields[i] != "*" {
			return false
		}
	}

	// Field 0 (minute) can be '*', '*/N', or integer 0..59
	minute := fields[0]
	if minute == "*" {
		return true
	}

	if strings.HasPrefix(minute, "*/") {
		nStr := minute[2:]
		n, err := strconv.Atoi(nStr)
		if err != nil {
			return false
		}
		return n >= 1 && n <= 59
	}

	n, err := strconv.Atoi(minute)
	if err != nil {
		return false
	}
	return n >= 0 && n <= 59
}

func computeNextRun(cron string, from time.Time) time.Time {
	fields := strings.Fields(cron)
	minuteField := fields[0]

	// Zero out seconds and nanoseconds
	from = from.Truncate(time.Minute)

	// We need strictly after `from`, so start from next minute
	candidate := from.Add(time.Minute)

	if minuteField == "*" {
		return candidate
	}

	if strings.HasPrefix(minuteField, "*/") {
		nStr := minuteField[2:]
		n, _ := strconv.Atoi(nStr)
		// Find next minute that is a multiple of n
		minute := candidate.Minute()
		remainder := minute % n
		if remainder != 0 {
			candidate = candidate.Add(time.Duration(n-remainder) * time.Minute)
		}
		return candidate
	}

	// Specific minute
	targetMinute, _ := strconv.Atoi(minuteField)
	candidateMinute := candidate.Minute()

	if candidateMinute <= targetMinute {
		// Same hour
		candidate = time.Date(candidate.Year(), candidate.Month(), candidate.Day(),
			candidate.Hour(), targetMinute, 0, 0, time.UTC)
	} else {
		// Next hour
		candidate = candidate.Add(time.Hour)
		candidate = time.Date(candidate.Year(), candidate.Month(), candidate.Day(),
			candidate.Hour(), targetMinute, 0, 0, time.UTC)
	}

	return candidate
}

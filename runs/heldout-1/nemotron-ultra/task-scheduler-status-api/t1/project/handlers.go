package main

import (
	"encoding/json"
	"errors"
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

func (s *Store) Create(name, cron string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.schedules[name]; exists {
		return errors.New("conflict")
	}
	s.schedules[name] = Schedule{Name: name, Cron: cron}
	schedulesTotal.Inc()
	return nil
}

func (s *Store) Get(name string) (Schedule, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sch, ok := s.schedules[name]
	return sch, ok
}

func (s *Store) List() []Schedule {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := make([]Schedule, 0, len(s.schedules))
	for _, sch := range s.schedules {
		result = append(result, sch)
	}
	return result
}

func (s *Store) Count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.schedules)
}

func parseCron(cron string) (minuteSpec string, err error) {
	fields := strings.Fields(cron)
	if len(fields) != 5 {
		return "", errors.New("invalid cron: must have 5 fields")
	}
	for i := 1; i < 5; i++ {
		if fields[i] != "*" {
			return "", errors.New("invalid cron: fields 2-5 must be *")
		}
	}
	minute := fields[0]
	if minute == "*" {
		return "*", nil
	}
	if strings.HasPrefix(minute, "*/") {
		nStr := minute[2:]
		n, err := strconv.Atoi(nStr)
		if err != nil || n < 1 || n > 59 {
			return "", errors.New("invalid cron: step must be 1..59")
		}
		return minute, nil
	}
	n, err := strconv.Atoi(minute)
	if err != nil || n < 0 || n > 59 {
		return "", errors.New("invalid cron: minute must be 0..59")
	}
	return minute, nil
}

func computeNextRun(cron string, from time.Time) time.Time {
	minuteSpec, _ := parseCron(cron)
	from = from.UTC().Truncate(time.Minute)
	candidate := from.Add(time.Minute)

	for {
		candidateMinute := candidate.Minute()
		match := false
		if minuteSpec == "*" {
			match = true
		} else if strings.HasPrefix(minuteSpec, "*/") {
			n, _ := strconv.Atoi(minuteSpec[2:])
			if candidateMinute%n == 0 {
				match = true
			}
		} else {
			n, _ := strconv.Atoi(minuteSpec)
			if candidateMinute == n {
				match = true
			}
		}
		if match {
			return candidate
		}
		candidate = candidate.Add(time.Minute)
	}
}

func registerRoutes(mux *http.ServeMux) {
	store := NewStore()

	mux.HandleFunc("POST /schedules", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Name string `json:"name"`
			Cron string `json:"cron"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}
		if req.Name == "" {
			http.Error(w, "name is required", http.StatusBadRequest)
			return
		}
		if _, err := parseCron(req.Cron); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := store.Create(req.Name, req.Cron); err != nil {
			if err.Error() == "conflict" {
				http.Error(w, "schedule already exists", http.StatusConflict)
			} else {
				http.Error(w, err.Error(), http.StatusBadRequest)
			}
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"name": req.Name, "cron": req.Cron})
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
			http.Error(w, "invalid from parameter: must be RFC3339", http.StatusBadRequest)
			return
		}
		sch, ok := store.Get(name)
		if !ok {
			http.Error(w, "schedule not found", http.StatusNotFound)
			return
		}
		next := computeNextRun(sch.Cron, from)
		scheduleQueriesTotal.Inc()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"next": next.Format(time.RFC3339)})
	})
}

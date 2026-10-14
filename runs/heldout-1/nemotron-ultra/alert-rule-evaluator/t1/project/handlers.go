package main

import (
	"encoding/json"
	"net/http"
	"sort"
	"sync"
)

type Rule struct {
	Name      string  `json:"name"`
	Metric    string  `json:"metric"`
	Op        string  `json:"op"`
	Threshold float64 `json:"threshold"`
}

type ruleStore struct {
	mu    sync.Mutex
	rules map[string]Rule
}

func (s *ruleStore) addRule(r Rule) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.rules[r.Name]; exists {
		return http.StatusConflict, nil
	}
	s.rules[r.Name] = r
	alertRules.Set(float64(len(s.rules)))
	return http.StatusCreated, nil
}

func (s *ruleStore) listRules() []Rule {
	s.mu.Lock()
	defer s.mu.Unlock()

	rules := make([]Rule, 0, len(s.rules))
	for _, r := range s.rules {
		rules = append(rules, r)
	}
	sort.Slice(rules, func(i, j int) bool {
		return rules[i].Name < rules[j].Name
	})
	return rules
}

func (s *ruleStore) evaluate(metric string, value float64) []string {
	s.mu.Lock()
	defer s.mu.Unlock()

	var firing []string
	for _, r := range s.rules {
		if r.Metric != metric {
			continue
		}
		var ok bool
		switch r.Op {
		case ">":
			ok = value > r.Threshold
		case "<":
			ok = value < r.Threshold
		}
		if ok {
			firing = append(firing, r.Name)
		}
	}
	sort.Strings(firing)
	alertsFiring.Set(float64(len(firing)))
	alertEvaluationsTotal.Inc()
	return firing
}

func registerRoutes(mux *http.ServeMux) {
	store := &ruleStore{rules: make(map[string]Rule)}

	mux.HandleFunc("POST /rules", func(w http.ResponseWriter, r *http.Request) {
		var rule Rule
		if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
			http.Error(w, "malformed JSON", http.StatusBadRequest)
			return
		}
		if rule.Name == "" || rule.Metric == "" || (rule.Op != ">" && rule.Op != "<") {
			http.Error(w, "invalid field", http.StatusBadRequest)
			return
		}
		status, err := store.addRule(rule)
		if err != nil {
			http.Error(w, err.Error(), status)
			return
		}
		w.WriteHeader(status)
	})

	mux.HandleFunc("GET /rules", func(w http.ResponseWriter, r *http.Request) {
		rules := store.listRules()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(rules)
	})

	mux.HandleFunc("POST /evaluate", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Metric string   `json:"metric"`
			Value  *float64 `json:"value"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "malformed JSON", http.StatusBadRequest)
			return
		}
		if req.Metric == "" || req.Value == nil {
			http.Error(w, "metric and value required", http.StatusBadRequest)
			return
		}
		firing := store.evaluate(req.Metric, *req.Value)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string][]string{"firing": firing})
	})
}

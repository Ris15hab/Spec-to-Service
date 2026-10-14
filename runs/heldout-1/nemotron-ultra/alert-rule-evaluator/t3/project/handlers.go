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

type RuleStore struct {
	mu    sync.Mutex
	rules map[string]Rule
}

func NewRuleStore() *RuleStore {
	return &RuleStore{
		rules: make(map[string]Rule),
	}
}

func (s *RuleStore) AddRule(rule Rule) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.rules[rule.Name]; exists {
		return false, nil // 409 conflict
	}
	s.rules[rule.Name] = rule
	return true, nil
}

func (s *RuleStore) GetRules() []Rule {
	s.mu.Lock()
	defer s.mu.Unlock()

	rules := make([]Rule, 0, len(s.rules))
	for _, r := range s.rules {
		rules = append(rules, r)
	}
	return rules
}

func (s *RuleStore) GetRulesForMetric(metric string) []Rule {
	s.mu.Lock()
	defer s.mu.Unlock()

	var rules []Rule
	for _, r := range s.rules {
		if r.Metric == metric {
			rules = append(rules, r)
		}
	}
	return rules
}

func (s *RuleStore) Count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.rules)
}

type CreateRuleRequest struct {
	Name      string  `json:"name"`
	Metric    string  `json:"metric"`
	Op        string  `json:"op"`
	Threshold float64 `json:"threshold"`
}

type EvaluateRequest struct {
	Metric string  `json:"metric"`
	Value  float64 `json:"value"`
}

type EvaluateResponse struct {
	Firing []string `json:"firing"`
}

func registerRoutes(mux *http.ServeMux) {
	store := NewRuleStore()

	mux.HandleFunc("POST /rules", func(w http.ResponseWriter, r *http.Request) {
		var req CreateRuleRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "malformed JSON", http.StatusBadRequest)
			return
		}

		if req.Name == "" || req.Metric == "" || (req.Op != ">" && req.Op != "<") {
			http.Error(w, "invalid field", http.StatusBadRequest)
			return
		}

		rule := Rule{
			Name:      req.Name,
			Metric:    req.Metric,
			Op:        req.Op,
			Threshold: req.Threshold,
		}

		added, err := store.AddRule(rule)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		if !added {
			http.Error(w, "rule name exists", http.StatusConflict)
			return
		}

		alertRules.Set(float64(store.Count()))
		w.WriteHeader(http.StatusCreated)
	})

	mux.HandleFunc("GET /rules", func(w http.ResponseWriter, r *http.Request) {
		rules := store.GetRules()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(rules)
	})

	mux.HandleFunc("POST /evaluate", func(w http.ResponseWriter, r *http.Request) {
		var req EvaluateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "malformed JSON", http.StatusBadRequest)
			return
		}

		if req.Metric == "" {
			http.Error(w, "metric missing", http.StatusBadRequest)
			return
		}

		rules := store.GetRulesForMetric(req.Metric)
		var firing []string
		for _, rule := range rules {
			var matches bool
			if rule.Op == ">" {
				matches = req.Value > rule.Threshold
			} else if rule.Op == "<" {
				matches = req.Value < rule.Threshold
			}
			if matches {
				firing = append(firing, rule.Name)
			}
		}
		sort.Strings(firing)

		alertEvaluationsTotal.Inc()
		alertsFiring.Set(float64(len(firing)))

		resp := EvaluateResponse{Firing: firing}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})
}

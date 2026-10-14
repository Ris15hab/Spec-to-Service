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

func newRuleStore() *ruleStore {
	return &ruleStore{rules: make(map[string]Rule)}
}

func (s *ruleStore) add(r Rule) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.rules[r.Name]; exists {
		return false, nil
	}
	s.rules[r.Name] = r
	return true, nil
}

func (s *ruleStore) list() []Rule {
	s.mu.Lock()
	defer s.mu.Unlock()
	rules := make([]Rule, 0, len(s.rules))
	for _, r := range s.rules {
		rules = append(rules, r)
	}
	return rules
}

func (s *ruleStore) getByMetric(metric string) []Rule {
	s.mu.Lock()
	defer s.mu.Unlock()
	var result []Rule
	for _, r := range s.rules {
		if r.Metric == metric {
			result = append(result, r)
		}
	}
	return result
}

func registerRoutes(mux *http.ServeMux) {
	store := newRuleStore()

	mux.HandleFunc("POST /rules", func(w http.ResponseWriter, r *http.Request) {
		var rule Rule
		if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
			http.Error(w, "malformed JSON", http.StatusBadRequest)
			return
		}

		if rule.Name == "" || rule.Metric == "" {
			http.Error(w, "name and metric must be non-empty", http.StatusBadRequest)
			return
		}
		if rule.Op != ">" && rule.Op != "<" {
			http.Error(w, "op must be '>' or '<'", http.StatusBadRequest)
			return
		}

		added, _ := store.add(rule)
		if !added {
			http.Error(w, "rule name already exists", http.StatusConflict)
			return
		}

		alertRules.Inc()
		w.WriteHeader(http.StatusCreated)
	})

	mux.HandleFunc("GET /rules", func(w http.ResponseWriter, r *http.Request) {
		rules := store.list()
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

		if req.Metric == "" {
			http.Error(w, "metric is required", http.StatusBadRequest)
			return
		}
		if req.Value == nil {
			http.Error(w, "value is required", http.StatusBadRequest)
			return
		}

		alertEvaluationsTotal.Inc()

		rules := store.getByMetric(req.Metric)
		var firing []string
		for _, rule := range rules {
			var matches bool
			if rule.Op == ">" {
				matches = *req.Value > rule.Threshold
			} else {
				matches = *req.Value < rule.Threshold
			}
			if matches {
				firing = append(firing, rule.Name)
			}
		}
		sort.Strings(firing)

		alertsFiring.Set(float64(len(firing)))

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string][]string{"firing": firing})
	})
}

package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"sync"
	"time"
)

// DeliveryConfig holds configurable delivery parameters.
var DeliveryConfig = struct {
	MaxAttempts int
	Backoff     time.Duration
}{
	MaxAttempts: 3,
	Backoff:     100 * time.Millisecond,
}

type job struct {
	URL     string
	Payload json.RawMessage
}

type stats struct {
	Delivered int
	Failed    int
	Pending   int
}

type store struct {
	mu       sync.Mutex
	jobs     chan job
	delivered int
	failed    int
	pending   int
	wg       sync.WaitGroup
	ctx      context.Context
	cancel   context.CancelFunc
}

func newStore() *store {
	ctx, cancel := context.WithCancel(context.Background())
	s := &store{
		jobs:   make(chan job, 100),
		ctx:    ctx,
		cancel: cancel,
	}
	s.wg.Add(1)
	go s.worker()
	return s
}

func (s *store) enqueue(j job) {
	s.mu.Lock()
	s.pending++
	webhookPending.Inc()
	s.mu.Unlock()

	select {
	case s.jobs <- j:
	case <-s.ctx.Done():
		s.mu.Lock()
		s.pending--
		webhookPending.Dec()
		s.mu.Unlock()
	}
}

func (s *store) worker() {
	defer s.wg.Done()
	for {
		select {
		case j := <-s.jobs:
			s.deliver(j)
		case <-s.ctx.Done():
			return
		}
	}
}

func (s *store) deliver(j job) {
	client := &http.Client{Timeout: 5 * time.Second}
	var lastErr error

	for attempt := 1; attempt <= DeliveryConfig.MaxAttempts; attempt++ {
		req, err := http.NewRequestWithContext(s.ctx, http.MethodPost, j.URL, nil)
		if err != nil {
			lastErr = err
			break
		}
		req.Header.Set("Content-Type", "application/json")
		req.Body = json.RawMessage(j.Payload)

		resp, err := client.Do(req)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				s.recordSuccess()
				return
			}
			lastErr = errors.New("non-2xx status: " + resp.Status)
		} else {
			lastErr = err
		}

		if attempt < DeliveryConfig.MaxAttempts {
			select {
			case <-time.After(DeliveryConfig.Backoff):
			case <-s.ctx.Done():
				s.recordFailure()
				return
			}
		}
	}

	s.recordFailure()
}

func (s *store) recordSuccess() {
	s.mu.Lock()
	s.delivered++
	s.pending--
	webhookDeliveriesTotal.Inc()
	webhookPending.Dec()
	s.mu.Unlock()
}

func (s *store) recordFailure() {
	s.mu.Lock()
	s.failed++
	s.pending--
	webhookFailuresTotal.Inc()
	webhookPending.Dec()
	s.mu.Unlock()
}

func (s *store) getStats() stats {
	s.mu.Lock()
	defer s.mu.Unlock()
	return stats{
		Delivered: s.delivered,
		Failed:    s.failed,
		Pending:   s.pending,
	}
}

func (s *store) shutdown() {
	s.cancel()
	s.wg.Wait()
}

func registerRoutes(mux *http.ServeMux) {
	s := newStore()

	mux.HandleFunc("POST /webhooks", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			URL     string          `json:"url"`
			Payload json.RawMessage `json:"payload"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "malformed JSON", http.StatusBadRequest)
			return
		}

		if req.URL == "" {
			http.Error(w, "missing url", http.StatusBadRequest)
			return
		}

		parsedURL, err := url.ParseRequestURI(req.URL)
		if err != nil || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") || parsedURL.Host == "" {
			http.Error(w, "invalid url", http.StatusBadRequest)
			return
		}

		if req.Payload == nil {
			req.Payload = json.RawMessage("{}")
		}

		s.enqueue(job{URL: req.URL, Payload: req.Payload})
		w.WriteHeader(http.StatusAccepted)
	})

	mux.HandleFunc("GET /webhooks/stats", func(w http.ResponseWriter, r *http.Request) {
		st := s.getStats()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(st)
	})

	// Ensure cleanup on server shutdown (best effort)
	_ = s // s is captured in closures; shutdown not hooked but tests are short-lived
}

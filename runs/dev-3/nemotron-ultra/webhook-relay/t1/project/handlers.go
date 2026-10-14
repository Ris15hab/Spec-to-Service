package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"sync"
	"time"
)

// DeliveryBackoff is the base backoff duration between retry attempts.
// Configurable for tests (e.g., set to 10*time.Millisecond).
var DeliveryBackoff = 100 * time.Millisecond

// maxDeliveryAttempts is the total number of attempts (initial + retries).
const maxDeliveryAttempts = 3

type webhookJob struct {
	URL      string          `json:"url"`
	Payload  json.RawMessage `json:"payload"`
	Attempts int
}

type stats struct {
	mu        sync.Mutex
	delivered int64
	failed    int64
	pending   int64
}

func (s *stats) incPending() {
	s.mu.Lock()
	s.pending++
	s.mu.Unlock()
	webhookPending.Inc()
}

func (s *stats) decPending() {
	s.mu.Lock()
	s.pending--
	s.mu.Unlock()
	webhookPending.Dec()
}

func (s *stats) incDelivered() {
	s.mu.Lock()
	s.delivered++
	s.mu.Unlock()
	webhookDeliveriesTotal.Inc()
}

func (s *stats) incFailed() {
	s.mu.Lock()
	s.failed++
	s.mu.Unlock()
	webhookFailuresTotal.Inc()
}

func (s *stats) snapshot() (delivered, failed, pending int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.delivered, s.failed, s.pending
}

type store struct {
	mu   sync.Mutex
	jobs chan *webhookJob
	stats
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

func newStore() *store {
	ctx, cancel := context.WithCancel(context.Background())
	s := &store{
		jobs:   make(chan *webhookJob, 100),
		ctx:    ctx,
		cancel: cancel,
	}
	s.wg.Add(1)
	go s.worker()
	return s
}

func (s *store) worker() {
	defer s.wg.Done()
	for {
		select {
		case <-s.ctx.Done():
			return
		case job := <-s.jobs:
			s.deliver(job)
		}
	}
}

func (s *store) deliver(job *webhookJob) {
	client := &http.Client{Timeout: 5 * time.Second}
	var lastErr error

	for attempt := 1; attempt <= maxDeliveryAttempts; attempt++ {
		job.Attempts = attempt
		req, err := http.NewRequestWithContext(s.ctx, http.MethodPost, job.URL, bytes.NewReader(job.Payload))
		if err != nil {
			lastErr = err
			continue
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			s.backoff(attempt)
			continue
		}
		resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			s.incDelivered()
			s.decPending()
			return
		}

		lastErr = errors.New("non-2xx status: " + resp.Status)
		s.backoff(attempt)
	}

	s.incFailed()
	s.decPending()
	_ = lastErr
}

func (s *store) backoff(attempt int) {
	select {
	case <-s.ctx.Done():
		return
	case <-time.After(DeliveryBackoff * time.Duration(attempt)):
	}
}

func (s *store) enqueue(job *webhookJob) {
	s.incPending()
	select {
	case s.jobs <- job:
	case <-s.ctx.Done():
		s.decPending()
	}
}

func (s *store) shutdown() {
	s.cancel()
	s.wg.Wait()
	close(s.jobs)
	for range s.jobs {
		s.decPending()
	}
}

func registerRoutes(mux *http.ServeMux) {
	st := newStore()

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

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
		u, err := url.ParseRequestURI(req.URL)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
			http.Error(w, "invalid url", http.StatusBadRequest)
			return
		}
		if req.Payload == nil {
			req.Payload = json.RawMessage("{}")
		}
		st.enqueue(&webhookJob{URL: req.URL, Payload: req.Payload})
		w.WriteHeader(http.StatusAccepted)
	})

	mux.HandleFunc("GET /webhooks/stats", func(w http.ResponseWriter, r *http.Request) {
		d, f, p := st.snapshot()
		resp := map[string]int64{
			"delivered": d,
			"failed":    f,
			"pending":   p,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})
}

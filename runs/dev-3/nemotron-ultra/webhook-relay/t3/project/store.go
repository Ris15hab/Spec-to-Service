package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"sync"
	"time"
)

// DeliveryBackoff is the delay between retry attempts.
// Configurable for tests (default 100ms).
var DeliveryBackoff = 100 * time.Millisecond

// Job represents a webhook delivery job.
type Job struct {
	URL      string          `json:"url"`
	Payload  json.RawMessage `json:"payload"`
	Attempts int             `json:"attempts"`
}

// Store holds the job queue and statistics.
type Store struct {
	mu        sync.Mutex
	jobs      chan *Job
	delivered int64
	failed    int64
	pending   int64
	workerWg  sync.WaitGroup
	ctx       context.Context
	cancel    context.CancelFunc
	client    *http.Client
}

// NewStore creates a new store with a background worker.
func NewStore() *Store {
	ctx, cancel := context.WithCancel(context.Background())
	s := &Store{
		jobs:   make(chan *Job, 100),
		ctx:    ctx,
		cancel: cancel,
		client: &http.Client{Timeout: 5 * time.Second},
	}
	s.workerWg.Add(1)
	go s.worker()
	return s
}

// Enqueue adds a job to the queue.
func (s *Store) Enqueue(job *Job) {
	s.mu.Lock()
	s.pending++
	webhookPending.Inc()
	s.mu.Unlock()

	select {
	case s.jobs <- job:
	case <-s.ctx.Done():
		s.mu.Lock()
		s.pending--
		webhookPending.Dec()
		s.mu.Unlock()
	}
}

// Stats returns current statistics.
func (s *Store) Stats() (delivered, failed, pending int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.delivered, s.failed, s.pending
}

// Shutdown stops the worker.
func (s *Store) Shutdown() {
	s.cancel()
	close(s.jobs)
	s.workerWg.Wait()
}

func (s *Store) worker() {
	defer s.workerWg.Done()
	for {
		select {
		case job, ok := <-s.jobs:
			if !ok {
				return
			}
			s.deliver(job)
		case <-s.ctx.Done():
			return
		}
	}
}

func (s *Store) deliver(job *Job) {
	maxAttempts := 3
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		job.Attempts = attempt
		err := s.doDelivery(job)
		if err == nil {
			s.recordSuccess()
			return
		}
		// Wait before retry (except on last attempt)
		if attempt < maxAttempts {
			select {
			case <-time.After(DeliveryBackoff):
			case <-s.ctx.Done():
				s.recordFailure()
				return
			}
		}
	}
	s.recordFailure()
}

func (s *Store) doDelivery(job *Job) error {
	req, err := http.NewRequestWithContext(s.ctx, http.MethodPost, job.URL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Body = &nopCloser{bytes: job.Payload}

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &non2xxError{status: resp.StatusCode}
	}
	return nil
}

func (s *Store) recordSuccess() {
	s.mu.Lock()
	s.delivered++
	s.pending--
	webhookDeliveriesTotal.Inc()
	webhookPending.Dec()
	s.mu.Unlock()
}

func (s *Store) recordFailure() {
	s.mu.Lock()
	s.failed++
	s.pending--
	webhookFailuresTotal.Inc()
	webhookPending.Dec()
	s.mu.Unlock()
}

// nopCloser wraps a byte slice as an io.ReadCloser.
type nopCloser struct {
	bytes []byte
	pos   int
}

func (n *nopCloser) Read(p []byte) (int, error) {
	if n.pos >= len(n.bytes) {
		return 0, io.EOF
	}
	nBytes := copy(p, n.bytes[n.pos:])
	n.pos += nBytes
	return nBytes, nil
}

func (n *nopCloser) Close() error { return nil }

type non2xxError struct {
	status int
}

func (e *non2xxError) Error() string {
	return "non-2xx status: " + http.StatusText(e.status)
}

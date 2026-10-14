package main

import (
	"sync"
	"time"
)

// DeliveryConfig holds configurable delivery parameters.
type DeliveryConfig struct {
	MaxAttempts int
	Backoff     time.Duration
}

// DefaultDeliveryConfig provides sensible defaults.
var DefaultDeliveryConfig = DeliveryConfig{
	MaxAttempts: 3,
	Backoff:     100 * time.Millisecond,
}

// WebhookJob represents a queued webhook delivery.
type WebhookJob struct {
	URL     string
	Payload any
}

// Store holds the in-memory state for webhook deliveries.
type Store struct {
	mu           sync.Mutex
	jobs         chan *WebhookJob
	delivered    int64
	failed       int64
	pending      int64
	config       DeliveryConfig
	workerDone   chan struct{}
	shutdownOnce sync.Once
	deliverFunc  func(*WebhookJob) error
}

// NewStore creates a new store with the given configuration.
func NewStore(config DeliveryConfig) *Store {
	s := &Store{
		jobs:       make(chan *WebhookJob, 1000),
		config:     config,
		workerDone: make(chan struct{}),
	}
	go s.worker()
	return s
}

// Enqueue adds a job to the queue and updates metrics.
func (s *Store) Enqueue(job *WebhookJob) {
	s.mu.Lock()
	s.pending++
	webhookPending.Set(float64(s.pending))
	s.mu.Unlock()

	s.jobs <- job
}

// Stats returns current delivery statistics.
func (s *Store) Stats() (delivered, failed, pending int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.delivered, s.failed, s.pending
}

// worker processes jobs from the queue.
func (s *Store) worker() {
	defer close(s.workerDone)
	for job := range s.jobs {
		s.deliver(job)
	}
}

// deliver attempts to deliver a webhook with retries.
func (s *Store) deliver(job *WebhookJob) {
	var lastErr error
	for attempt := 1; attempt <= s.config.MaxAttempts; attempt++ {
		err := s.doDelivery(job)
		if err == nil {
			s.recordSuccess()
			return
		}
		lastErr = err
		if attempt < s.config.MaxAttempts {
			time.Sleep(s.config.Backoff)
		}
	}
	s.recordFailure(lastErr)
}

// doDelivery performs a single delivery attempt.
func (s *Store) doDelivery(job *WebhookJob) error {
	return s.deliverFunc(job)
}

// recordSuccess increments delivered counter and decrements pending.
func (s *Store) recordSuccess() {
	s.mu.Lock()
	s.delivered++
	s.pending--
	webhookDeliveriesTotal.Inc()
	webhookPending.Set(float64(s.pending))
	s.mu.Unlock()
}

// recordFailure increments failed counter and decrements pending.
func (s *Store) recordFailure(err error) {
	s.mu.Lock()
	s.failed++
	s.pending--
	webhookFailuresTotal.Inc()
	webhookPending.Set(float64(s.pending))
	s.mu.Unlock()
}

// Shutdown gracefully stops the worker.
func (s *Store) Shutdown() {
	s.shutdownOnce.Do(func() {
		close(s.jobs)
		<-s.workerDone
	})
}

package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"
)

// DeliveryBackoff is the base backoff duration between retry attempts.
// Configurable for tests (set to <= 100ms).
var DeliveryBackoff = 50 * time.Millisecond

// WebhookRequest represents the incoming webhook request body.
type WebhookRequest struct {
	URL     string          `json:"url"`
	Payload json.RawMessage `json:"payload"`
}

// Stats holds delivery statistics.
type Stats struct {
	Delivered int `json:"delivered"`
	Failed    int `json:"failed"`
	Pending   int `json:"pending"`
}

// Store holds the in-memory state for webhook deliveries.
type Store struct {
	mu        sync.Mutex
	delivered int
	failed    int
	pending   int
}

// NewStore creates a new Store.
func NewStore() *Store {
	return &Store{}
}

// IncrementPending atomically increments the pending counter and updates the gauge.
func (s *Store) IncrementPending() {
	s.mu.Lock()
	s.pending++
	s.mu.Unlock()
	webhookPending.Inc()
}

// DecrementPending atomically decrements the pending counter and updates the gauge.
func (s *Store) DecrementPending() {
	s.mu.Lock()
	s.pending--
	s.mu.Unlock()
	webhookPending.Dec()
}

// IncrementDelivered atomically increments the delivered counter.
func (s *Store) IncrementDelivered() {
	s.mu.Lock()
	s.delivered++
	s.mu.Unlock()
	webhookDeliveriesTotal.Inc()
}

// IncrementFailed atomically increments the failed counter.
func (s *Store) IncrementFailed() {
	s.mu.Lock()
	s.failed++
	s.mu.Unlock()
	webhookFailuresTotal.Inc()
}

// GetStats returns a snapshot of current statistics.
func (s *Store) GetStats() Stats {
	s.mu.Lock()
	defer s.mu.Unlock()
	return Stats{
		Delivered: s.delivered,
		Failed:    s.failed,
		Pending:   s.pending,
	}
}

// ValidateURL checks if the URL is a valid absolute HTTP/HTTPS URL.
func ValidateURL(rawURL string) (*url.URL, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, ErrInvalidScheme
	}
	if u.Host == "" {
		return nil, ErrMissingHost
	}
	return u, nil
}

// ErrInvalidScheme is returned when URL scheme is not http or https.
var ErrInvalidScheme = &urlError{"invalid scheme, must be http or https"}

// ErrMissingHost is returned when URL has no host.
var ErrMissingHost = &urlError{"missing host"}

type urlError struct {
	msg string
}

func (e *urlError) Error() string {
	return e.msg
}

// DeliverWebhook attempts to deliver the payload to the target URL with retries.
func (s *Store) DeliverWebhook(targetURL string, payload json.RawMessage) {
	defer s.DecrementPending()

	client := &http.Client{Timeout: 5 * time.Second}
	var lastErr error

	for attempt := 1; attempt <= 3; attempt++ {
		req, err := http.NewRequest(http.MethodPost, targetURL, bytes.NewReader(payload))
		if err != nil {
			lastErr = err
			break
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
		} else {
			resp.Body.Close()
			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				s.IncrementDelivered()
				return
			}
			lastErr = &statusError{status: resp.StatusCode}
		}

		if attempt < 3 {
			time.Sleep(DeliveryBackoff * time.Duration(attempt))
		}
	}

	s.IncrementFailed()
	_ = lastErr // avoid unused variable when not logging
}

type statusError struct {
	status int
}

func (e *statusError) Error() string {
	return "non-2xx status: " + strconv.Itoa(e.status)
}

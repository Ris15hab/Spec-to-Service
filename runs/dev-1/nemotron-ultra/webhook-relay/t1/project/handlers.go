package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
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

// WebhookJob represents a queued webhook delivery.
type WebhookJob struct {
	URL     string
	Payload any
}

// Stats holds delivery statistics.
type Stats struct {
	Delivered int `json:"delivered"`
	Failed    int `json:"failed"`
	Pending   int `json:"pending"`
}

// webhookState holds the in-memory state for webhook deliveries.
type webhookState struct {
	mu        sync.Mutex
	delivered int
	failed    int
	pending   int
	queue     chan WebhookJob
	done      chan struct{}
	wg        sync.WaitGroup
}

var state = &webhookState{
	queue: make(chan WebhookJob, 100),
	done:  make(chan struct{}),
}

// registerRoutes registers the webhook-related routes.
func registerRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /webhooks", handlePostWebhooks)
	mux.HandleFunc("GET /webhooks/stats", handleGetWebhookStats)

	// Start the delivery worker
	state.wg.Add(1)
	go deliveryWorker()
}

// handlePostWebhooks handles POST /webhooks requests.
func handlePostWebhooks(w http.ResponseWriter, r *http.Request) {
	var req struct {
		URL     string `json:"url"`
		Payload any    `json:"payload"`
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

	// Queue the job
	select {
	case state.queue <- WebhookJob{URL: req.URL, Payload: req.Payload}:
		state.mu.Lock()
		state.pending++
		webhookPending.Inc()
		state.mu.Unlock()
		w.WriteHeader(http.StatusAccepted)
	default:
		http.Error(w, "queue full", http.StatusServiceUnavailable)
	}
}

// handleGetWebhookStats handles GET /webhooks/stats requests.
func handleGetWebhookStats(w http.ResponseWriter, r *http.Request) {
	state.mu.Lock()
	stats := Stats{
		Delivered: state.delivered,
		Failed:    state.failed,
		Pending:   state.pending,
	}
	state.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// deliveryWorker processes webhook deliveries from the queue.
func deliveryWorker() {
	defer state.wg.Done()

	for {
		select {
		case job := <-state.queue:
			deliverWithRetry(job)
		case <-state.done:
			return
		}
	}
}

// deliverWithRetry attempts to deliver a webhook with retries.
func deliverWithRetry(job WebhookJob) {
	var lastErr error

	for attempt := 1; attempt <= DeliveryConfig.MaxAttempts; attempt++ {
		err := deliver(job)
		if err == nil {
			// Success
			state.mu.Lock()
			state.delivered++
			state.pending--
			webhookDeliveriesTotal.Inc()
			webhookPending.Dec()
			state.mu.Unlock()
			return
		}

		lastErr = err

		// Don't sleep after the last attempt
		if attempt < DeliveryConfig.MaxAttempts {
			time.Sleep(DeliveryConfig.Backoff)
		}
	}

	// All attempts failed
	state.mu.Lock()
	state.failed++
	state.pending--
	webhookFailuresTotal.Inc()
	webhookPending.Dec()
	state.mu.Unlock()

	_ = lastErr // suppress unused variable warning
}

// deliver performs a single delivery attempt.
func deliver(job WebhookJob) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	body, err := json.Marshal(job.Payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, job.URL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Body = &nopCloser{bytes: body}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return errors.New("non-2xx response")
	}

	return nil
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
	copied := copy(p, n.bytes[n.pos:])
	n.pos += copied
	return copied, nil
}

func (n *nopCloser) Close() error {
	return nil
}

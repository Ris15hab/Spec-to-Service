package main

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// registerRoutes registers the webhook relay routes on the given mux.
func registerRoutes(mux *http.ServeMux) {
	store := NewStore(DefaultDeliveryConfig)
	store.deliverFunc = doHTTPDelivery

	mux.HandleFunc("POST /webhooks", func(w http.ResponseWriter, r *http.Request) {
		handleCreateWebhook(w, r, store)
	})
	mux.HandleFunc("GET /webhooks/stats", func(w http.ResponseWriter, r *http.Request) {
		handleStats(w, r, store)
	})
}

// handleCreateWebhook handles POST /webhooks requests.
func handleCreateWebhook(w http.ResponseWriter, r *http.Request, store *Store) {
	if r.Header.Get("Content-Type") != "application/json" {
		http.Error(w, `{"error": "Content-Type must be application/json"}`, http.StatusBadRequest)
		return
	}

	var req struct {
		URL     string `json:"url"`
		Payload any    `json:"payload"`
	}

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		http.Error(w, `{"error": "malformed JSON"}`, http.StatusBadRequest)
		return
	}

	if req.URL == "" {
		http.Error(w, `{"error": "url is required"}`, http.StatusBadRequest)
		return
	}

	parsedURL, err := url.ParseRequestURI(req.URL)
	if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
		http.Error(w, `{"error": "invalid url"}`, http.StatusBadRequest)
		return
	}

	if !strings.EqualFold(parsedURL.Scheme, "http") && !strings.EqualFold(parsedURL.Scheme, "https") {
		http.Error(w, `{"error": "url must be http or https"}`, http.StatusBadRequest)
		return
	}

	job := &WebhookJob{
		URL:     req.URL,
		Payload: req.Payload,
	}

	store.Enqueue(job)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{"status": "queued"})
}

// handleStats handles GET /webhooks/stats requests.
func handleStats(w http.ResponseWriter, r *http.Request, store *Store) {
	delivered, failed, pending := store.Stats()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]int64{
		"delivered": delivered,
		"failed":    failed,
		"pending":   pending,
	})
}

// doHTTPDelivery performs the actual HTTP delivery of a webhook.
func doHTTPDelivery(job *WebhookJob) error {
	body, err := json.Marshal(job.Payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", job.URL, strings.NewReader(string(body)))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &non2xxError{statusCode: resp.StatusCode}
	}

	return nil
}

type non2xxError struct {
	statusCode int
}

func (e *non2xxError) Error() string {
	return "non-2xx response: " + http.StatusText(e.statusCode)
}

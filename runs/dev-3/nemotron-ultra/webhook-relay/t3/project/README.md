# Webhook Relay Service

A simple HTTP service that accepts webhook requests and forwards them to a target URL with automatic retries.

## Endpoints

- `GET /health` - Health check endpoint
- `GET /metrics` - Prometheus metrics endpoint
- `POST /webhooks` - Accept a webhook for delivery
  - Request body: `{"url": "https://example.com/webhook", "payload": {"key": "value"}}`
  - Returns `202 Accepted` when queued for delivery
  - Returns `400 Bad Request` for missing/invalid URL or malformed JSON
- `GET /webhooks/stats` - Get delivery statistics
  - Returns `{"delivered": n, "failed": n, "pending": n}`

## Metrics

- `webhook_deliveries_total` - Counter of successful deliveries
- `webhook_failures_total` - Counter of deliveries that failed after all retries
- `webhook_pending` - Gauge of deliveries queued or in progress

## Behavior

- Delivery is asynchronous: the webhook is queued and processed in the background
- Retries up to 3 attempts total on network error or non-2xx response
- Configurable backoff between retries (default 100ms, set via `DeliveryBackoff` variable)
- Payload is forwarded as JSON with `Content-Type: application/json`

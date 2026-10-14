# Webhook Relay Service

A simple HTTP service that accepts webhook requests and forwards them to a target URL with automatic retries.

## Endpoints

- `GET /health` - Health check endpoint
- `GET /metrics` - Prometheus metrics endpoint
- `POST /webhooks` - Accept a webhook for delivery
- `GET /webhooks/stats` - Get delivery statistics

## Metrics

- `webhook_deliveries_total` - Counter of successful deliveries
- `webhook_failures_total` - Counter of deliveries that failed after all retries
- `webhook_pending` - Gauge of deliveries queued or in progress

## Webhook Delivery

Webhooks are delivered asynchronously with the following behavior:

- POSTs the payload as JSON to the target URL
- Retries up to 3 attempts total on network error or non-2xx response
- Exponential backoff (configurable, default 100ms)
- Returns 202 Accepted when queued for delivery
- Returns 400 for missing/invalid URL or malformed JSON

## Configuration

Delivery behavior can be tuned via the `DeliveryConfig` package variable:

```go
DeliveryConfig.MaxAttempts = 3  // Total attempts (initial + retries)
DeliveryConfig.Backoff = 100 * time.Millisecond  // Delay between retries
```

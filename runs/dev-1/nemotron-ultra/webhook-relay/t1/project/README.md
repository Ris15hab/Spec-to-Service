# Webhook Relay Service

A simple HTTP service that accepts webhook requests and forwards them to a target URL with retries.

## Endpoints

- `GET /health` - Health check endpoint
- `GET /metrics` - Prometheus metrics endpoint
- `POST /webhooks` - Accept a webhook for delivery
- `GET /webhooks/stats` - Get delivery statistics

## Metrics

- `webhook_deliveries_total` - Counter of successful deliveries
- `webhook_failures_total` - Counter of deliveries that failed after all retries
- `webhook_pending` - Gauge of deliveries queued or in progress

## Usage

### POST /webhooks

Accepts a JSON body with:
- `url` (required): Absolute HTTP/HTTPS URL to deliver the webhook to
- `payload` (optional): Any JSON value to send as the request body

Returns:
- `202 Accepted` - Webhook queued for delivery
- `400 Bad Request` - Missing/invalid URL or malformed JSON

Example:
```bash
curl -X POST http://localhost:8080/webhooks \
  -H "Content-Type: application/json" \
  -d '{"url": "https://example.com/webhook", "payload": {"event": "test"}}'
```

### GET /webhooks/stats

Returns delivery statistics:
```json
{
  "delivered": 10,
  "failed": 2,
  "pending": 1
}
```

## Delivery Behavior

- Deliveries are asynchronous
- Retries up to 3 attempts (configurable via `DeliveryConfig.MaxAttempts`)
- Retries on network errors or non-2xx responses
- Exponential backoff (configurable via `DeliveryConfig.Backoff`, default 100ms)

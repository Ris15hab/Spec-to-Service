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
- `webhook_pending` - Gauge of deliveries currently queued or in progress

## Webhook Delivery

The `POST /webhooks` endpoint accepts a JSON body with:
- `url` (required): Absolute HTTP/HTTPS URL to deliver the webhook to
- `payload` (optional): JSON payload to send (defaults to `{}`)

Returns `202 Accepted` when the webhook is queued for delivery.

Delivery is asynchronous with up to 3 total attempts (initial + 2 retries) on network errors or non-2xx responses, with exponential backoff.

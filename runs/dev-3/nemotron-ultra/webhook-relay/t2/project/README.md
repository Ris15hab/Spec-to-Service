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

The `POST /webhooks` endpoint accepts a JSON body with:
- `url` (required): Absolute HTTP or HTTPS URL to deliver the webhook to
- `payload` (optional): Any JSON value to send as the request body

Returns `202 Accepted` when the webhook is queued for delivery.

Delivery is asynchronous with up to 3 attempts (configurable) and exponential backoff (default 100ms, configurable via `DefaultDeliveryConfig.Backoff`). Retries occur on network errors or non-2xx responses.

## Example Usage

```bash
# Queue a webhook
curl -X POST http://localhost:8080/webhooks \
  -H "Content-Type: application/json" \
  -d '{"url": "https://example.com/webhook", "payload": {"event": "user.created", "user_id": 123}}'

# Check delivery stats
curl http://localhost:8080/webhooks/stats
```

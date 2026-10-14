# Webhook Relay Service

A simple HTTP service that accepts webhook requests and forwards them to a target URL with automatic retries.

## Endpoints

- `GET /health` - Health check endpoint, returns "ok"
- `GET /metrics` - Prometheus metrics endpoint
- `POST /webhooks` - Accept a webhook for delivery
- `GET /webhooks/stats` - Get delivery statistics

## Webhook Request Format

```json
{
  "url": "https://example.com/webhook",
  "payload": { "any": "json" }
}
```

- `url` (required): Absolute HTTP or HTTPS URL
- `payload` (optional): Any valid JSON object, defaults to `{}`

## Responses

- `202 Accepted` - Webhook queued for delivery
- `400 Bad Request` - Missing/invalid URL or malformed JSON

## Statistics Response

```json
{
  "delivered": 10,
  "failed": 2,
  "pending": 1
}
```

## Delivery Behavior

- Delivery is asynchronous (returns 202 immediately)
- Posts payload as JSON to the target URL
- Retries up to 3 attempts total on network error or non-2xx response
- Exponential backoff between attempts (configurable via `DeliveryBackoff` variable, default 100ms)
- Metrics tracked via Prometheus counters/gauges

## Metrics

- `webhook_deliveries_total` (counter) - Successful deliveries
- `webhook_failures_total` (counter) - Deliveries that failed after all retries
- `webhook_pending` (gauge) - Deliveries queued or in progress

## Running

```bash
go run .
```

Server starts on `:8080` by default.

## Testing

```bash
go test -v
```

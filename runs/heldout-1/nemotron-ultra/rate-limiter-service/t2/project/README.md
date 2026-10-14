# Rate Limiter Service

A token-bucket rate limiter keyed by client.

## Endpoints

- `GET /health`
- `GET /metrics`
- `POST /check`
- `GET /buckets/{key}`

## Metrics

- `ratelimit_allowed_total`
- `ratelimit_denied_total`
- `ratelimit_buckets`

# Rate Limiter Service

A token-bucket rate limiter keyed by client identifier.

## Endpoints

- `GET /health` - Health check endpoint
- `GET /metrics` - Prometheus metrics endpoint
- `POST /check` - Check and consume a token for a key
- `GET /buckets/{key}` - Get current token count for a key

## Metrics

- `ratelimit_allowed_total` - Counter of allowed checks
- `ratelimit_denied_total` - Counter of denied checks
- `ratelimit_buckets` - Gauge of known keys

## Rate Limiting Behavior

Each key has an independent token bucket with:
- Capacity: 5 tokens
- Refill rate: 1 token per second
- Starts full (5 tokens)

When a request is made to `POST /check` with a key:
- If tokens available: consumes 1 token, returns `200 {allowed: true, remaining: n}`
- If bucket empty: returns `429 {allowed: false, remaining: 0}`
- Invalid/missing key: returns `400`

The `GET /buckets/{key}` endpoint returns the current token count (with real-time refill calculation) or `404` if the key has never been checked.

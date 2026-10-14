# Rate Limiter Service

A token-bucket rate limiter keyed by client, implemented in Go.

## Endpoints

- `GET /health`
- `GET /metrics`
- `POST /check`
- `GET /buckets/{key}`

## Metrics

- `ratelimit_allowed_total`
- `ratelimit_denied_total`
- `ratelimit_buckets`

## Usage

### POST /check

Check and consume a token for a given key.

**Request:**
```json
{"key": "client-1"}
```

**Response (200 - allowed):**
```json
{"allowed": true, "remaining": 4}
```

**Response (429 - rate limited):**
```json
{"allowed": false, "remaining": 0}
```

**Response (400 - bad request):**
```json
{"error": "key is required"}
```

Each key has an independent bucket with capacity 5 tokens that refills at 1 token per second.

### GET /buckets/{key}

Get current token count for a key.

**Response (200):**
```json
{"key": "client-1", "tokens": 4.5}
```

**Response (404):**
Key has never been checked.

### GET /health

Health check endpoint.

### GET /metrics

Prometheus metrics endpoint.

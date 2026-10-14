# TTL Cache Service

An in-memory key-value cache with TTL (Time-To-Live) expiration, built with Go 1.22+ standard library.

## Features

- **In-memory storage** with automatic expiration
- **Lazy expiration** on read operations
- **Background sweeper** cleans expired entries every second
- **Prometheus metrics** for monitoring
- **Thread-safe** concurrent access

## Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/health` | Health check endpoint |
| GET | `/metrics` | Prometheus metrics endpoint |
| PUT | `/cache/{key}` | Store a value with TTL |
| GET | `/cache/{key}` | Retrieve a value |
| DELETE | `/cache/{key}` | Delete a value |

### PUT /cache/{key}

Request body:
```json
{
  "value": "string",
  "ttl_seconds": 60
}
```

- `value` (required): The string value to store
- `ttl_seconds` (required): Time-to-live in seconds, must be > 0

Responses:
- `201 Created` - Value stored successfully
- `400 Bad Request` - Missing value, invalid TTL, or malformed JSON

### GET /cache/{key}

Responses:
- `200 OK` - Returns `{"key": "...", "value": "..."}`
- `404 Not Found` - Key doesn't exist or has expired

### DELETE /cache/{key}

Responses:
- `204 No Content` - Deleted successfully
- `404 Not Found` - Key doesn't exist

## Metrics

| Name | Type | Description |
|------|------|-------------|
| `cache_entries` | Gauge | Current number of live (unexpired) entries |
| `cache_hits_total` | Counter | Total successful reads |
| `cache_misses_total` | Counter | Total reads of absent or expired keys |

## Running

```bash
go run .
```

Server starts on `:8080` by default.

## Testing

```bash
go test -v
```

## Example Usage

```bash
# Store a value
curl -X PUT http://localhost:8080/cache/mykey \
  -H "Content-Type: application/json" \
  -d '{"value": "hello", "ttl_seconds": 30}'

# Retrieve the value
curl http://localhost:8080/cache/mykey

# Delete the value
curl -X DELETE http://localhost:8080/cache/mykey

# Check metrics
curl http://localhost:8080/metrics
```

# TTL Cache Service

An in-memory key-value cache where entries expire after a configurable TTL.

## Endpoints

- `GET /health` - Health check endpoint
- `GET /metrics` - Prometheus metrics endpoint
- `PUT /cache/{key}` - Store a value with TTL
- `GET /cache/{key}` - Retrieve a value
- `DELETE /cache/{key}` - Delete a value

## Metrics

- `cache_entries` - Gauge of live (unexpired) entries
- `cache_hits_total` - Counter of successful reads
- `cache_misses_total` - Counter of reads of absent or expired keys

## API Details

### PUT /cache/{key}

Store a value with a time-to-live.

**Request Body:**
```json
{
  "value": "string",
  "ttl_seconds": 60
}
```

**Responses:**
- `201 Created` - Value stored successfully
- `400 Bad Request` - Missing value, invalid TTL, or malformed JSON

### GET /cache/{key}

Retrieve a value by key.

**Responses:**
- `200 OK` - Returns `{"key": "...", "value": "..."}`
- `404 Not Found` - Key not found or expired

### DELETE /cache/{key}

Delete a value by key.

**Responses:**
- `204 No Content` - Deleted successfully
- `404 Not Found` - Key not found

## Behavior

- Entries expire after their TTL
- Expiry is evaluated lazily on read and by a background sweeper
- The `cache_entries` metric stays accurate through both mechanisms
- All operations are thread-safe

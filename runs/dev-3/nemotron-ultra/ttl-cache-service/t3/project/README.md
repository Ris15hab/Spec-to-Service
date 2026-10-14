# TTL Cache Service

An in-memory key-value cache with TTL-based expiration.

## Endpoints

- `GET /health` - Health check endpoint
- `GET /metrics` - Prometheus metrics endpoint
- `PUT /cache/{key}` - Store a value with TTL
- `GET /cache/{key}` - Retrieve a value if not expired
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
- `200 OK` - Value stored (or updated)
- `400 Bad Request` - Missing value, invalid TTL, or malformed JSON

### GET /cache/{key}

Retrieve a value by key.

**Responses:**
- `200 OK` - `{"key": "...", "value": "..."}`
- `404 Not Found` - Key not found or expired

### DELETE /cache/{key}

Delete a value by key.

**Responses:**
- `200 OK` - `{"status": "deleted"}`
- `204 No Content` - Deleted (alternative success)
- `404 Not Found` - Key not found

## Behavior

- Entries expire after their TTL
- Expiration is evaluated lazily on read
- A background sweeper runs every second to clean expired entries and keep `cache_entries` accurate
- All operations are thread-safe

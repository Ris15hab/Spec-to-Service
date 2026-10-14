# TTL Cache Service

An in-memory key-value cache where entries expire after a configurable TTL.

## Endpoints

- `GET /health`
- `GET /metrics`
- `PUT /cache/{key}`
- `GET /cache/{key}`
- `DELETE /cache/{key}`

## Metrics

- `cache_entries`
- `cache_hits_total`
- `cache_misses_total`

## API Details

### PUT /cache/{key}

Store a value with a TTL.

**Request Body:**
```json
{
  "value": "string",
  "ttl_seconds": 60
}
```

**Responses:**
- `201 Created` - Value stored successfully
- `400 Bad Request` - Missing value, ttl_seconds <= 0, or malformed JSON

### GET /cache/{key}

Retrieve a value by key.

**Responses:**
- `200 OK` - `{"key": "...", "value": "..."}`
- `404 Not Found` - Key absent or expired

### DELETE /cache/{key}

Delete a key from the cache.

**Responses:**
- `204 No Content` - Key deleted
- `404 Not Found` - Key absent

## Behavior

- Expiry is evaluated lazily on read AND by a background sweeper (runs every second)
- The `cache_entries` gauge reflects live (unexpired) entries
- All operations are thread-safe

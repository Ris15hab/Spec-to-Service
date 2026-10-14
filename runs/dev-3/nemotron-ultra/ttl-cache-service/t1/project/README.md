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

Retrieve a value if it exists and hasn't expired.

**Responses:**
- `200 OK` - Returns `{"key": "...", "value": "..."}`
- `404 Not Found` - Key doesn't exist or has expired

### DELETE /cache/{key}

Delete a value.

**Responses:**
- `204 No Content` - Deleted successfully
- `404 Not Found` - Key doesn't exist

### GET /health

Health check endpoint.

**Response:**
- `200 OK`

### GET /metrics

Prometheus metrics endpoint.

**Response:**
- `200 OK` - Prometheus format metrics

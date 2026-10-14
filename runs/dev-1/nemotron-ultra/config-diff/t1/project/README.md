# config-diff

A simple HTTP service that compares two flat configuration maps and returns a structured diff.

## Endpoints

- `GET /health` - Health check endpoint
- `GET /metrics` - Prometheus metrics endpoint
- `POST /diff` - Compare two configuration objects

### POST /diff

**Request Body:**
```json
{
  "old": {"key": "value", ...},
  "new": {"key": "value", ...}
}
```

Both `old` and `new` must be JSON objects with string keys and string values.

**Response (200 OK):**
```json
{
  "added": {"key": "value", ...},
  "removed": {"key": "value", ...},
  "changed": {"key": {"old": "value", "new": "value"}, ...}
}
```

- `added`: Keys present in `new` but not in `old`
- `removed`: Keys present in `old` but not in `new`
- `changed`: Keys present in both but with different values

**Error Responses:**
- `400 Bad Request` - Invalid JSON, missing `old` or `new`, or either is not an object

## Metrics

- `config_diffs_total` - Counter of diff requests served
- `config_diff_changes_last` - Gauge of number of differing keys in the most recent diff (added + removed + changed)

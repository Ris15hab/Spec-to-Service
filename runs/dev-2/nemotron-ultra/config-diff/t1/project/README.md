# config-diff

A simple HTTP service that compares two flat configuration maps and returns a structured diff.

## Endpoints

- `GET /health` - Health check endpoint
- `GET /metrics` - Prometheus metrics endpoint
- `POST /diff` - Compare two configuration objects

## Metrics

- `config_diffs_total` - Counter of diff requests served
- `config_diff_changes_last` - Gauge of number of differing keys in the most recent diff

## Usage

### POST /diff

Request body:
```json
{
  "old": {"key1": "value1", "key2": "value2"},
  "new": {"key1": "value1", "key3": "value3"}
}
```

Response (200 OK):
```json
{
  "added": {"key3": "value3"},
  "removed": {"key2": "value2"},
  "changed": {}
}
```

Error response (400 Bad Request):
```json
{"error": "old and new must be objects"}
```

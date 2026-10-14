# Disk Usage Exporter

A simple HTTP service that exposes filesystem disk usage statistics as JSON and Prometheus metrics.

## Endpoints

- `GET /health` - Health check endpoint
- `GET /metrics` - Prometheus metrics endpoint
- `GET /disk` - Get disk usage for a specific path (requires `path` query parameter)

## Metrics

- `disk_used_percent` - Used percent of the root filesystem (gauge)
- `disk_free_bytes` - Free bytes of the root filesystem (gauge)

## Usage

### Get disk usage for a path

```bash
curl "http://localhost:8080/disk?path=/"
```

Response:
```json
{
  "path": "/",
  "total_bytes": 1000000000,
  "free_bytes": 500000000,
  "used_percent": 50.0
}
```

### Prometheus metrics

```bash
curl http://localhost:8080/metrics
```

The service automatically updates the root filesystem (`/`) metrics every 10 seconds in the background.

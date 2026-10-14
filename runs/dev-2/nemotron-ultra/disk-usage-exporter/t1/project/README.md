# disk-usage-exporter

A service that reads filesystem statistics and exposes them over HTTP and as Prometheus metrics.

## Endpoints

- `GET /health` - Health check endpoint
- `GET /metrics` - Prometheus metrics endpoint
- `GET /disk` - Get disk usage statistics for a given path
  - Query parameter: `path` (required) - The filesystem path to query

## Metrics

- `disk_used_percent` - Used percent of the root filesystem (gauge)
- `disk_free_bytes` - Free bytes of the root filesystem (gauge)

## Behavior

A background goroutine refreshes the metrics for the root filesystem (`/`) every 10 seconds, and once at startup.

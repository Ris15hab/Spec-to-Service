# disk-usage-exporter

A service that reads filesystem statistics and exposes them over HTTP and as Prometheus metrics.

## Endpoints

- `GET /health` - Health check endpoint
- `GET /metrics` - Prometheus metrics endpoint
- `GET /disk` - Get disk usage for a path (requires `path` query parameter)

## Metrics

- `disk_used_percent` - Used percent of the root filesystem
- `disk_free_bytes` - Free bytes of the root filesystem

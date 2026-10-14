# disk-usage-exporter

A service that reads filesystem statistics and exposes them over HTTP and as Prometheus metrics.

## Endpoints

- `GET /health`
- `GET /metrics`
- `GET /disk` — query parameter `path` (required)

## Metrics

- `disk_used_percent`
- `disk_free_bytes`

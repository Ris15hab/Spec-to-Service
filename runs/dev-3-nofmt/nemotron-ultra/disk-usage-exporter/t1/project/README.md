# Disk Usage Exporter

A simple HTTP service that exposes filesystem statistics over HTTP and as Prometheus metrics.

## Endpoints

- `GET /health` - Health check endpoint
- `GET /metrics` - Prometheus metrics endpoint
- `GET /disk` - Get disk usage for a specific path
  - Query parameter: `path` (required)
  - Response: `{ "path": string, "total_bytes": number, "free_bytes": number, "used_percent": number }`

## Metrics

- `disk_used_percent` - Used percent of the root filesystem (gauge)
- `disk_free_bytes` - Free bytes of the root filesystem (gauge)

## Background Refresh

The service automatically refreshes the root filesystem (`/`) metrics every 10 seconds and once at startup.

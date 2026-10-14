# Disk Usage Exporter

A simple HTTP service that exposes filesystem statistics as JSON and Prometheus metrics.

## Endpoints

- `GET /health` - Health check endpoint
- `GET /metrics` - Prometheus metrics endpoint
- `GET /disk` - Get disk usage for a specific path
  - Query parameter: `path` (required)
  - Response: `{path, total_bytes, free_bytes, used_percent}`

## Metrics

- `disk_used_percent` - Used percent of the root filesystem (gauge)
- `disk_free_bytes` - Free bytes of the root filesystem (gauge)

## Usage

```bash
# Build and run
go build -o disk-usage-exporter .
./disk-usage-exporter

# Query disk usage
curl "http://localhost:8080/disk?path=/"

# Check health
curl http://localhost:8080/health

# Get Prometheus metrics
curl http://localhost:8080/metrics
```

The service automatically refreshes the root filesystem metrics (`/`) every 10 seconds in the background.

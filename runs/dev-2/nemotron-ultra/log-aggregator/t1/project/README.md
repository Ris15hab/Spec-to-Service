# Log Aggregator

A simple HTTP service that ingests log lines and counts them by level.

## Endpoints

- `GET /health` - Health check endpoint
- `GET /metrics` - Prometheus metrics endpoint
- `POST /logs` - Ingest a log line (body: `{"level": "info|warn|error", "message": "non-empty string"}`)
- `GET /logs/stats` - Get log counts by level

## Metrics

- `log_lines_info_total` - Counter of info lines ingested
- `log_lines_warn_total` - Counter of warn lines ingested
- `log_lines_error_total` - Counter of error lines ingested

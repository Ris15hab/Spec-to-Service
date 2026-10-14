# Log Aggregator

A simple HTTP service that ingests log lines and counts them by level.

## Endpoints

- `GET /health` - Health check endpoint
- `GET /metrics` - Prometheus metrics endpoint
- `POST /logs` - Ingest a log line
- `GET /logs/stats` - Get log counts by level

## Metrics

- `log_lines_info_total` - Counter of info lines ingested
- `log_lines_warn_total` - Counter of warn lines ingested
- `log_lines_error_total` - Counter of error lines ingested

## Usage

### Ingest a log line

```bash
curl -X POST http://localhost:8080/logs \
  -H "Content-Type: application/json" \
  -d '{"level": "info", "message": "application started"}'
```

Response: `202 Accepted`

### Get statistics

```bash
curl http://localhost:8080/logs/stats
```

Response:
```json
{
  "counts": {
    "info": 10,
    "warn": 3,
    "error": 1
  }
}
```

### Health check

```bash
curl http://localhost:8080/health
```

### Prometheus metrics

```bash
curl http://localhost:8080/metrics
```

# Task Scheduler Status API

A simple HTTP service for storing cron-like schedules and computing their next run time.

## Endpoints

- `GET /health` - Health check endpoint
- `GET /metrics` - Prometheus metrics endpoint
- `POST /schedules` - Create a new schedule
- `GET /schedules` - List all schedules
- `GET /schedules/{name}/next` - Get next run time for a schedule

## Metrics

- `schedules_total` - Gauge of stored schedules
- `schedule_queries_total` - Counter of next-run computations served

## Schedule Format

Schedules use a simplified cron format with exactly 5 space-separated fields:
- Minute: `*`, `*/N` (N in 1..59), or integer 0..59
- Hour: `*` (required)
- Day of month: `*` (required)
- Month: `*` (required)
- Day of week: `*` (required)

Examples:
- `*/15 * * * *` - Every 15 minutes
- `5 * * * *` - At minute 5 of every hour
- `* * * * *` - Every minute

## API Examples

### Create a schedule
```bash
curl -X POST /schedules \
  -H "Content-Type: application/json" \
  -d '{"name": "my-schedule", "cron": "*/15 * * * *"}'
```

### List schedules
```bash
curl /schedules
```

### Get next run time
```bash
curl "/schedules/my-schedule/next?from=2026-01-01T00:07:00Z"
# Returns: {"next": "2026-01-01T00:15:00Z"}
```

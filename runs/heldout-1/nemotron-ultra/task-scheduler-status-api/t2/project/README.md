# Task Scheduler Status API

A simple HTTP service for storing cron-like schedules and computing their next run times.

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

Schedules use a restricted cron format with exactly 5 space-separated fields:
- Minute: `*`, `*/N` (N in 1..59), or integer 0..59
- Hour: `*` (must be wildcard)
- Day of month: `*` (must be wildcard)
- Month: `*` (must be wildcard)
- Day of week: `*` (must be wildcard)

Examples:
- `*/15 * * * *` - Every 15 minutes
- `5 * * * *` - At minute 5 of every hour
- `* * * * *` - Every minute

## Next Run Computation

The `/schedules/{name}/next` endpoint requires a `from` query parameter in RFC3339 format.
It returns the first matching minute strictly after the given time (seconds zeroed).

Example:
- Cron: `*/15 * * * *`, From: `2026-01-01T00:07:00Z` → Next: `2026-01-01T00:15:00Z`
- Cron: `5 * * * *`, From: `2026-01-01T00:05:00Z` → Next: `2026-01-01T01:05:00Z`

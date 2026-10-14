# Node Heartbeat Monitor

A simple HTTP service for monitoring node heartbeats. Nodes periodically POST heartbeats; a node is considered stale if it has not been heard from within the configured threshold.

## Endpoints

- `GET /health` - Health check endpoint
- `GET /metrics` - Prometheus metrics endpoint
- `POST /heartbeat` - Submit a heartbeat for a node
  - Body: `{"node": "node-name"}` (node must be non-empty string)
  - Returns: `204 No Content` on success, `400 Bad Request` if node missing or malformed JSON
- `GET /nodes` - List all known nodes with their status
  - Returns: `200 OK` with JSON array of `{node, status: healthy|stale, last_seen: RFC3339}`
- `GET /nodes/{node}` - Get status of a specific node
  - Returns: `200 OK` with node object, `404 Not Found` if unknown

## Metrics

- `nodes_healthy` - Gauge: Nodes with a recent heartbeat
- `nodes_stale` - Gauge: Nodes with no recent heartbeat

## Configuration

- `STALE_AFTER_SECONDS` - Time in seconds after which a node is considered stale (default: 30)

## Running

```bash
go run .
```

The server listens on `:8080` by default.

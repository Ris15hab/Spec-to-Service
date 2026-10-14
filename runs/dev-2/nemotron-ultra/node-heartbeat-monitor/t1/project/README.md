# Node Heartbeat Monitor

A simple HTTP service for monitoring node heartbeats. Nodes periodically send heartbeats; a node is considered stale if it hasn't sent a heartbeat within the configured threshold.

## Endpoints

- `GET /health` - Health check endpoint
- `GET /metrics` - Prometheus metrics endpoint
- `POST /heartbeat` - Receive a heartbeat from a node
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

## Behavior

- A background goroutine recomputes the `nodes_healthy` and `nodes_stale` gauges every second
- State is kept in memory and reset on service restart

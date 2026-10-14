# Node Heartbeat Monitor

A simple HTTP service for monitoring node heartbeats. Nodes periodically send heartbeats, and the service tracks their status (healthy or stale) based on the last received heartbeat.

## Endpoints

- `GET /health` - Health check endpoint
- `GET /metrics` - Prometheus metrics endpoint
- `POST /heartbeat` - Receive a heartbeat from a node
  - Body: `{"node": "node-name"}` (node must be a non-empty string)
  - Returns: `204 No Content` on success, `400 Bad Request` if node is missing or JSON is malformed
- `GET /nodes` - List all known nodes with their status
  - Returns: `200 OK` with JSON array of node objects
- `GET /nodes/{node}` - Get status of a specific node
  - Returns: `200 OK` with node object, `404 Not Found` if node is unknown

## Metrics

- `nodes_healthy` - Gauge: Number of nodes with a recent heartbeat (healthy)
- `nodes_stale` - Gauge: Number of nodes with no recent heartbeat (stale)

## Configuration

- `STALE_AFTER_SECONDS` - Time in seconds after which a node is considered stale (default: 30)

## Node Status

A node is considered **healthy** if its last heartbeat was received within `STALE_AFTER_SECONDS` seconds.
A node is considered **stale** if its last heartbeat is older than `STALE_AFTER_SECONDS` seconds.

Node objects returned by the API have the following structure:
```json
{
  "node": "node-name",
  "status": "healthy|stale",
  "last_seen": "2024-01-15T10:30:00Z"
}
```
The `last_seen` field is in RFC3339 format.

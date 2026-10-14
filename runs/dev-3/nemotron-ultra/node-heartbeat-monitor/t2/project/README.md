# Node Heartbeat Monitor

A simple HTTP service for monitoring node heartbeats. Nodes periodically send heartbeats, and the service tracks their status as healthy or stale based on the last received heartbeat.

## Endpoints

- `GET /health` - Health check endpoint
- `GET /metrics` - Prometheus metrics endpoint
- `POST /heartbeat` - Submit a heartbeat for a node
- `GET /nodes` - List all nodes with their status
- `GET /nodes/{node}` - Get status of a specific node

## Metrics

- `nodes_healthy` - Gauge of nodes with a recent heartbeat
- `nodes_stale` - Gauge of nodes with no recent heartbeat

## Configuration

- `STALE_AFTER_SECONDS` - Time in seconds after which a node is considered stale (default: 30)

## Usage

### Submit a heartbeat

```bash
curl -X POST http://localhost:8080/heartbeat \
  -H "Content-Type: application/json" \
  -d '{"node": "node-1"}'
```

Response: `204 No Content`

### List all nodes

```bash
curl http://localhost:8080/nodes
```

Response:
```json
[
  {
    "node": "node-1",
    "status": "healthy",
    "last_seen": "2024-01-15T10:30:00Z"
  }
]
```

### Get a specific node

```bash
curl http://localhost:8080/nodes/node-1
```

Response:
```json
{
  "node": "node-1",
  "status": "healthy",
  "last_seen": "2024-01-15T10:30:00Z"
}
```

If the node is unknown, returns `404 Not Found`.

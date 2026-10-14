# Node Heartbeat Monitor

A simple HTTP service for monitoring node heartbeats. Nodes send periodic heartbeats; a node is considered stale if it hasn't sent a heartbeat within the configured threshold.

## Endpoints

- `GET /health` - Health check endpoint
- `GET /metrics` - Prometheus metrics endpoint
- `POST /heartbeat` - Submit a heartbeat for a node
- `GET /nodes` - List all nodes with their status
- `GET /nodes/{node}` - Get status of a specific node

## Metrics

- `nodes_healthy` - Gauge: Nodes with a recent heartbeat
- `nodes_stale` - Gauge: Nodes with no recent heartbeat

## Configuration

- `STALE_AFTER_SECONDS` - Time in seconds after which a node is considered stale (default: 30)

## Usage

### Submit a heartbeat
```bash
curl -X POST /heartbeat -H "Content-Type: application/json" -d '{"node": "node-1"}'
```

### List all nodes
```bash
curl /nodes
```

### Get a specific node
```bash
curl /nodes/node-1
```

# Cluster Inventory API

A simple HTTP service for registering and querying cluster nodes with labels.

## Endpoints

- `GET /health` - Health check endpoint
- `GET /metrics` - Prometheus metrics endpoint
- `POST /nodes` - Register a new node
- `GET /nodes` - List all nodes (with optional label filtering)
- `GET /nodes/{name}` - Get a specific node by name

## Metrics

- `inventory_nodes` - Gauge: Number of registered nodes
- `inventory_queries_total` - Counter: Total number of list queries served

## Node Registration

Register a node with a name and optional labels:

```bash
curl -X POST /nodes \
  -H "Content-Type: application/json" \
  -d '{"name": "node-1", "labels": {"zone": "us-east-1", "gpu": "h100"}}'
```

- Returns `201 Created` on success
- Returns `409 Conflict` if a node with the same name already exists
- Returns `400 Bad Request` if name is missing or JSON is malformed

## Listing Nodes

List all registered nodes:

```bash
curl /nodes
```

Filter by labels (repeatable, all must match):

```bash
curl '/nodes?label=zone=us-east-1&label=gpu=h100'
```

- Returns `200 OK` with JSON array of nodes
- Returns `400 Bad Request` if a label parameter is not in `key=value` format

## Getting a Single Node

```bash
curl /nodes/node-1
```

- Returns `200 OK` with the node JSON
- Returns `404 Not Found` if the node doesn't exist

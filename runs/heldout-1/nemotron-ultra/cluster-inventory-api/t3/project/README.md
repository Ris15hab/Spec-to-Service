# Cluster Inventory API

A simple HTTP service for registering cluster nodes with labels and querying them.

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

Response: `201 Created` with the node JSON, or `409 Conflict` if name exists, or `400 Bad Request` if name is missing or JSON is malformed.

## Listing Nodes

List all nodes:

```bash
curl /nodes
```

Filter by labels (repeatable, all must match):

```bash
curl '/nodes?label=zone=us-east-1&label=gpu=h100'
```

Response: `200 OK` with JSON array of nodes, or `400 Bad Request` if a label parameter is malformed.

## Getting a Single Node

```bash
curl /nodes/node-1
```

Response: `200 OK` with node JSON, or `404 Not Found` if unknown.

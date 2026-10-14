# Cluster Inventory API

A simple HTTP service for registering cluster nodes with labels and querying them.

## Endpoints

- `GET /health`
- `GET /metrics`
- `POST /nodes`
- `GET /nodes`
- `GET /nodes/{name}`

## Metrics

- `inventory_nodes`
- `inventory_queries_total`

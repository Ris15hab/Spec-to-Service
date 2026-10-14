# Artifact Registry Metadata Service

Stores metadata (not the blobs) for versioned build artifacts.

## Endpoints

- `GET /health`
- `GET /metrics`
- `POST /artifacts`
- `GET /artifacts`
- `GET /artifacts/{name}`
- `GET /artifacts/{name}/{version}`

## Metrics

- `artifacts_total`
- `artifact_uploads_total`

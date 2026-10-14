# Feature Flag Service

A simple HTTP service for managing feature flags with percentage rollout and per-user evaluation.

## Endpoints

- `GET /health`
- `GET /metrics`
- `PUT /flags/{name}`
- `GET /flags/{name}`
- `GET /flags`
- `GET /evaluate`

## Metrics

- `flags_total`
- `flag_evaluations_total`

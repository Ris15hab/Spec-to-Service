# Feature Flag Service

A simple HTTP service for managing feature flags with percentage-based rollout and deterministic per-user evaluation.

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

## Evaluation Logic

Flags are evaluated deterministically per user using FNV-32a hash:
- If flag is disabled: `enabled: false`
- If flag is enabled: `enabled: true` iff `fnv32a(user) % 100 < rollout_percent`

This ensures the same user always gets the same result for the same flag configuration.

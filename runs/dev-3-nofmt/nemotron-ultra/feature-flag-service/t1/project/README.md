# Feature Flag Service

A simple HTTP service for managing feature flags with percentage-based rollout and deterministic per-user evaluation.

## Endpoints

- `GET /health` - Health check endpoint
- `GET /metrics` - Prometheus metrics endpoint
- `PUT /flags/{name}` - Create or update a feature flag
- `GET /flags/{name}` - Get a specific feature flag
- `GET /flags` - List all feature flags
- `GET /evaluate` - Evaluate a feature flag for a user

## Metrics

- `flags_total` - Gauge: Number of flags defined
- `flag_evaluations_total` - Counter: Successful evaluations

## Flag Format

Flags are JSON objects with:
- `name` (string): Flag identifier
- `enabled` (boolean): Whether the flag is active
- `rollout_percent` (integer 0-100): Percentage of users who see the flag enabled

## Evaluation Logic

When evaluating a flag for a user:
1. If the flag is disabled, returns `{"enabled": false}`
2. If enabled, computes `fnv32a(user) % 100 < rollout_percent`
3. Returns `{"enabled": true}` if the condition holds, otherwise `false`

The FNV-32a hash ensures deterministic evaluation: the same user always gets the same result for the same flag configuration.

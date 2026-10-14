# Feature Flag Service

A simple HTTP service for managing feature flags with percentage-based rollout and deterministic per-user evaluation.

## Endpoints

- `GET /health` - Health check endpoint
- `GET /metrics` - Prometheus metrics endpoint
- `PUT /flags/{name}` - Create or update a feature flag
  - Body: `{"enabled": bool, "rollout_percent": int (0-100)}`
  - Returns: 201 (created) or 200 (updated), 400 for invalid input
- `GET /flags/{name}` - Get a specific feature flag
  - Returns: 200 with flag JSON, 404 if not found
- `GET /flags` - List all feature flags
  - Returns: 200 with JSON array of flags
- `GET /evaluate` - Evaluate a flag for a specific user
  - Query params: `flag` (required), `user` (required)
  - Returns: 200 `{"enabled": bool}`, 400 if params missing, 404 if flag unknown

## Metrics

- `flags_total` (gauge) - Number of flags defined
- `flag_evaluations_total` (counter) - Successful evaluations

## Evaluation Logic

Evaluation is deterministic: the same user always gets the same result for the same flag.
- If the flag is disabled, returns `{"enabled": false}`
- If enabled, computes `fnv32a(user) % 100 < rollout_percent` to determine enabled state

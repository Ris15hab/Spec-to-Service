# Feature Flag Service

A simple HTTP service for managing feature flags with percentage-based rollout and deterministic per-user evaluation.

## Endpoints

- `GET /health` - Health check endpoint
- `GET /metrics` - Prometheus metrics endpoint
- `PUT /flags/{name}` - Create or update a feature flag
- `GET /flags/{name}` - Get a specific feature flag
- `GET /flags` - List all feature flags
- `GET /evaluate` - Evaluate a flag for a specific user

## Metrics

- `flags_total` - Number of flags defined (gauge)
- `flag_evaluations_total` - Total successful flag evaluations (counter)

## Flag Format

```json
{
  "name": "beta",
  "enabled": true,
  "rollout_percent": 50
}
```

- `enabled`: Boolean, whether the flag is active
- `rollout_percent`: Integer 0-100, percentage of users who get the feature

## Evaluation Logic

When evaluating a flag for a user:
1. If the flag is disabled, returns `{"enabled": false}`
2. If enabled, computes FNV-32a hash of the user string
3. Returns `{"enabled": true}` if `hash % 100 < rollout_percent`, otherwise `false`

The same user always gets the same result for the same flag (deterministic).

## Example Usage

```bash
# Create a flag
curl -X PUT /flags/beta \
  -H "Content-Type: application/json" \
  -d '{"enabled": true, "rollout_percent": 25}'

# Evaluate for a user
curl "/evaluate?flag=beta&user=user123"

# List all flags
curl /flags
```

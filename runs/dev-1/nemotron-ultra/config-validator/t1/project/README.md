# config-validator

A simple HTTP service that validates service configurations submitted as JSON.

## Endpoints

- `GET /health` - Health check endpoint
- `GET /metrics` - Prometheus metrics endpoint
- `POST /validate` - Validate a service configuration

### POST /validate

Validates a service configuration with the following rules:
- `name` (string): must be non-empty
- `replicas` (int): must be between 1 and 100
- `port` (int): must be between 1 and 65535

**Request body:**
```json
{
  "name": "api",
  "replicas": 3,
  "port": 8080
}
```

**Responses:**
- `200 OK` - Valid configuration: `{"valid": true, "errors": []}`
- `422 Unprocessable Entity` - Invalid configuration: `{"valid": false, "errors": ["error1", "error2", ...]}`
- `400 Bad Request` - Malformed JSON: `{"valid": false, "errors": ["malformed JSON"]}`

All violated rules are reported, not just the first one.

## Metrics

- `config_validations_total` - Counter of total validation requests
- `config_validation_failures_total` - Counter of validation requests that failed (422 responses only)

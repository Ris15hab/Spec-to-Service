# Alert Rule Evaluator

A simple HTTP service for storing threshold alert rules and evaluating metric values against them.

## Endpoints

- `GET /health` - Health check endpoint
- `GET /metrics` - Prometheus metrics endpoint
- `POST /rules` - Create a new alert rule
- `GET /rules` - List all alert rules
- `POST /evaluate` - Evaluate a metric value against stored rules

## Metrics

- `alert_rules` - Gauge: Number of defined rules
- `alerts_firing` - Gauge: Number of rules firing in the most recent evaluation
- `alert_evaluations_total` - Counter: Total number of evaluations performed

## Rule Format

Rules are created with a JSON body:
```json
{
  "name": "rule-name",
  "metric": "metric-name",
  "op": ">" or "<",
  "threshold": 80.0
}
```

## Evaluation

Evaluate a metric value:
```json
{
  "metric": "temp",
  "value": 95.0
}
```

Response:
```json
{
  "firing": ["rule-name-1", "rule-name-2"]
}
```

Only rules matching the evaluated metric are considered. Firing rules are sorted by name.

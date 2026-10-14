# Alert Rule Evaluator

A simple HTTP service for storing threshold alert rules and evaluating metric values against them.

## Endpoints

- `GET /health`
- `GET /metrics`
- `POST /rules`
- `GET /rules`
- `POST /evaluate`

## Metrics

- `alert_rules`
- `alerts_firing`
- `alert_evaluations_total`

## Rule Format

Rules are JSON objects with the following fields:
- `name` (string, required, unique): Rule identifier
- `metric` (string, required): Metric name to monitor
- `op` (string, required): Comparison operator, either `>` or `<`
- `threshold` (number, required): Threshold value

Example:
```json
{"name": "high_cpu", "metric": "cpu_usage", "op": ">", "threshold": 90}
```

## Evaluation

POST `/evaluate` with a metric name and value:
```json
{"metric": "cpu_usage", "value": 95}
```

Returns a list of firing rule names sorted alphabetically:
```json
{"firing": ["high_cpu"]}
```

Rules for other metrics are ignored during evaluation.

# Alert Rule Evaluator

A simple HTTP service for storing threshold alert rules and evaluating metric values against them.

## Endpoints

- `GET /health` - Health check endpoint
- `GET /metrics` - Prometheus metrics endpoint
- `POST /rules` - Create a new alert rule
- `GET /rules` - List all alert rules
- `POST /evaluate` - Evaluate a metric value against rules

## Metrics

- `alert_rules` - Gauge: Number of defined rules
- `alerts_firing` - Gauge: Number of rules firing in the most recent evaluation
- `alert_evaluations_total` - Counter: Total number of evaluations performed

## Rule Format

Rules are JSON objects with the following fields:
- `name` (string, required, unique): Rule identifier
- `metric` (string, required): Metric name to monitor
- `op` (string, required): Comparison operator, either `>` or `<`
- `threshold` (number, required): Threshold value

Example:
```json
{
  "name": "high-temperature",
  "metric": "temperature",
  "op": ">",
  "threshold": 80
}
```

## Evaluation

POST to `/evaluate` with a JSON body containing `metric` (string) and `value` (number).
Returns a JSON object with a `firing` array containing the names of all rules for that metric
whose condition is satisfied, sorted alphabetically.

Example request:
```json
{"metric": "temperature", "value": 95}
```

Example response:
```json
{"firing": ["high-temperature"]}
```

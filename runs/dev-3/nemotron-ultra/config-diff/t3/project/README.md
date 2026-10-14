# config-diff

A simple HTTP service that compares two flat configuration maps and returns a structured diff.

## Endpoints

- `GET /health`
- `GET /metrics`
- `POST /diff`

## Metrics

- `config_diffs_total`
- `config_diff_changes_last`

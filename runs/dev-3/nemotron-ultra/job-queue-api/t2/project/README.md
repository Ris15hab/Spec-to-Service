# Job Queue API

A simple FIFO job queue service with enqueue, dequeue, and acknowledge operations.

## Endpoints

- `GET /health`
- `GET /metrics`
- `POST /jobs`
- `POST /jobs/dequeue`
- `POST /jobs/{id}/ack`

## Metrics

- `queue_depth`
- `jobs_enqueued_total`
- `jobs_acked_total`

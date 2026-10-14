# Job Queue API

A simple FIFO job queue service with enqueue, dequeue, and acknowledge operations.

## Endpoints

- `GET /health` - Health check endpoint
- `GET /metrics` - Prometheus metrics endpoint
- `POST /jobs` - Enqueue a job with body `{"id": "string", "payload": any}`
  - Returns `201 Created` on success
  - Returns `409 Conflict` if job ID already exists
  - Returns `400 Bad Request` if ID is missing or invalid JSON
- `POST /jobs/dequeue` - Dequeue the oldest job
  - Returns `200 OK` with job `{"id": "string", "payload": any}` and marks it in-flight
  - Returns `204 No Content` if queue is empty
- `POST /jobs/{id}/ack` - Acknowledge a job as completed
  - Returns `200 OK` if job was in-flight
  - Returns `404 Not Found` if job ID is unknown
  - Returns `409 Conflict` if job exists but is not in-flight

## Metrics

- `queue_depth` (gauge) - Jobs waiting in queue (not in-flight)
- `jobs_enqueued_total` (counter) - Total jobs accepted
- `jobs_acked_total` (counter) - Total jobs acknowledged

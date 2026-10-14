# Job Queue API

A simple FIFO job queue service with enqueue, dequeue, and acknowledge operations.

## Endpoints

- `GET /health` - Health check endpoint
- `GET /metrics` - Prometheus metrics endpoint
- `POST /jobs` - Enqueue a new job
- `POST /jobs/dequeue` - Dequeue the oldest queued job
- `POST /jobs/{id}/ack` - Acknowledge a job as completed

## Metrics

- `queue_depth` - Gauge of jobs waiting (not in-flight)
- `jobs_enqueued_total` - Counter of jobs accepted
- `jobs_acked_total` - Counter of jobs acknowledged

## Job Lifecycle

1. **Enqueue** (`POST /jobs`): Add a job with a unique ID and arbitrary payload. Returns 201 on success, 409 if ID exists, 400 if ID missing.
2. **Dequeue** (`POST /jobs/dequeue`): Retrieve and mark the oldest queued job as in-flight. Returns 200 with job details, 204 if queue empty.
3. **Acknowledge** (`POST /jobs/{id}/ack`): Mark an in-flight job as completed. Returns 200 on success, 404 if unknown, 409 if not in-flight.

## Example Usage

```bash
# Enqueue a job
curl -X POST http://localhost:8080/jobs \
  -H "Content-Type: application/json" \
  -d '{"id": "job-1", "payload": {"task": "process"}}'

# Dequeue a job
curl -X POST http://localhost:8080/jobs/dequeue

# Acknowledge a job
curl -X POST http://localhost:8080/jobs/job-1/ack
```

# Job Queue API

A simple FIFO job queue service with enqueue, dequeue, and acknowledge operations.

## Endpoints

- `GET /health` - Health check endpoint
- `GET /metrics` - Prometheus metrics endpoint
- `POST /jobs` - Enqueue a job with body `{id: string, payload: any}`
- `POST /jobs/dequeue` - Dequeue the oldest job (FIFO), marks it in-flight
- `POST /jobs/{id}/ack` - Acknowledge a job that is in-flight

## Metrics

- `queue_depth` - Gauge: Jobs waiting (not in-flight)
- `jobs_enqueued_total` - Counter: Jobs accepted
- `jobs_acked_total` - Counter: Jobs acknowledged

## Usage

### Enqueue a job
```bash
curl -X POST http://localhost:8080/jobs \
  -H "Content-Type: application/json" \
  -d '{"id": "job-1", "payload": {"data": "example"}}'
```
Returns `201 Created` on success, `409 Conflict` if ID exists, `400 Bad Request` if ID missing.

### Dequeue a job
```bash
curl -X POST http://localhost:8080/jobs/dequeue
```
Returns `200 OK` with job `{id, payload}` on success, `204 No Content` if queue empty.

### Acknowledge a job
```bash
curl -X POST http://localhost:8080/jobs/job-1/ack
```
Returns `200 OK` if job was in-flight, `404 Not Found` if unknown, `409 Conflict` if not in-flight.

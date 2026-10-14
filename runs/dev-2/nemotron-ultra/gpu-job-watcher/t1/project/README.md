# GPU Job Watcher

A simple HTTP service for tracking GPU jobs in memory, exposing queue length and GPU utilization metrics.

## Endpoints

- `GET /health` - Health check endpoint
- `GET /metrics` - Prometheus metrics endpoint
- `POST /jobs` - Enqueue a job with `{id, gpus}`; returns 201 on success
- `GET /jobs` - List all jobs with their status

## Metrics

- `gpu_job_queue_length` - Gauge: Number of jobs waiting in queue
- `gpu_utilization_percent` - Gauge: Mean GPU utilization percentage

## Usage

Start the server:
```bash
go run .
```

The server listens on `:8080` by default.

### Enqueue a job
```bash
curl -X POST http://localhost:8080/jobs \
  -H "Content-Type: application/json" \
  -d '{"id": "job-1", "gpus": 4}'
```

### List jobs
```bash
curl http://localhost:8080/jobs
```

### Check metrics
```bash
curl http://localhost:8080/metrics
```

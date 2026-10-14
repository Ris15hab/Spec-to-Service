# GPU Job Watcher

A simple HTTP service for tracking GPU jobs in memory and exposing queue length and GPU utilization metrics.

## Endpoints

- `GET /health` - Health check endpoint
- `GET /metrics` - Prometheus metrics endpoint
- `POST /jobs` - Enqueue a job with `{id, gpus}`; returns 201 on success
- `GET /jobs` - List all jobs with their status

## Metrics

- `gpu_job_queue_length` - Gauge representing the number of jobs waiting in the queue
- `gpu_utilization_percent` - Gauge representing the mean GPU utilization percentage

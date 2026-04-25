# Batch and Async Processing in LocalAI

This guide covers how to implement and use batch inference and asynchronous job processing in LocalAI.

## Overview

LocalAI supports two async processing patterns:
1. **Batch inference** — submit multiple prompts in a single request
2. **Async jobs** — submit a request and poll for results later

These patterns are useful for high-throughput workloads where you don't need an immediate response.

---

## Async Job API

### Submitting an Async Request

Any standard endpoint can be made async by adding `"async": true` to the request body or using the `X-LocalAI-Async: true` header.

```bash
curl http://localhost:8080/v1/completions \
  -H 'Content-Type: application/json' \
  -H 'X-LocalAI-Async: true' \
  -d '{
    "model": "gpt-4",
    "prompt": "Tell me a joke",
    "max_tokens": 100
  }'
```

Response:
```json
{
  "job_id": "job_abc123",
  "status": "queued",
  "created_at": 1712000000
}
```

### Polling for Job Status

```bash
curl http://localhost:8080/v1/jobs/job_abc123
```

Response when complete:
```json
{
  "job_id": "job_abc123",
  "status": "completed",
  "created_at": 1712000000,
  "completed_at": 1712000005,
  "result": {
    "id": "cmpl-xyz",
    "object": "text_completion",
    "choices": [{"text": "Why did the chicken...", "finish_reason": "stop"}]
  }
}
```

Possible `status` values: `queued`, `running`, `completed`, `failed`, `cancelled`

### Cancelling a Job

```bash
curl -X DELETE http://localhost:8080/v1/jobs/job_abc123
```

---

## Batch Inference API

### Submitting a Batch

The `/v1/batch/completions` endpoint accepts an array of requests:

```bash
curl http://localhost:8080/v1/batch/completions \
  -H 'Content-Type: application/json' \
  -d '{
    "model": "gpt-4",
    "requests": [
      {"prompt": "What is 2+2?", "max_tokens": 20},
      {"prompt": "Name a planet.", "max_tokens": 20},
      {"prompt": "Say hello.", "max_tokens": 20}
    ]
  }'
```

Returns a batch job ID:
```json
{
  "batch_id": "batch_def456",
  "status": "queued",
  "total": 3
}
```

### Retrieving Batch Results

```bash
curl http://localhost:8080/v1/batch/completions/batch_def456
```

```json
{
  "batch_id": "batch_def456",
  "status": "completed",
  "total": 3,
  "completed": 3,
  "failed": 0,
  "results": [
    {"index": 0, "result": {"choices": [{"text": "4"}]}},
    {"index": 1, "result": {"choices": [{"text": "Mars"}]}},
    {"index": 2, "result": {"choices": [{"text": "Hello!"}]}}
  ]
}
```

---

## Backend Implementation Notes

### Job Queue

Jobs are managed via an internal queue backed by a configurable store:
- **In-memory** (default, non-persistent): suitable for single-node deployments
- **Redis** (optional): enables distributed queuing across multiple LocalAI nodes

Configure via environment variables:
```bash
LOCALAI_ASYNC_BACKEND=redis
LOCALAI_REDIS_URL=redis://localhost:6379
LOCALAI_ASYNC_WORKERS=4        # number of concurrent job workers
LOCALAI_ASYNC_JOB_TTL=3600    # seconds to retain completed jobs
```

### Worker Goroutines

Async jobs are processed by a pool of worker goroutines. The pool size is controlled by `LOCALAI_ASYNC_WORKERS` (default: number of CPU cores).

Each worker:
1. Dequeues the next job
2. Calls the appropriate backend handler
3. Stores the result
4. Updates job status

### Adding Async Support to a New Endpoint

Wrap your handler with the `AsyncMiddleware`:

```go
func RegisterMyRoutes(router *fiber.App, deps *Dependencies) {
    router.Post("/v1/my-endpoint", AsyncMiddleware(deps), MyHandler(deps))
}
```

The middleware checks for the async flag and, if present, enqueues the request and returns a job ID immediately instead of waiting for the handler to complete.

---

## Concurrency and Rate Limiting

- Batch requests are split into individual jobs internally and processed concurrently up to the worker pool limit.
- Per-model concurrency limits are respected: if a model allows only 1 concurrent request, batch items for that model are serialized.
- Use `LOCALAI_MAX_QUEUE_SIZE` (default: 1000) to limit the number of pending jobs. Requests exceeding the limit receive HTTP 429.

---

## Error Handling

If an individual item in a batch fails, the batch as a whole is still marked `completed` but the failed item will have:
```json
{"index": 1, "error": {"code": "model_error", "message": "context length exceeded"}}
```

The batch `failed` counter will be incremented accordingly.

---

## Monitoring

Async job metrics are exposed via the `/metrics` Prometheus endpoint:

| Metric | Description |
|--------|-------------|
| `localai_async_jobs_queued` | Current number of queued jobs |
| `localai_async_jobs_running` | Current number of running jobs |
| `localai_async_jobs_completed_total` | Total completed jobs |
| `localai_async_jobs_failed_total` | Total failed jobs |
| `localai_async_job_duration_seconds` | Histogram of job processing time |

---

## Related

- [Streaming and WebSockets](.agents/streaming-and-websockets.md) — for real-time responses
- [API Endpoints and Auth](.agents/api-endpoints-and-auth.md) — for adding new endpoints
- [Context and Memory](.agents/context-and-memory.md) — for stateful conversations

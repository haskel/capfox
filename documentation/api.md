# API Reference

**capfox** REST API documentation.

Base URL: `http://localhost:8080`

## Authentication

When `auth.enabled: true` in config, all endpoints require Basic Authentication:

```
Authorization: Basic base64(user:password)
```

Example:

```bash
curl -u admin:secret http://localhost:8080/status
```

---

## Health & Readiness

### GET /health

Health check endpoint. Always returns 200 if the server is running.

```
GET /health

→ 200 OK
{"status": "ok"}
```

---

### GET /ready

Readiness check. Returns 200 when the server has collected initial metrics and is ready to serve requests.

```
GET /ready

→ 200 OK
{"ready": true}

→ 503 Service Unavailable
{"ready": false, "message": "aggregator has not collected initial metrics"}
```

Use this endpoint for Kubernetes readiness probes.

---

## Info

### GET /

Server info endpoint.

```
GET /

→ 200 OK
{"name": "capfox", "version": "0.2.0"}
```

---

## Status

### GET /status

Get current resource metrics.

```
GET /status

→ 200 OK
{
  "cpu": {
    "usage_percent": 45.2
  },
  "memory": {
    "usage_percent": 62.1,
    "total_bytes": 34359738368,
    "used_bytes": 21367234560
  },
  "storage": {
    "/": {
      "total_bytes": 536870912000,
      "free_bytes": 161395916800
    }
  },
  "gpus": [
    {
      "usage_percent": 30.0,
      "vram_used_bytes": 4294967296,
      "vram_total_bytes": 25769803776
    }
  ],
  "process": {
    "total_processes": 342,
    "total_threads": 1256
  }
}
```

---

## Capacity Check

### POST /ask

Check if a task can run based on current resource availability.

**Request:**

```json
{
  "task": "video_encode",
  "complexity": 30,
  "resources": {
    "cpu": 50,
    "memory": 30,
    "gpu": 0
  }
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `task` | string | Yes | Task name |
| `complexity` | int | No | Task complexity (for prediction) |
| `resources.cpu` | int | No | Estimated CPU usage % |
| `resources.memory` | int | No | Estimated memory usage % |
| `resources.gpu` | int | No | Estimated GPU usage % |

**Query Parameters:**

| Param | Description |
|-------|-------------|
| `reason=true` | Include denial reasons in response |

**Headers:**

| Header | Description |
|--------|-------------|
| `X-Reason: true` | Alternative to query param |

**Response (allowed):**

```
→ 200 OK
{"allowed": true}
```

**Response (denied):**

```
→ 503 Service Unavailable
{
  "allowed": false,
  "reasons": ["cpu_high", "memory_high"]
}
```

**Reason codes:**

| Code | Description |
|------|-------------|
| `cpu_high` | CPU usage exceeds threshold |
| `memory_high` | Memory usage exceeds threshold |
| `gpu_high` | GPU usage exceeds threshold |
| `vram_high` | VRAM usage exceeds threshold |
| `disk_low` | Disk free space below threshold |

---

## Task Notification

### POST /task/notify

Notify the server that a task has started. Used by the learning engine to observe resource impact.

**Request:**

```json
{
  "task": "video_encode",
  "complexity": 30
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `task` | string | Yes | Task name |
| `complexity` | int | No | Task complexity |

**Response:**

```
→ 200 OK
{"received": true, "task": "video_encode"}
```

---

## Statistics

### GET /stats

Get learned task statistics from the learning engine.

**Query Parameters:**

| Param | Description |
|-------|-------------|
| `task=<name>` | Get stats for specific task |

**Response (all tasks):**

```
GET /stats

→ 200 OK
{
  "tasks": {
    "video_encode": {
      "task": "video_encode",
      "count": 42,
      "avg_cpu_delta": 25.3,
      "avg_mem_delta": 12.5,
      "avg_gpu_delta": 0,
      "avg_vram_delta": 0
    },
    "ml_training": {
      "task": "ml_training",
      "count": 114,
      "avg_cpu_delta": 45.8,
      "avg_mem_delta": 35.2,
      "avg_gpu_delta": 80.1,
      "avg_vram_delta": 65.4
    }
  },
  "total_tasks": 156
}
```

**Response (single task):**

```
GET /stats?task=video_encode

→ 200 OK
{
  "task": "video_encode",
  "count": 42,
  "avg_cpu_delta": 25.3,
  "avg_mem_delta": 12.5
}

→ 404 Not Found
task not found
```

---

## V2 API (Experimental)

Advanced decision engine with prediction and confidence scoring.

### POST /v2/ask

Enhanced capacity check with prediction details.

**Request:**

```json
{
  "task": "video_encode",
  "complexity": 30,
  "resources": {
    "cpu": 50,
    "memory": 30
  }
}
```

**Response:**

```
→ 200 OK
{
  "allowed": true,
  "reasons": [],
  "predicted": {
    "cpu": 75.2,
    "memory": 82.5,
    "gpu": 30.0,
    "vram": 25.0
  },
  "confidence": 0.85,
  "strategy": "predictive",
  "model": "linear"
}

→ 503 Service Unavailable
{
  "allowed": false,
  "reasons": ["cpu_high"],
  "predicted": {
    "cpu": 95.5,
    "memory": 82.5
  },
  "confidence": 0.72,
  "strategy": "predictive",
  "model": "linear"
}
```

---

### GET /v2/model/stats

Get detailed model statistics including regression coefficients.

```
GET /v2/model/stats

→ 200 OK
{
  "model_name": "linear",
  "learning_type": "online",
  "total_observations": 156,
  "tasks": {
    "video_encode": {
      "task": "video_encode",
      "count": 42,
      "avg_cpu_delta": 25.3,
      "avg_mem_delta": 12.5,
      "coefficients": {
        "cpu_a": 0.25,
        "cpu_b": 5.0,
        "mem_a": 0.12,
        "mem_b": 3.0
      }
    }
  }
}
```

---

### GET /v2/scheduler/stats

Get model retraining scheduler statistics.

```
GET /v2/scheduler/stats

→ 200 OK
{
  "running": true,
  "interval": "1h0m0s",
  "retrain_count": 24,
  "last_retrain": "2026-02-21T10:00:00Z",
  "last_error": ""
}
```

---

### POST /v2/scheduler/retrain

Force immediate model retraining.

```
POST /v2/scheduler/retrain

→ 200 OK
{"success": true}

→ 500 Internal Server Error
retrain operation failed
```

---

## Debug Endpoints

Available when `debug.enabled: true`. Requires authentication.

### POST /debug/inject-metrics

Inject test metrics (for testing only).

### GET /debug/status

Get debug status information.

---

## Profiling Endpoints

Available when `server.profiling.enabled: true`. Requires authentication.

- `GET /debug/pprof/` — pprof index
- `GET /debug/pprof/heap` — heap profile
- `GET /debug/pprof/goroutine` — goroutine profile
- `GET /debug/pprof/profile` — CPU profile
- `GET /debug/pprof/trace` — execution trace

---

## Error Responses

| Status | Description |
|--------|-------------|
| `400 Bad Request` | Invalid request body |
| `401 Unauthorized` | Missing or invalid auth |
| `404 Not Found` | Resource not found |
| `503 Service Unavailable` | No capacity / not ready |

Error response format:

```
HTTP/1.1 400 Bad Request
Content-Type: text/plain

invalid request body
```

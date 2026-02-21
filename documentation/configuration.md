# Configuration

Complete configuration reference for **capfox**.

## Config File

**capfox** uses YAML configuration files.

**Locations (in order of precedence):**
1. Path specified via `--config` flag
2. `./capfox.yaml` (current directory)
3. Built-in defaults

## Full Example

```yaml
server:
  host: "0.0.0.0"
  port: 8080
  pid_file: "/var/run/capfox.pid"
  shutdown_timeout_sec: 25
  rate_limit:
    enabled: false
    requests_per_second: 100
    burst: 200
  profiling:
    enabled: false

auth:
  enabled: false
  user: ""
  password: ""

thresholds:
  cpu:
    max_percent: 80
  memory:
    max_percent: 85
  gpu:
    max_percent: 90
  vram:
    max_percent: 85
  storage:
    min_free_gb: 10

monitoring:
  interval_ms: 1000
  paths:
    - "/"

persistence:
  data_dir: "/var/lib/capfox"
  flush_interval_sec: 600

logging:
  level: "info"
  format: "json"

learning:
  model: "moving_average"
  observation_delay_sec: 5

decision:
  strategy: "predictive"
  model: "linear"
  fallback_strategy: "threshold"
  min_observations: 5
  safety_buffer_percent: 10
  model_params:
    alpha: 0.2

debug:
  enabled: false
  auth:
    token: ""
```

## Sections

### Server

HTTP server configuration.

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `host` | string | `0.0.0.0` | Listen address |
| `port` | int | `8080` | Listen port (1-65535) |
| `pid_file` | string | `/var/run/capfox.pid` | PID file path |
| `shutdown_timeout_sec` | int | `25` | Graceful shutdown timeout |

**Rate Limiting:**

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `rate_limit.enabled` | bool | `false` | Enable rate limiting |
| `rate_limit.requests_per_second` | float | `100` | Request rate limit |
| `rate_limit.burst` | int | `200` | Burst allowance |

**Profiling:**

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `profiling.enabled` | bool | `false` | Enable pprof endpoints |

Profiling endpoints require authentication when enabled.

---

### Auth

Basic authentication for API endpoints.

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `enabled` | bool | `false` | Enable authentication |
| `user` | string | | Username (required if enabled) |
| `password` | string | | Password (required if enabled) |

```yaml
auth:
  enabled: true
  user: "admin"
  password: "${CAPFOX_AUTH_PASSWORD}"
```

---

### Thresholds

Resource limits for capacity decisions.

**CPU:**

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `cpu.max_percent` | float | `80` | Max CPU usage (0-100) |

**Memory:**

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `memory.max_percent` | float | `85` | Max memory usage (0-100) |

**GPU:**

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `gpu.max_percent` | float | `90` | Max GPU usage (0-100) |

**VRAM:**

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `vram.max_percent` | float | `85` | Max VRAM usage (0-100) |

**Storage:**

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `storage.min_free_gb` | float | `10` | Minimum free disk space |

---

### Monitoring

Resource collection settings.

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `interval_ms` | int | `1000` | Poll interval (min 100ms) |
| `paths` | []string | `["/"]` | Disk paths to monitor |

```yaml
monitoring:
  interval_ms: 500
  paths:
    - "/"
    - "/data"
    - "/var/lib/datasets"
```

---

### Persistence

Data storage for learning engine.

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `data_dir` | string | `/var/lib/capfox` | Data directory |
| `flush_interval_sec` | int | `600` | Save interval (seconds) |

Learned task statistics are persisted to disk and restored on restart.

---

### Logging

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `level` | string | `info` | Log level: `debug`, `info`, `warn`, `error` |
| `format` | string | `json` | Log format: `json`, `text` |

---

### Learning

Learning engine configuration.

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `model` | string | `moving_average` | Model type: `moving_average`, `linear_regression` |
| `observation_delay_sec` | int | `5` | Delay before measuring resource impact |

The observation delay allows the system to stabilize after a task starts before measuring its resource impact.

---

### Decision

Decision engine configuration.

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `strategy` | string | `predictive` | Strategy: `threshold`, `predictive`, `conservative` |
| `model` | string | `linear` | Prediction model: `none`, `moving_average`, `linear` |
| `fallback_strategy` | string | `threshold` | Fallback when insufficient data |
| `min_observations` | int | `5` | Min observations before prediction |
| `safety_buffer_percent` | float | `10` | Extra buffer for conservative strategy |

**Strategies:**

- `threshold` — Static limits only. No learning required.
- `predictive` — Uses learned task impact to predict resource usage.
- `conservative` — Predictive + safety buffer.

**Model params:**

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `model_params.alpha` | float | `0.2` | Smoothing factor for moving average (0.1-0.3) |

---

### Debug

Debug mode for development and testing.

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `enabled` | bool | `false` | Enable debug endpoints |
| `auth.token` | string | | Bearer token for debug endpoints |

Debug endpoints (like `/debug/inject-metrics`) require authentication when enabled.

---

## Environment Variables

Use `${VAR_NAME}` syntax in config files:

```yaml
auth:
  enabled: true
  user: "${CAPFOX_USER}"
  password: "${CAPFOX_PASSWORD}"

persistence:
  data_dir: "${CAPFOX_DATA_DIR}"
```

**Common variables:**

| Variable | Description |
|----------|-------------|
| `CAPFOX_USER` | Auth username |
| `CAPFOX_PASSWORD` | Auth password |
| `CAPFOX_DATA_DIR` | Data directory path |

---

## Validation

Configuration is validated on load. Invalid configs produce errors:

```
error: config validation failed: thresholds: cpu.max_percent must be between 0 and 100
```

Use `capfox config --validate` to check config without starting the server.

---

## Hot Reload

Send SIGHUP to reload configuration:

```bash
kill -HUP $(cat /var/run/capfox.pid)
# or
capfox reload
```

**What reloads:**
- Thresholds (cpu, memory, gpu, vram, storage limits)
- Auth settings (user, password, enabled)
- The new config is validated before applying

**What does NOT reload (requires restart):**
- Server host/port
- Monitoring paths
- Data directory
- Monitoring interval

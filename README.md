# Capfox

Server-side monitoring utility for tracking system resources and managing task capacity.

## Features

- Monitor CPU, Memory, GPU, VRAM, Storage, Processes
- REST API for capacity queries (`/ask`)
- Learning engine for task impact prediction
- Hot-reload configuration via SIGHUP
- Basic Auth support
- Graceful degradation (works without GPU)

## Quick Start

### Build from source

```bash
go build -o capfox ./cmd/capfox
./capfox start --config configs/capfox.example.yaml
```

### Docker

```bash
docker compose up -d
```

## CLI Commands

```bash
capfox start           # Start server
capfox stop            # Stop server (via PID file)
capfox reload          # Reload configuration (SIGHUP)
capfox config          # Show/validate configuration
capfox status          # Get resource status
capfox ask <task>      # Check if task can run
capfox notify <task>   # Notify task start
capfox stats           # Get learning statistics
```

## API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/` | Server info |
| GET | `/health` | Health check (no auth) |
| GET | `/status` | Current resource metrics |
| POST | `/ask` | Check task capacity |
| POST | `/task/notify` | Notify task start |
| GET | `/stats` | Learning statistics |

## Configuration

```yaml
server:
  host: "0.0.0.0"
  port: 8080
  pid_file: "/var/run/capfox.pid"

auth:
  enabled: false
  user: "admin"
  password: "secret"

thresholds:
  cpu:
    max_percent: 80
  memory:
    max_percent: 85
  gpu:
    max_percent: 90
  vram:
    max_percent: 90
  storage:
    min_free_gb: 10

monitoring:
  interval_ms: 1000
  paths:
    - "/"

persistence:
  data_dir: "./data"
  flush_interval_sec: 60

logging:
  level: "info"
  format: "text"

learning:
  model: "moving_average"
  observation_delay_sec: 5
```

## Example Usage

### Check if task can run

```bash
# CLI
capfox ask video_encoding --complexity 100 --reason

# curl
curl -X POST http://localhost:8080/ask \
  -H "Content-Type: application/json" \
  -d '{"task": "video_encoding", "complexity": 100}'
```

Response:
```json
{"allowed": true}
```

Or if denied:
```json
{"allowed": false, "reasons": ["cpu_overload", "memory_overload"]}
```

### Notify task start

```bash
capfox notify ml_training --complexity 500
```

### Get statistics

```bash
capfox stats
capfox stats ml_training
```

## Hot Reload

```bash
# Edit config file, then:
capfox reload

# Or send SIGHUP directly:
kill -HUP $(cat /var/run/capfox.pid)
```

Reloadable settings: auth, thresholds.
Non-reloadable (require restart): host, port.

## License

MIT

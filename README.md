# 🦊 Capfox

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![Release](https://img.shields.io/github/v/release/haskel/capfox)](https://github.com/haskel/capfox/releases)

**Run or wait? Guard for hungry server tasks — ask before OOM kills**

Capfox monitors system resources (CPU, Memory, GPU, Storage) and predicts whether your server can handle incoming tasks — before they start.

## ✨ Features

- **Real-time Monitoring** — CPU, Memory, GPU, VRAM, Storage, Processes
- **Predictive Capacity Planning** — ML models (Linear, Polynomial, Gradient Boosting)
- **Task Impact Learning** — Learns resource impact of task types over time
- **Decision Strategies** — Threshold, Predictive, Conservative, Queue-aware
- **REST API** — Simple HTTP API for integration
- **TUI Dashboard** — Terminal UI for real-time monitoring
- **Hot Reload** — Configuration reload without restart (SIGHUP)
- **Graceful Degradation** — Works without GPU

## 🚀 Quick Start

### Installation

**From releases:**
```bash
# Linux (amd64)
curl -sSL https://github.com/haskel/capfox/releases/latest/download/capfox_linux_amd64.tar.gz | tar xz
sudo mv capfox /usr/local/bin/

# macOS (Apple Silicon)
curl -sSL https://github.com/haskel/capfox/releases/latest/download/capfox_darwin_arm64.tar.gz | tar xz
sudo mv capfox /usr/local/bin/
```

**From source:**
```bash
git clone https://github.com/haskel/capfox.git
cd capfox
make build
./bin/capfox --help
```

**Docker:**
```bash
docker compose up -d
```

### Run

```bash
# Start the server
capfox start

# With custom config
capfox start --config /path/to/config.yaml

# Check system status
capfox status

# Open TUI dashboard
capfox tui
```

## 📡 API

### Check Capacity

Ask if the system can handle a task:

```bash
curl -X POST http://localhost:8080/ask \
  -H "Content-Type: application/json" \
  -d '{"task": "video_encoding", "complexity": 1.5}'
```

Response:
```json
{
  "allowed": true,
  "task": "video_encoding"
}
```

If denied:
```json
{
  "allowed": false,
  "reasons": ["cpu_overload", "memory_overload"]
}
```

### Get System Status

```bash
curl http://localhost:8080/status
```

### Notify Task Start

Help Capfox learn task impact:

```bash
curl -X POST http://localhost:8080/task/start \
  -H "Content-Type: application/json" \
  -d '{"task": "video_encoding", "complexity": 1.5}'
```

### Endpoints

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/` | Service info |
| `GET` | `/health` | Health check |
| `GET` | `/ready` | Readiness check |
| `GET` | `/status` | Current system metrics |
| `POST` | `/ask` | Check capacity for task |
| `POST` | `/task/start` | Notify task start |
| `GET` | `/stats` | Task statistics |

## 🖥️ TUI Dashboard

```
┌─────────────────────────────────────────────────────────────────┐
│  CAPFOX DASHBOARD                          ↻ 1s │ q:quit r:ref │
├─────────────────────────────────────────────────────────────────┤
│  CPU [████████████░░░░░░░░] 65.2%    Memory [██████░░░░] 48.3%  │
├─────────────────────────────────────────────────────────────────┤
│  GPU 0: NVIDIA RTX 4090                                         │
│  Usage [████████░░░░] 35.0%    VRAM [██████████░░] 12.4/24.0 GB │
├─────────────────────────────────────────────────────────────────┤
│  Task Statistics                                                │
│  Task              │ Count │ CPU Δ  │ Mem Δ  │ GPU Δ           │
│  video_encoding    │   142 │ +15.2% │  +8.3% │ +45.0%          │
│  ml_training       │    53 │  +2.8% │ +12.5% │ +68.4%          │
└─────────────────────────────────────────────────────────────────┘
```

```bash
capfox tui --refresh 500ms
```

## 🔧 CLI Commands

```bash
capfox start    # Start the server
capfox stop     # Stop the server
capfox status   # Show system status
capfox stats    # Show task statistics
capfox ask      # Check task capacity
capfox reload   # Reload configuration
capfox tui      # Open TUI dashboard
capfox config   # Show current config
```

## ⚙️ Configuration

```yaml
server:
  host: "0.0.0.0"
  port: 8080

thresholds:
  cpu:
    max_percent: 80
  memory:
    max_percent: 85
  gpu:
    max_percent: 90
  storage:
    min_free_gb: 10

decision:
  strategy: "predictive"  # threshold, predictive, conservative, queue_aware
  model: "linear"         # none, moving_average, linear, polynomial, gradient_boosting
  min_observations: 10
  safety_buffer_percent: 10

monitoring:
  interval_ms: 1000
  paths:
    - "/"

logging:
  level: "info"
  format: "json"
```

See [configs/capfox.example.yaml](configs/capfox.example.yaml) for full configuration.

### Hot Reload

```bash
# Edit config, then:
capfox reload

# Or send SIGHUP:
kill -HUP $(cat /var/run/capfox.pid)
```

**Reloadable:** auth, thresholds
**Requires restart:** host, port

## 🏗️ Architecture

```
┌─────────────┐    ┌─────────────┐    ┌─────────────┐
│   Monitors  │───▶│  Aggregator │───▶│  Capacity   │
│ CPU/Mem/GPU │    │             │    │  Manager    │
└─────────────┘    └─────────────┘    └──────┬──────┘
                                             │
┌─────────────┐    ┌─────────────┐    ┌──────▼──────┐
│  Learning   │◀───│  Decision   │◀───│   Server    │
│   Engine    │    │   Engine    │    │  (REST API) │
└─────────────┘    └─────────────┘    └─────────────┘
```

## 🤝 Contributing

Contributions are welcome! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## 📄 License

MIT License — see [LICENSE](LICENSE) for details.

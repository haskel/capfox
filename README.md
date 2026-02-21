# 🦊 capfox

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![Tests](https://img.shields.io/github/actions/workflow/status/haskel/capfox/ci.yml?label=tests)](https://github.com/haskel/capfox/actions)
[![Go Report Card](https://goreportcard.com/badge/github.com/haskel/capfox)](https://goreportcard.com/report/github.com/haskel/capfox)
[![Release](https://img.shields.io/github/v/release/haskel/capfox)](https://github.com/haskel/capfox/releases)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

**Capacity check for resource-heavy tasks on standalone servers**

Single node. No Kubernetes. No Slurm. No queues. Just ask.

*CPU · RAM · GPU · VRAM · Disk*

A lightweight utility that knows and predicts if your server can handle another consuming task. Ask before launch — get `{"allowed": true}` or `{"allowed": false}`.

---

**Contents:**
- ⚙️ [When You Need This](#️-when-you-need-this)
- 🔧 [Core Concepts](#-core-concepts)
- 🚀 [Quick Start](#-quick-start)
- 📚 [Full Documentation](documentation/readme.md)

---

## ⚙️ When You Need This

Running heavy tasks on a single powerful server?

### Use Cases

- **OOM Killer** — memory exhaustion kills processes mid-execution
- **Swap thrashing** — RAM fills up, system starts swapping, everything slows to crawl
- **Disk fills up** — video encoding, dataset downloads, ML checkpoints, Jupyter outputs eat storage fast
- **GPU/VRAM exhaustion** — CUDA OOM, driver crashes, silent failures
- **Threshold guessing** — want to utilize the machine fully, but hard to know the safe limits

### Why not use X?

- **Orchestration (K8s, Slurm, Nomad)** — overkill for 1-2 servers
- **Job queues (Celery, RabbitMQ)** — queue still needs capacity info to schedule tasks
- **Monitoring stacks (Prometheus + Grafana)** — too heavy for dev/experimental setups

## 🔧 Core Concepts

### Capacity Check

Ask **capfox** if there's capacity for your task. It answers yes or no — you decide what to do.

**HTTP API:**

```
POST /ask
{"task": "video_encode", "complexity": 30}

→ 200 {"allowed": true}
→ 503 {"allowed": false, "reasons": ["cpu_overload"]}
```

**CLI:**

```bash
capfox ask video_encode --complexity 30
# exit 75 = no capacity
```

**Wrapper** (check + run + notify):

```bash
capfox run --task video_encode --complexity 30 ./encode.sh
# exit 75 = no capacity, task not started
# automatically sends notify on start
```

---

### Complexity Points

You define what points mean for your workloads. Any positive integer.

Complexity is defined **per task type**. Each task type has its own scale:
- `video_encode` — one scale (resolution, bitrate)
- `data_processing` — another scale (file size, row count)
- `gpu_render` — yet another scale

**capfox** builds separate predictions for each task type.

#### Examples

**Video encoding** (task: `video_encode`):

| Resolution | Complexity |
|------------|------------|
| 720p | 10 |
| 1080p | 15 |
| 2160p (4K) | 30 |

Or use bitrate directly: `--complexity 8000` for 8 Mbps.

**GPU tasks** (task: `gpu_processing`):

| Task | Complexity |
|------|------------|
| Video transcode (NVENC) | 30 |
| Batch image resize (CUDA) | 20 |
| Embedding generation | 40 |
| Data augmentation pipeline | 25 |

**RAM-heavy tasks** (task: `data_processing`):

| Task | Complexity |
|------|------------|
| CSV processing (100MB) | 10 |
| CSV processing (1GB) | 50 |
| Log analysis job | 40 |
| Report generation | 30 |

---

### Making Decisions

**Threshold-based (simple):**

Static limits — if CPU > 80% or RAM > 85%, deny new tasks. No learning required.

**Predictive:**

**capfox** learns from history. Send notifications when tasks start.

API:

```
POST /task/notify
{"task": "video_encode", "complexity": 30}
```

CLI:

```bash
capfox notify video_encode --complexity 30
```

Wrapper (automatic):

```bash
capfox run --task video_encode --complexity 30 ./encode.sh
# sends notify automatically when complexity is specified
```

Over time, **capfox** understands how complexity affects resources and predicts impact of new tasks.

---

### How Prediction Works

1. **Current state** — **capfox** knows current CPU, RAM, GPU, disk usage
2. **Thresholds** — configured limits (e.g., CPU < 80%)
3. **History** — previous tasks with their complexity and resource impact
4. **Prediction** — estimates how new task will affect resources (linear, moving average, and experimental models available)
5. **Decision** — if (current + predicted) > threshold → no capacity

**Important:** For prediction to improve, send `notify` when tasks start with their complexity.

## 🚀 Quick Start

### Install

**From source:**

```bash
git clone https://github.com/haskel/capfox.git
cd capfox
make build
sudo mv bin/capfox /usr/local/bin/
```

**From release:**

```bash
# Linux (amd64)
curl -sSL https://github.com/haskel/capfox/releases/latest/download/capfox_linux_amd64.tar.gz | tar xz
sudo mv capfox /usr/local/bin/

# macOS (Apple Silicon)
curl -sSL https://github.com/haskel/capfox/releases/latest/download/capfox_darwin_arm64.tar.gz | tar xz
sudo mv capfox /usr/local/bin/
```

### Run Server

```bash
capfox start
# starts monitoring system resources
# accepts task notifications via API on port 9329
```

📖 All CLI commands: [documentation/cli-commands.md](documentation/cli-commands.md)

### Check Capacity

**CLI:**

```bash
capfox ask video_encode --complexity 50
# exit 0 = capacity available
# exit 75 = no capacity
```

**API:**

```
POST /ask
{"task": "video_encode", "complexity": 50}

→ 200 OK
{"allowed": true}

→ 503 Service Unavailable
{"allowed": false, "reasons": ["cpu_overload"]}
```

📖 Full API reference: [documentation/api.md](documentation/api.md)

### Wrapper

Like `time` and `nice` — wrap any command. Useful for cron tasks.

```bash
capfox run --task video_encode --complexity 50 ./encode.sh
# exit 0 = command completed
# exit 75 = no capacity, command not started
```

📖 Wrapper details: [documentation/run-command.md](documentation/run-command.md)

---

## 👉 [Full Documentation](documentation/readme.md)

## 🤝 Contributing

Contributions welcome! See [CONTRIBUTING.md](CONTRIBUTING.md).

## 📄 License

MIT License — see [LICENSE](LICENSE).

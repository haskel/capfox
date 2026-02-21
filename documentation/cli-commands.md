# CLI Commands

Complete reference for **capfox** command-line interface.

## Global Flags

Available for all commands:

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--config` | `-c` | string | | Config file path |
| `--host` | | string | `localhost` | Server host |
| `--port` | `-p` | int | `9329` | Server port |
| `--json` | | bool | `false` | Output in JSON format |
| `--verbose` | `-v` | bool | `false` | Verbose output |
| `--user` | | string | | Auth username |
| `--password` | | string | | Auth password |

## Commands

### capfox start

Start the **capfox** server in foreground mode.

```bash
capfox start
capfox start --port 9329
capfox start --config /etc/capfox/config.yaml
```

The server:
- Starts all resource monitors (CPU, memory, GPU, storage)
- Loads persisted learning data
- Listens for HTTP requests
- Writes PID file (if configured)
- Handles SIGHUP for config reload
- Handles SIGINT/SIGTERM for graceful shutdown

---

### capfox stop

Stop the running server by sending SIGTERM.

```bash
capfox stop
capfox stop --pid-file /var/run/capfox.pid
```

| Flag | Type | Description |
|------|------|-------------|
| `--pid-file` | string | PID file path (overrides config) |

Reads PID from file and sends SIGTERM to the process.

---

### capfox status

Get current resource metrics from the server.

```bash
capfox status
capfox status --json
capfox status --host 10.0.0.1 --port 9329
```

Output (human-readable):

```
=== System Status ===

CPU:
  Usage: 45.2%

Memory:
  Usage: 62.1%
  Total: 32.0 GB
  Used:  19.9 GB

Storage:
  /: 150.3 GB free / 500.0 GB total

GPU:
  GPU 0: 30.0% usage, VRAM: 4.0 / 24.0 GB

Processes:
  Total: 342
  Threads: 1256
```

Output (JSON):

```json
{
  "cpu": {"usage_percent": 45.2},
  "memory": {"usage_percent": 62.1, "total_bytes": 34359738368, "used_bytes": 21367234560},
  "storage": {"/": {"total_bytes": 536870912000, "free_bytes": 161395916800}},
  "gpus": [{"usage_percent": 30.0, "vram_used_bytes": 4294967296, "vram_total_bytes": 25769803776}]
}
```

---

### capfox ask

Ask if a task can run based on current capacity.

```bash
capfox ask video_encode
capfox ask video_encode --complexity 100
capfox ask ml_training --complexity 500 --reason
capfox ask batch_job --cpu 50 --mem 30
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--complexity` | int | `0` | Task complexity (for prediction) |
| `--reason` | bool | `false` | Show denial reasons |
| `--cpu` | float64 | `0` | Estimated CPU usage % |
| `--mem` | float64 | `0` | Estimated memory usage % |
| `--gpu` | float64 | `0` | Estimated GPU usage % |
| `--vram` | float64 | `0` | Estimated VRAM usage % |

**Exit codes:**
- `0` — task allowed
- `1` — task denied

Output (allowed):

```
✓ Task 'video_encode' is ALLOWED
```

Output (denied with reasons):

```
✗ Task 'video_encode' is DENIED
Reasons:
  - cpu_high
  - memory_high
```

---

### capfox notify

Notify server that a task has started. Used for learning.

```bash
capfox notify video_encode
capfox notify ml_training --complexity 500
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--complexity` | int | `0` | Task complexity |

The learning engine observes resource changes after notification and builds prediction models.

Output:

```
✓ Task 'video_encode' notification received
```

---

### capfox run

Wrapper command — check capacity and run command if allowed.

```bash
capfox run ./script.sh
capfox run --task video_encode ffmpeg -i in.mp4 out.webm
capfox run --complexity 100 make build
capfox run --cpu 50 --mem 30 ./heavy.sh
capfox run --quiet ./build.sh
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--task` | string | command name | Task name for /ask |
| `--complexity` | int | `0` | Task complexity |
| `--cpu` | float64 | `0` | Estimated CPU usage % |
| `--mem` | float64 | `0` | Estimated memory usage % |
| `--gpu` | float64 | `0` | Estimated GPU usage % |
| `--vram` | float64 | `0` | Estimated VRAM usage % |
| `--reason` | bool | `false` | Show denial reasons |
| `--quiet` | bool | `false` | Suppress capfox output |

**Exit codes:**
- `0-125` — command's exit code
- `75` — no capacity (EX_TEMPFAIL, command not started)
- `126` — command not executable
- `127` — command not found

**Behavior:**
1. Calls `/ask` to check capacity
2. If denied → exit 75
3. If complexity > 0 → sends `/task/notify`
4. Executes command with stdin/stdout/stderr passthrough
5. Returns command's exit code

**Fail-open:** If server is unreachable, the command runs anyway.

---

### capfox stats

Get task statistics from the learning engine.

```bash
capfox stats
capfox stats video_encode
capfox stats --json
```

Output (all tasks):

```
=== Task Statistics ===
Total observations: 156

Task: video_encode
  Observations: 42
  Avg CPU delta:  +25.30%
  Avg Memory delta: +12.50%

Task: ml_training
  Observations: 114
  Avg CPU delta:  +45.80%
  Avg Memory delta: +35.20%
  Avg GPU delta:  +80.10%
  Avg VRAM delta: +65.40%
```

Output (single task):

```
Task: video_encode
  Observations: 42
  Avg CPU delta:  +25.30%
  Avg Memory delta: +12.50%
```

---

### capfox config

Show or validate current configuration.

```bash
capfox config
capfox config --validate
capfox config --json
capfox config --config /etc/capfox/config.yaml
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--validate` | bool | `false` | Only validate, don't print |

Default output is YAML. Use `--json` for JSON format.

---

### capfox reload

Reload server configuration (hot reload).

```bash
capfox reload
capfox reload --pid-file /var/run/capfox.pid
```

| Flag | Type | Description |
|------|------|-------------|
| `--pid-file` | string | PID file path (overrides config) |

Sends SIGHUP to the server process. The server validates the new config before applying.

**What reloads:**
- Thresholds (CPU, memory, GPU, VRAM, storage)
- Auth settings (enabled, user, password)

**What doesn't reload (requires restart):**
- Server host/port
- Monitoring interval/paths
- Data directory
- Rate limiting
- Decision engine settings
- Logging level

---

### capfox tui

Launch interactive terminal dashboard.

```bash
capfox tui
capfox tui --refresh 500ms
capfox tui --host 10.0.0.1 --port 9329
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--refresh` | duration | `1s` | Dashboard refresh interval |

See [TUI Dashboard](tui-dashboard.md) for details.

---

## Examples

### Basic workflow

```bash
# Start server
capfox start &

# Check if task can run
capfox ask video_encode --complexity 50
# exit 0 = allowed

# Run with wrapper
capfox run --task video_encode --complexity 50 ./encode.sh

# Check learned statistics
capfox stats video_encode

# Stop server
capfox stop
```

### Remote server

```bash
# Connect to remote server
capfox status --host 10.0.0.5 --port 9329

# With authentication
capfox ask video_encode --user admin --password secret
```

### CI/CD integration

```bash
# In CI pipeline - fail if no capacity
capfox run --task ci_build --quiet make build || exit 1

# With custom exit handling
if ! capfox run --task deploy ./deploy.sh; then
    if [ $? -eq 75 ]; then
        echo "No capacity, will retry later"
        exit 0
    fi
    exit 1
fi
```

### Cron tasks

```bash
# crontab entry
0 * * * * /usr/local/bin/capfox run --task hourly_backup --quiet /opt/scripts/backup.sh
```

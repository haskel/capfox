# Run Wrapper

**capfox run** — wrap any command with capacity check.

Works like `time` and `nice` — a wrapper you put before any command:

```bash
time ./script.sh           # shows time after execution
nice ./script.sh           # runs with lower priority
capfox run ./script.sh     # checks capacity, then runs or refuses
```

## Usage

```bash
capfox run [flags] <command> [args...]
```

## Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--task` | string | command name | Task name for capacity check |
| `--complexity` | int | `0` | Task complexity (for prediction) |
| `--cpu` | float64 | `0` | Estimated CPU usage % |
| `--mem` | float64 | `0` | Estimated memory usage % |
| `--gpu` | float64 | `0` | Estimated GPU usage % |
| `--vram` | float64 | `0` | Estimated VRAM usage % |
| `--reason` | bool | `false` | Show denial reasons |
| `--quiet` | bool | `false` | Suppress capfox output |

## Exit Codes

| Code | Meaning |
|------|---------|
| `0-125` | Command's exit code (passthrough) |
| `75` | No capacity (EX_TEMPFAIL from sysexits.h) |
| `126` | Command not executable |
| `127` | Command not found |

Code 75 (`EX_TEMPFAIL`) indicates a temporary failure — the command wasn't started because capacity is unavailable. This is a standard exit code for "try again later".

---

## How It Works

```
capfox run --task build ./build.sh
         │
         ▼
    ┌─────────────────────┐
    │  1. Check capacity  │◄── POST /ask
    │     (call server)   │
    └──────────┬──────────┘
               │
       ┌───────┴───────┐
       ▼               ▼
   ALLOWED          DENIED
       │               │
       │               └─► exit 75
       ▼
    ┌─────────────────────┐
    │  2. Send notify     │◄── POST /task/notify
    │  (if complexity>0)  │    (fire-and-forget)
    └──────────┬──────────┘
               │
               ▼
    ┌─────────────────────┐
    │  3. Execute command │
    │  (with passthrough) │
    └──────────┬──────────┘
               │
               ▼
    ┌─────────────────────┐
    │  4. Return exit     │
    │     code            │
    └─────────────────────┘
```

### Fail-Open Behavior

If the **capfox** server is unreachable, the command runs anyway:

```bash
# Server not running
capfox run ./script.sh
# capfox: failed to check capacity: connection refused
# (command still runs)
```

This is by design — capacity checking is a safety net, not a gate. If the net is down, you still want your tasks to run.

---

## Task Name

If `--task` is not specified, the wrapper uses the command name:

```bash
capfox run ./encode.sh
# task name = "encode.sh"

capfox run --task video_encode ./encode.sh
# task name = "video_encode"

capfox run python train.py
# task name = "python"

capfox run --task ml_training python train.py
# task name = "ml_training"
```

Use `--task` when:
- The command name isn't descriptive (`python`, `bash`, `sh`)
- You want consistent task names across different scripts
- You're grouping related commands under one task type

---

## Notify Integration

When complexity is specified (`--complexity > 0`), the wrapper automatically sends a notification after capacity is approved:

```bash
capfox run --task video_encode --complexity 30 ./encode.sh
```

This:
1. Calls `/ask` with task=video_encode, complexity=30
2. If allowed, calls `/task/notify` with same parameters
3. Runs `./encode.sh`

The notification enables the learning engine to observe resource impact.

**Without complexity:**

```bash
capfox run ./script.sh
```

This only checks current thresholds — no notification, no learning.

---

## Examples

### Basic usage

```bash
# Simple check + run
capfox run ./heavy-task.sh

# With task name
capfox run --task video_encode ./encode.sh

# With complexity for learning
capfox run --task video_encode --complexity 30 ./encode.sh
```

### Resource estimates

```bash
# Estimate 50% CPU, 30% memory
capfox run --cpu 50 --mem 30 ./process-data.sh

# GPU task
capfox run --gpu 80 --vram 70 ./train-model.py
```

### Output control

```bash
# Show denial reasons
capfox run --reason ./heavy-task.sh
# capfox: denied
#   - cpu_high
#   - memory_high

# Suppress all capfox output
capfox run --quiet ./script.sh
```

### Cron integration

```bash
# crontab -e
0 * * * * /usr/local/bin/capfox run --task hourly_backup --quiet /opt/scripts/backup.sh

# Daily heavy processing
0 3 * * * /usr/local/bin/capfox run --task daily_report --complexity 100 /opt/scripts/generate-report.sh
```

### CI/CD pipelines

```bash
# GitHub Actions / GitLab CI
- name: Build with capacity check
  run: capfox run --task ci_build --quiet make build || exit 1
```

### Exit code handling

```bash
# Handle denial separately
if ! capfox run --task deploy ./deploy.sh; then
    if [ $? -eq 75 ]; then
        echo "No capacity available, will retry later"
        exit 0  # Don't fail the pipeline
    fi
    exit 1  # Real failure
fi
```

### Shell script integration

```bash
#!/bin/bash

# Check before starting heavy work
capfox run --task data_import --complexity 50 ./import.sh
status=$?

if [ $status -eq 75 ]; then
    echo "Server busy, scheduling for later"
    at now + 30 minutes <<< "$0"
    exit 0
fi

if [ $status -ne 0 ]; then
    echo "Import failed with exit code $status"
    exit $status
fi

echo "Import completed successfully"
```

### Retry loop

```bash
#!/bin/bash

MAX_RETRIES=5
RETRY_DELAY=60

for i in $(seq 1 $MAX_RETRIES); do
    capfox run --task heavy_job --quiet ./process.sh
    status=$?

    if [ $status -eq 0 ]; then
        echo "Success"
        exit 0
    elif [ $status -eq 75 ]; then
        echo "Attempt $i: No capacity, waiting ${RETRY_DELAY}s..."
        sleep $RETRY_DELAY
    else
        echo "Failed with exit code $status"
        exit $status
    fi
done

echo "Max retries reached"
exit 1
```

# TUI Dashboard

Terminal UI for real-time monitoring of **capfox** server.

## Launch

```bash
capfox tui
capfox tui --refresh 500ms
capfox tui --host 10.0.0.1 --port 9329
```

## Options

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--refresh` | duration | `1s` | Refresh interval |
| `--host` | string | `localhost` | Server host |
| `--port` | int | `8080` | Server port |
| `--user` | string | | Auth username |
| `--password` | string | | Auth password |

---

## Interface

```
CAPFOX DASHBOARD                        ↻ 1s | q:quit r:refresh ↑↓:scroll
  CPU    [████████░░░░░░░░░░░░]  42.5%    Memory [██████████████░░░░░░]  68.2%

  GPU 0: NVIDIA GeForce RTX 3090
  Usage  [████████░░░░]  65.3%    VRAM   [███████████░]  91.2% 22.0/24.0 GB

  Storage
  /      [███████████████░░░░░]  75.2%  (150.3 / 200.0 GB)
  /data  [██████████░░░░░░░░░░]  48.5%  (485.0 / 1000.0 GB)

  Task Statistics
  Task                 │  Count │   CPU Δ │   Mem Δ │   GPU Δ
  video_encode         │     42 │  +25.3% │  +12.5% │       -
  ml_training          │    114 │  +45.8% │  +35.2% │  +80.1%
  data_processing      │     28 │  +15.2% │  +22.1% │       -
  [1-3 of 5 tasks]

  Processes: 342 │ Threads: 1,256 │ Updated: 14:32:15
```

---

## Keyboard Shortcuts

| Key | Action |
|-----|--------|
| `q` | Quit dashboard |
| `Ctrl+C` | Quit dashboard |
| `r` | Force refresh |
| `↑` / `k` | Scroll task table up |
| `↓` / `j` | Scroll task table down |

---

## Sections

### Title Bar

Shows:
- Dashboard title
- Current refresh interval
- Loading indicator
- Keyboard shortcut hints

### CPU & Memory

Progress bars with color coding:
- Green (0-60%) — normal
- Yellow (60-80%) — warning
- Red (80-100%) — critical

### GPU

Displayed only if NVIDIA GPU is present. Shows:
- GPU index and name
- Usage percentage
- VRAM usage (percentage and absolute)

Multiple GPUs are shown separately.

### Storage

Shows each monitored path:
- Mount point
- Usage percentage
- Used / Total space in GB

Paths are sorted alphabetically.

### Task Statistics

Table showing learned task statistics:
- Task name
- Observation count
- Average CPU delta (%)
- Average memory delta (%)
- Average GPU delta (%)

Tasks are sorted by observation count (descending). Use ↑/↓ to scroll through the list.

### Footer

Shows:
- Total process count
- Total thread count
- Last update timestamp

---

## Color Coding

Progress bars change color based on usage:

| Usage | Color |
|-------|-------|
| 0-60% | Green |
| 60-80% | Yellow |
| 80-100% | Red |

Delta values in task statistics:
- Positive deltas (+) — green
- Negative deltas (-) — red

---

## Data Sources

The TUI fetches data from:
- `/status` — CPU, memory, GPU, storage metrics
- `/stats` — Task statistics from learning engine

Data is fetched in parallel at the configured refresh interval.

---

## Requirements

- Terminal with color support (truecolor or 256-color)
- Running **capfox** server
- Terminal width at least 80 characters

---

## Examples

### Basic monitoring

```bash
# Start server
capfox start &

# Launch TUI
capfox tui
```

### Fast refresh for debugging

```bash
capfox tui --refresh 200ms
```

### Monitor remote server

```bash
capfox tui --host 192.168.1.100 --port 9329

# With authentication
capfox tui --host 192.168.1.100 --user admin --password secret
```

### SSH forwarding

```bash
# On local machine
ssh -L 9329:localhost:9329 user@server

# Then run TUI locally
capfox tui --port 9329
```

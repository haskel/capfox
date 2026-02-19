#!/bin/bash
# Profiling helper script for Capfox
# Usage: ./scripts/profile.sh [command] [options]

set -e

HOST="${CAPFOX_HOST:-localhost}"
PORT="${CAPFOX_PORT:-8080}"
BASE_URL="http://${HOST}:${PORT}"
OUTPUT_DIR="${OUTPUT_DIR:-./profiles}"

mkdir -p "$OUTPUT_DIR"

usage() {
    cat << EOF
Capfox Profiling Helper

Usage: $0 <command> [options]

Commands:
  cpu [seconds]     Capture CPU profile (default: 30s)
  heap              Capture heap profile
  goroutine         Capture goroutine profile
  allocs            Capture allocations profile
  block             Capture block profile
  mutex             Capture mutex profile
  trace [seconds]   Capture execution trace (default: 5s)
  web <profile>     Open profile in web browser (requires go tool pprof)
  all               Capture all profiles at once

Examples:
  $0 cpu 60           # 60-second CPU profile
  $0 heap             # Current heap profile
  $0 web cpu          # View latest CPU profile in browser
  $0 all              # Capture all profiles

Environment:
  CAPFOX_HOST        Server host (default: localhost)
  CAPFOX_PORT        Server port (default: 8080)
  OUTPUT_DIR         Output directory (default: ./profiles)

EOF
    exit 1
}

timestamp() {
    date +%Y%m%d_%H%M%S
}

capture_cpu() {
    local seconds="${1:-30}"
    local output="${OUTPUT_DIR}/cpu_$(timestamp).pprof"
    echo "Capturing CPU profile for ${seconds}s..."
    curl -s "${BASE_URL}/debug/pprof/profile?seconds=${seconds}" -o "$output"
    echo "Saved to: $output"
    echo "$output"
}

capture_heap() {
    local output="${OUTPUT_DIR}/heap_$(timestamp).pprof"
    echo "Capturing heap profile..."
    curl -s "${BASE_URL}/debug/pprof/heap" -o "$output"
    echo "Saved to: $output"
    echo "$output"
}

capture_goroutine() {
    local output="${OUTPUT_DIR}/goroutine_$(timestamp).pprof"
    echo "Capturing goroutine profile..."
    curl -s "${BASE_URL}/debug/pprof/goroutine" -o "$output"
    echo "Saved to: $output"
    echo "$output"
}

capture_allocs() {
    local output="${OUTPUT_DIR}/allocs_$(timestamp).pprof"
    echo "Capturing allocations profile..."
    curl -s "${BASE_URL}/debug/pprof/allocs" -o "$output"
    echo "Saved to: $output"
    echo "$output"
}

capture_block() {
    local output="${OUTPUT_DIR}/block_$(timestamp).pprof"
    echo "Capturing block profile..."
    curl -s "${BASE_URL}/debug/pprof/block" -o "$output"
    echo "Saved to: $output"
    echo "$output"
}

capture_mutex() {
    local output="${OUTPUT_DIR}/mutex_$(timestamp).pprof"
    echo "Capturing mutex profile..."
    curl -s "${BASE_URL}/debug/pprof/mutex" -o "$output"
    echo "Saved to: $output"
    echo "$output"
}

capture_trace() {
    local seconds="${1:-5}"
    local output="${OUTPUT_DIR}/trace_$(timestamp).out"
    echo "Capturing execution trace for ${seconds}s..."
    curl -s "${BASE_URL}/debug/pprof/trace?seconds=${seconds}" -o "$output"
    echo "Saved to: $output"
    echo "View with: go tool trace $output"
    echo "$output"
}

open_web() {
    local profile_type="$1"
    local latest

    case "$profile_type" in
        cpu)
            latest=$(ls -t "${OUTPUT_DIR}"/cpu_*.pprof 2>/dev/null | head -1)
            ;;
        heap)
            latest=$(ls -t "${OUTPUT_DIR}"/heap_*.pprof 2>/dev/null | head -1)
            ;;
        goroutine)
            latest=$(ls -t "${OUTPUT_DIR}"/goroutine_*.pprof 2>/dev/null | head -1)
            ;;
        allocs)
            latest=$(ls -t "${OUTPUT_DIR}"/allocs_*.pprof 2>/dev/null | head -1)
            ;;
        block)
            latest=$(ls -t "${OUTPUT_DIR}"/block_*.pprof 2>/dev/null | head -1)
            ;;
        mutex)
            latest=$(ls -t "${OUTPUT_DIR}"/mutex_*.pprof 2>/dev/null | head -1)
            ;;
        trace)
            latest=$(ls -t "${OUTPUT_DIR}"/trace_*.out 2>/dev/null | head -1)
            if [ -n "$latest" ]; then
                echo "Opening trace: $latest"
                go tool trace "$latest"
                return
            fi
            ;;
        *)
            echo "Unknown profile type: $profile_type"
            echo "Available: cpu, heap, goroutine, allocs, block, mutex, trace"
            exit 1
            ;;
    esac

    if [ -z "$latest" ]; then
        echo "No $profile_type profile found. Run: $0 $profile_type"
        exit 1
    fi

    echo "Opening: $latest"
    go tool pprof -http=:8081 "$latest"
}

capture_all() {
    echo "Capturing all profiles..."
    echo ""
    capture_heap
    capture_goroutine
    capture_allocs
    capture_block
    capture_mutex
    echo ""
    echo "Starting CPU profile (30s)..."
    capture_cpu 30
    echo ""
    echo "All profiles saved to: $OUTPUT_DIR"
}

# Main
case "${1:-}" in
    cpu)
        capture_cpu "$2"
        ;;
    heap)
        capture_heap
        ;;
    goroutine)
        capture_goroutine
        ;;
    allocs)
        capture_allocs
        ;;
    block)
        capture_block
        ;;
    mutex)
        capture_mutex
        ;;
    trace)
        capture_trace "$2"
        ;;
    web)
        open_web "$2"
        ;;
    all)
        capture_all
        ;;
    *)
        usage
        ;;
esac

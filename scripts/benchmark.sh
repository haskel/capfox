#!/bin/bash
# Load testing / benchmarking script for Capfox
# Usage: ./scripts/benchmark.sh [command] [options]

set -e

HOST="${CAPFOX_HOST:-localhost}"
PORT="${CAPFOX_PORT:-8080}"
BASE_URL="http://${HOST}:${PORT}"
OUTPUT_DIR="${OUTPUT_DIR:-./benchmark_results}"

mkdir -p "$OUTPUT_DIR"

usage() {
    cat << EOF
Capfox Load Testing / Benchmarking

Usage: $0 <command> [options]

Commands:
  ask [count] [concurrency]    Benchmark /ask endpoint
  notify [count] [tasks...]    Send task notifications
  mixed [duration]             Mixed workload simulation
  train [count]                Train model with synthetic data
  compare-models               Compare model prediction accuracy
  report                       Generate benchmark report

Examples:
  $0 ask 1000 10               # 1000 requests, 10 concurrent
  $0 notify 100 video_encoding ml_inference
  $0 mixed 60                  # 60 seconds mixed load
  $0 train 200                 # Train with 200 observations
  $0 compare-models            # Compare model accuracy

Environment:
  CAPFOX_HOST        Server host (default: localhost)
  CAPFOX_PORT        Server port (default: 8080)
  OUTPUT_DIR         Output directory (default: ./benchmark_results)

EOF
    exit 1
}

timestamp() {
    date +%Y%m%d_%H%M%S
}

check_server() {
    if ! curl -s "${BASE_URL}/health" > /dev/null 2>&1; then
        echo "Error: Capfox server not responding at ${BASE_URL}"
        echo "Start with: make profile-start (or ./bin/capfox start)"
        exit 1
    fi
    echo "Server is up at ${BASE_URL}"
}

# Benchmark /ask endpoint
benchmark_ask() {
    local count="${1:-100}"
    local concurrency="${2:-10}"
    local output="${OUTPUT_DIR}/ask_benchmark_$(timestamp).txt"

    check_server

    echo "Benchmarking /ask: $count requests, $concurrency concurrent"
    echo ""

    # Check if hey is installed
    if command -v hey &> /dev/null; then
        hey -n "$count" -c "$concurrency" \
            -m POST \
            -H "Content-Type: application/json" \
            -d '{"task":"benchmark_task","complexity":100}' \
            "${BASE_URL}/ask" | tee "$output"
    # Fallback to ab (Apache Bench)
    elif command -v ab &> /dev/null; then
        echo '{"task":"benchmark_task","complexity":100}' > /tmp/ask_body.json
        ab -n "$count" -c "$concurrency" \
            -T "application/json" \
            -p /tmp/ask_body.json \
            "${BASE_URL}/ask" | tee "$output"
    # Fallback to simple curl loop
    else
        echo "Warning: hey or ab not found, using simple curl loop (slower)"
        local start_time=$(date +%s.%N)
        local success=0
        local failed=0

        for i in $(seq 1 "$count"); do
            if curl -s -X POST "${BASE_URL}/ask" \
                -H "Content-Type: application/json" \
                -d '{"task":"benchmark_task","complexity":100}' > /dev/null; then
                ((success++))
            else
                ((failed++))
            fi

            if (( i % 100 == 0 )); then
                echo "Progress: $i / $count"
            fi
        done

        local end_time=$(date +%s.%N)
        local duration=$(echo "$end_time - $start_time" | bc)
        local rps=$(echo "scale=2; $count / $duration" | bc)

        echo ""
        echo "Results:"
        echo "  Total requests: $count"
        echo "  Successful: $success"
        echo "  Failed: $failed"
        echo "  Duration: ${duration}s"
        echo "  RPS: $rps"
    fi

    echo ""
    echo "Results saved to: $output"
}

# Send task notifications to train the model
benchmark_notify() {
    local count="${1:-100}"
    shift
    local tasks=("${@:-video_encoding ml_inference data_processing}")

    check_server

    echo "Sending $count task notifications..."
    echo "Tasks: ${tasks[*]}"
    echo ""

    local success=0
    local failed=0

    for i in $(seq 1 "$count"); do
        # Random task from list
        local task=${tasks[$((RANDOM % ${#tasks[@]}))]}
        # Random complexity 10-500
        local complexity=$((RANDOM % 491 + 10))

        if curl -s -X POST "${BASE_URL}/task/notify" \
            -H "Content-Type: application/json" \
            -d "{\"task\":\"${task}\",\"complexity\":${complexity}}" > /dev/null; then
            ((success++))
        else
            ((failed++))
        fi

        if (( i % 50 == 0 )); then
            echo "Progress: $i / $count (success: $success, failed: $failed)"
        fi

        # Small delay to let observation happen
        sleep 0.1
    done

    echo ""
    echo "Completed: $success successful, $failed failed"
    echo ""

    # Show stats
    echo "Model stats:"
    curl -s "${BASE_URL}/stats" | jq .
}

# Mixed workload simulation
benchmark_mixed() {
    local duration="${1:-60}"

    check_server

    echo "Running mixed workload for ${duration}s..."
    echo ""

    local end_time=$(($(date +%s) + duration))
    local ask_count=0
    local notify_count=0
    local status_count=0

    local tasks=("video_encoding" "ml_inference" "data_processing" "backup" "report_generation")

    while [ $(date +%s) -lt $end_time ]; do
        local op=$((RANDOM % 10))

        if [ $op -lt 6 ]; then
            # 60% /ask requests
            local task=${tasks[$((RANDOM % ${#tasks[@]}))]}
            local complexity=$((RANDOM % 491 + 10))
            curl -s -X POST "${BASE_URL}/ask" \
                -H "Content-Type: application/json" \
                -d "{\"task\":\"${task}\",\"complexity\":${complexity}}" > /dev/null &
            ((ask_count++))
        elif [ $op -lt 9 ]; then
            # 30% /task/notify
            local task=${tasks[$((RANDOM % ${#tasks[@]}))]}
            local complexity=$((RANDOM % 491 + 10))
            curl -s -X POST "${BASE_URL}/task/notify" \
                -H "Content-Type: application/json" \
                -d "{\"task\":\"${task}\",\"complexity\":${complexity}}" > /dev/null &
            ((notify_count++))
        else
            # 10% /status
            curl -s "${BASE_URL}/status" > /dev/null &
            ((status_count++))
        fi

        # Control request rate
        sleep 0.05
    done

    # Wait for background requests
    wait

    local total=$((ask_count + notify_count + status_count))

    echo ""
    echo "Mixed workload results:"
    echo "  Duration: ${duration}s"
    echo "  Total requests: $total"
    echo "  /ask: $ask_count"
    echo "  /task/notify: $notify_count"
    echo "  /status: $status_count"
    echo "  RPS: $(echo "scale=2; $total / $duration" | bc)"
}

# Train model with synthetic data
train_model() {
    local count="${1:-100}"

    check_server

    echo "Training model with $count synthetic observations..."
    echo ""

    local tasks=("video_encoding" "ml_inference" "data_processing")

    for i in $(seq 1 "$count"); do
        local task=${tasks[$((RANDOM % ${#tasks[@]}))]}
        local complexity=$((RANDOM % 491 + 10))

        curl -s -X POST "${BASE_URL}/task/notify" \
            -H "Content-Type: application/json" \
            -d "{\"task\":\"${task}\",\"complexity\":${complexity}}" > /dev/null

        if (( i % 20 == 0 )); then
            echo "Progress: $i / $count"
        fi

        sleep 0.2
    done

    echo ""
    echo "Training complete. Model stats:"
    curl -s "${BASE_URL}/stats" | jq .
}

# Compare model predictions
compare_models() {
    check_server

    echo "Comparing model predictions..."
    echo ""

    local output="${OUTPUT_DIR}/model_comparison_$(timestamp).csv"
    echo "task,complexity,predicted_cpu,predicted_mem,confidence" > "$output"

    local tasks=("video_encoding" "ml_inference" "data_processing")

    for task in "${tasks[@]}"; do
        for complexity in 50 100 150 200 250 300 400 500; do
            local response=$(curl -s -X POST "${BASE_URL}/v2/ask" \
                -H "Content-Type: application/json" \
                -d "{\"task\":\"${task}\",\"complexity\":${complexity}}")

            local cpu=$(echo "$response" | jq -r '.predicted.cpu // 0')
            local mem=$(echo "$response" | jq -r '.predicted.memory // 0')
            local conf=$(echo "$response" | jq -r '.confidence // 0')

            echo "${task},${complexity},${cpu},${mem},${conf}" >> "$output"
            echo "  ${task} @ ${complexity}: CPU=${cpu}%, Mem=${mem}%, Confidence=${conf}"
        done
    done

    echo ""
    echo "Results saved to: $output"
}

# Generate benchmark report
generate_report() {
    local output="${OUTPUT_DIR}/report_$(timestamp).txt"

    check_server

    {
        echo "======================================"
        echo "Capfox Benchmark Report"
        echo "Generated: $(date)"
        echo "Server: ${BASE_URL}"
        echo "======================================"
        echo ""

        echo "=== Server Info ==="
        curl -s "${BASE_URL}/" | jq .
        echo ""

        echo "=== Current Status ==="
        curl -s "${BASE_URL}/status" | jq .
        echo ""

        echo "=== Model Stats ==="
        curl -s "${BASE_URL}/stats" | jq .
        echo ""

        if curl -s "${BASE_URL}/v2/model/stats" > /dev/null 2>&1; then
            echo "=== V2 Model Stats ==="
            curl -s "${BASE_URL}/v2/model/stats" | jq .
            echo ""

            echo "=== V2 Scheduler Stats ==="
            curl -s "${BASE_URL}/v2/scheduler/stats" | jq .
            echo ""
        fi

    } | tee "$output"

    echo ""
    echo "Report saved to: $output"
}

# Main
case "${1:-}" in
    ask)
        benchmark_ask "$2" "$3"
        ;;
    notify)
        shift
        benchmark_notify "$@"
        ;;
    mixed)
        benchmark_mixed "$2"
        ;;
    train)
        train_model "$2"
        ;;
    compare-models)
        compare_models
        ;;
    report)
        generate_report
        ;;
    *)
        usage
        ;;
esac

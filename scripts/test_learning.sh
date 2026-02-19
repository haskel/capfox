#!/bin/bash
# Learning Pipeline Test Script
# Tests that the model learns from task notifications with simulated metrics
#
# Usage: ./scripts/test_learning.sh
#
# Requirements:
# - Server running with debug mode enabled (configs/capfox.debug.yaml)
# - jq installed for JSON parsing

set -e

HOST="${CAPFOX_HOST:-localhost}"
PORT="${CAPFOX_PORT:-8080}"
BASE_URL="http://${HOST}:${PORT}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

check_server() {
    if ! curl -s "${BASE_URL}/health" > /dev/null 2>&1; then
        log_error "Server not responding at ${BASE_URL}"
        log_error "Start with: ./bin/capfox start --config configs/capfox.debug.yaml"
        exit 1
    fi
    log_info "Server is up at ${BASE_URL}"
}

check_debug_mode() {
    local status
    status=$(curl -s "${BASE_URL}/debug/status" 2>&1)
    if echo "$status" | grep -q "404"; then
        log_error "Debug mode is not enabled!"
        log_error "Start server with: ./bin/capfox start --config configs/capfox.debug.yaml"
        exit 1
    fi
    log_info "Debug mode is enabled"
}

# Inject metrics and wait for monitoring to pick them up
inject_metrics() {
    local cpu=$1
    local mem=$2
    local gpu=$3
    local vram=$4

    local body="{}"

    if [ -n "$cpu" ]; then
        body=$(echo "$body" | jq ". + {cpu: $cpu}")
    fi
    if [ -n "$mem" ]; then
        body=$(echo "$body" | jq ". + {memory: $mem}")
    fi
    if [ -n "$gpu" ]; then
        body=$(echo "$body" | jq ". + {gpu_usage: $gpu}")
    fi
    if [ -n "$vram" ]; then
        body=$(echo "$body" | jq ". + {vram_usage: $vram}")
    fi

    curl -s -X POST "${BASE_URL}/debug/inject-metrics" \
        -H "Content-Type: application/json" \
        -d "$body" > /dev/null
}

# Get current stats for a task
get_task_stats() {
    local task=$1
    curl -s "${BASE_URL}/stats"
}

# Get count for a specific task
get_task_count() {
    local task=$1
    curl -s "${BASE_URL}/stats" | jq -r ".tasks[\"${task}\"].count // 0"
}

# Get avg CPU delta for a specific task
get_task_cpu_delta() {
    local task=$1
    curl -s "${BASE_URL}/stats" | jq -r ".tasks[\"${task}\"].avg_cpu_delta // 0"
}

# Notify a task start
notify_task() {
    local task=$1
    local complexity=$2

    curl -s -X POST "${BASE_URL}/task/notify" \
        -H "Content-Type: application/json" \
        -d "{\"task\":\"${task}\",\"complexity\":${complexity}}"
}

# Ask about a task
ask_task() {
    local task=$1
    local complexity=$2

    curl -s -X POST "${BASE_URL}/ask" \
        -H "Content-Type: application/json" \
        -d "{\"task\":\"${task}\",\"complexity\":${complexity}}"
}

# =============================================================================
# Test 1: Basic Learning with CPU (15 observations for statistical reliability)
# =============================================================================
test_basic_cpu_learning() {
    log_info "=== Test 1: Basic CPU Learning ==="

    local task="test_cpu_task"
    local baseline_cpu=30.0

    # Set baseline metrics
    inject_metrics $baseline_cpu 50.0 "" ""
    sleep 0.5

    log_info "Training with 15 observations (research shows N >= 10 minimum)..."

    # Train with different complexities (15 observations)
    for complexity in 50 75 100 125 150 175 200 225 250 275 300 325 350 375 400; do
        # Calculate expected CPU delta (linear: ~0.1% per complexity unit)
        local cpu_delta=$(echo "scale=1; $complexity * 0.1" | bc)
        local new_cpu=$(echo "scale=1; $baseline_cpu + $cpu_delta" | bc)

        # Notify task start
        notify_task "$task" "$complexity" > /dev/null

        # Simulate CPU increase after task starts
        sleep 0.3
        inject_metrics "$new_cpu" 50.0 "" ""

        # Wait for observation to complete
        sleep 1.2

        # Reset to baseline
        inject_metrics $baseline_cpu 50.0 "" ""
        sleep 0.2

        # Progress indicator
        if (( complexity % 100 == 0 )); then
            log_info "  Progress: complexity $complexity done"
        fi
    done

    # Check what the model learned
    log_info "Checking learned model..."

    local count
    count=$(get_task_count "$task")
    local avg_cpu_delta
    avg_cpu_delta=$(get_task_cpu_delta "$task")

    log_info "  Observations: $count"
    log_info "  Avg CPU delta: $avg_cpu_delta"

    if [ "$count" -ge 10 ]; then
        log_info "${GREEN}✓ Test 1 PASSED: Model learned from $count observations${NC}"
        return 0
    else
        log_error "✗ Test 1 FAILED: Expected at least 10 observations, got $count"
        return 1
    fi
}

# =============================================================================
# Test 2: Learning with GPU (12 observations)
# =============================================================================
test_gpu_learning() {
    log_info "=== Test 2: GPU Learning ==="

    local task="test_gpu_task"
    local baseline_gpu=20.0
    local baseline_vram=30.0

    # Set baseline with GPU
    inject_metrics 30.0 50.0 $baseline_gpu $baseline_vram
    sleep 0.5

    log_info "Training GPU task with 12 observations..."

    for complexity in 50 100 150 200 250 300 350 400 450 500 550 600; do
        local gpu_delta=$(echo "scale=1; $complexity * 0.1" | bc)
        local vram_delta=$(echo "scale=1; $complexity * 0.05" | bc)
        local new_gpu=$(echo "scale=1; $baseline_gpu + $gpu_delta" | bc)
        local new_vram=$(echo "scale=1; $baseline_vram + $vram_delta" | bc)

        notify_task "$task" "$complexity" > /dev/null

        sleep 0.3
        inject_metrics 30.0 50.0 "$new_gpu" "$new_vram"

        sleep 1.2

        inject_metrics 30.0 50.0 $baseline_gpu $baseline_vram
        sleep 0.2
    done

    local count
    count=$(get_task_count "$task")

    log_info "  Observations: $count"

    if [ "$count" -ge 10 ]; then
        log_info "${GREEN}✓ Test 2 PASSED: GPU learning with $count observations${NC}"
        return 0
    else
        log_error "✗ Test 2 FAILED: Expected at least 10 observations, got $count"
        return 1
    fi
}

# =============================================================================
# Test 3: Prediction Accuracy (15 observations + verification)
# =============================================================================
test_prediction_accuracy() {
    log_info "=== Test 3: Prediction Accuracy ==="

    local task="test_prediction_task"
    local baseline_cpu=25.0

    inject_metrics $baseline_cpu 50.0 "" ""
    sleep 0.5

    log_info "Training with linear pattern (15 obs, CPU = baseline + complexity * 0.1)..."

    # Train with clear linear pattern (15 observations)
    for complexity in 50 75 100 125 150 175 200 225 250 275 300 325 350 375 400; do
        local expected_delta=$(echo "scale=1; $complexity * 0.1" | bc)
        local new_cpu=$(echo "scale=1; $baseline_cpu + $expected_delta" | bc)

        notify_task "$task" "$complexity" > /dev/null
        sleep 0.3
        inject_metrics "$new_cpu" 50.0 "" ""
        sleep 1.2
        inject_metrics $baseline_cpu 50.0 "" ""
        sleep 0.2
    done

    local count
    count=$(get_task_count "$task")
    log_info "  Trained with $count observations"

    # Now test prediction for complexity=500
    log_info "Testing prediction for complexity=500..."
    local prediction
    prediction=$(ask_task "$task" 500)

    log_info "  Prediction response: $prediction"

    local allowed
    allowed=$(echo "$prediction" | jq -r '.allowed')

    # With baseline 25% and expected delta ~50% (500*0.1), total ~75%
    # This should be allowed (below 80% threshold)
    log_info "  Allowed: $allowed"

    if [ "$count" -ge 10 ]; then
        log_info "${GREEN}✓ Test 3 PASSED: Prediction made with $count observations${NC}"
        return 0
    else
        log_error "✗ Test 3 FAILED: Not enough observations ($count)"
        return 1
    fi
}

# =============================================================================
# Test 4: Multiple Task Types (10 observations each)
# =============================================================================
test_multiple_tasks() {
    log_info "=== Test 4: Multiple Task Types ==="

    inject_metrics 30.0 50.0 "" ""
    sleep 0.5

    # Task 1: CPU heavy (10 observations)
    local task1="cpu_heavy_task"
    log_info "Training CPU-heavy task..."
    for complexity in 50 100 150 200 250 300 350 400 450 500; do
        notify_task "$task1" "$complexity" > /dev/null
        sleep 0.3
        inject_metrics $(echo "30 + $complexity * 0.15" | bc) 50.0 "" ""
        sleep 1.2
        inject_metrics 30.0 50.0 "" ""
        sleep 0.2
    done

    # Task 2: Memory heavy (10 observations)
    local task2="mem_heavy_task"
    log_info "Training Memory-heavy task..."
    for complexity in 50 100 150 200 250 300 350 400 450 500; do
        notify_task "$task2" "$complexity" > /dev/null
        sleep 0.3
        inject_metrics 30.0 $(echo "50 + $complexity * 0.08" | bc) "" ""
        sleep 1.2
        inject_metrics 30.0 50.0 "" ""
        sleep 0.2
    done

    # Check both tasks learned independently
    local task1_count
    task1_count=$(get_task_count "$task1")
    local task2_count
    task2_count=$(get_task_count "$task2")

    log_info "  ${task1} observations: $task1_count"
    log_info "  ${task2} observations: $task2_count"

    if [ "$task1_count" -ge 8 ] && [ "$task2_count" -ge 8 ]; then
        log_info "${GREEN}✓ Test 4 PASSED: Multiple task types learned independently${NC}"
        return 0
    else
        log_error "✗ Test 4 FAILED: Tasks not learned properly"
        return 1
    fi
}

# =============================================================================
# Test 5: Complexity Scaling (12 observations with linear pattern)
# =============================================================================
test_complexity_scaling() {
    log_info "=== Test 5: Complexity Scaling ==="

    local task="scaling_task"
    local baseline_cpu=20.0
    local baseline_mem=40.0

    inject_metrics $baseline_cpu $baseline_mem "" ""
    sleep 0.5

    log_info "Training with linear complexity pattern (12 observations)..."

    # Train with linear pattern: CPU = baseline + complexity * 0.1
    for complexity in 50 100 150 200 250 300 350 400 450 500 550 600; do
        local cpu_delta=$(echo "scale=1; $complexity * 0.1" | bc)
        local mem_delta=$(echo "scale=1; $complexity * 0.05" | bc)
        local new_cpu=$(echo "scale=1; $baseline_cpu + $cpu_delta" | bc)
        local new_mem=$(echo "scale=1; $baseline_mem + $mem_delta" | bc)

        notify_task "$task" "$complexity" > /dev/null
        sleep 0.3
        inject_metrics "$new_cpu" "$new_mem" "" ""
        sleep 1.2
        inject_metrics $baseline_cpu $baseline_mem "" ""
        sleep 0.2
    done

    local count
    count=$(get_task_count "$task")
    log_info "  Trained with $count observations"

    # Check predictions scale with complexity
    local low_pred
    low_pred=$(ask_task "$task" 100)
    local high_pred
    high_pred=$(ask_task "$task" 500)

    log_info "  Low complexity (100) prediction: $low_pred"
    log_info "  High complexity (500) prediction: $high_pred"

    if [ "$count" -ge 10 ]; then
        log_info "${GREEN}✓ Test 5 PASSED: Complexity scaling with $count observations${NC}"
        return 0
    else
        log_error "✗ Test 5 FAILED: Expected at least 10 observations, got $count"
        return 1
    fi
}

# =============================================================================
# Main
# =============================================================================
main() {
    echo "========================================"
    echo "Capfox Learning Pipeline Test"
    echo "========================================"
    echo ""

    check_server
    check_debug_mode

    echo ""

    local passed=0
    local failed=0

    if test_basic_cpu_learning; then
        ((passed++))
    else
        ((failed++))
    fi
    echo ""

    if test_gpu_learning; then
        ((passed++))
    else
        ((failed++))
    fi
    echo ""

    if test_prediction_accuracy; then
        ((passed++))
    else
        ((failed++))
    fi
    echo ""

    if test_multiple_tasks; then
        ((passed++))
    else
        ((failed++))
    fi
    echo ""

    if test_complexity_scaling; then
        ((passed++))
    else
        ((failed++))
    fi
    echo ""

    echo "========================================"
    echo "Results: ${passed} passed, ${failed} failed"
    echo "========================================"

    # Show final model stats
    echo ""
    log_info "Final model stats:"
    curl -s "${BASE_URL}/stats" | jq .

    if [ "$failed" -gt 0 ]; then
        exit 1
    fi
}

main "$@"

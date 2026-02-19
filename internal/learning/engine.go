package learning

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/haskel/capfox/internal/monitor"
)

const (
	// DefaultMaxWorkers is the default number of concurrent observation workers.
	DefaultMaxWorkers = 100
)

// Engine coordinates learning from task executions.
type Engine struct {
	model      Model
	aggregator *monitor.Aggregator
	logger     *slog.Logger

	observationDelay time.Duration
	maxWorkers       int

	mu             sync.Mutex
	pendingTasks   map[string]*pendingTask
	taskCounter    int64

	// Goroutine management
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	workerSem  chan struct{} // semaphore for limiting concurrent workers
	stopped    bool
}

// pendingTask holds state for a task awaiting observation.
type pendingTask struct {
	id         string
	task       string
	complexity int
	startedAt  time.Time
	baseline   *monitor.SystemState
}

// NewEngine creates a new learning engine.
func NewEngine(model Model, aggregator *monitor.Aggregator, observationDelay time.Duration, logger *slog.Logger) *Engine {
	return NewEngineWithWorkers(model, aggregator, observationDelay, logger, DefaultMaxWorkers)
}

// NewEngineWithWorkers creates a new learning engine with custom worker limit.
func NewEngineWithWorkers(model Model, aggregator *monitor.Aggregator, observationDelay time.Duration, logger *slog.Logger, maxWorkers int) *Engine {
	if maxWorkers <= 0 {
		maxWorkers = DefaultMaxWorkers
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &Engine{
		model:            model,
		aggregator:       aggregator,
		logger:           logger,
		observationDelay: observationDelay,
		maxWorkers:       maxWorkers,
		pendingTasks:     make(map[string]*pendingTask),
		ctx:              ctx,
		cancel:           cancel,
		workerSem:        make(chan struct{}, maxWorkers),
	}
}

// NotifyTaskStart records that a task has started.
// It captures a baseline of system state and schedules an observation.
func (e *Engine) NotifyTaskStart(task string, complexity int) {
	e.mu.Lock()
	if e.stopped {
		e.mu.Unlock()
		return
	}
	e.taskCounter++
	taskID := task + "_" + time.Now().Format("20060102150405") + "_" + formatCounter(e.taskCounter)

	baseline := e.aggregator.GetState()

	pt := &pendingTask{
		id:         taskID,
		task:       task,
		complexity: complexity,
		startedAt:  time.Now(),
		baseline:   baseline,
	}
	e.pendingTasks[taskID] = pt
	e.mu.Unlock()

	e.logger.Debug("task started, scheduling observation",
		"task", task,
		"task_id", taskID,
		"delay", e.observationDelay,
	)

	// Schedule observation after delay with bounded concurrency
	e.wg.Add(1)
	go func() {
		defer e.wg.Done()

		// Acquire semaphore slot (bounded concurrency)
		select {
		case e.workerSem <- struct{}{}:
			// Got slot
		case <-e.ctx.Done():
			return
		}
		defer func() { <-e.workerSem }()

		// Wait for observation delay or cancellation
		select {
		case <-time.After(e.observationDelay):
			e.observe(taskID)
		case <-e.ctx.Done():
			return
		}
	}()
}

// formatCounter formats a counter value as a zero-padded string.
func formatCounter(n int64) string {
	return fmt.Sprintf("%03d", n%1000)
}

// observe captures the impact of a task after the observation delay.
func (e *Engine) observe(taskID string) {
	e.mu.Lock()
	pt, exists := e.pendingTasks[taskID]
	if !exists {
		e.mu.Unlock()
		return
	}
	delete(e.pendingTasks, taskID)
	e.mu.Unlock()

	// Get current state
	current := e.aggregator.GetState()
	if current == nil || pt.baseline == nil {
		e.logger.Warn("observation skipped: missing state",
			"task_id", taskID,
		)
		return
	}

	// Calculate impact (delta between baseline and current)
	impact := &ResourceImpact{
		CPUDelta:    current.CPU.UsagePercent - pt.baseline.CPU.UsagePercent,
		MemoryDelta: current.Memory.UsagePercent - pt.baseline.Memory.UsagePercent,
	}

	// GPU delta (average across all GPUs)
	if len(current.GPUs) > 0 && len(pt.baseline.GPUs) > 0 {
		var gpuDelta, vramDelta float64
		count := min(len(current.GPUs), len(pt.baseline.GPUs))
		for i := 0; i < count; i++ {
			gpuDelta += current.GPUs[i].UsagePercent - pt.baseline.GPUs[i].UsagePercent

			// VRAM delta (as percentage)
			if current.GPUs[i].VRAMTotalBytes > 0 && pt.baseline.GPUs[i].VRAMTotalBytes > 0 {
				currentVRAMPct := float64(current.GPUs[i].VRAMUsedBytes) / float64(current.GPUs[i].VRAMTotalBytes) * 100
				baselineVRAMPct := float64(pt.baseline.GPUs[i].VRAMUsedBytes) / float64(pt.baseline.GPUs[i].VRAMTotalBytes) * 100
				vramDelta += currentVRAMPct - baselineVRAMPct
			}
		}
		impact.GPUDelta = gpuDelta / float64(count)
		impact.VRAMDelta = vramDelta / float64(count)
	}

	e.logger.Debug("task impact observed",
		"task", pt.task,
		"task_id", taskID,
		"cpu_delta", impact.CPUDelta,
		"mem_delta", impact.MemoryDelta,
		"gpu_delta", impact.GPUDelta,
	)

	// Feed observation to the model
	e.model.Observe(pt.task, pt.complexity, impact)
}

// Predict returns predicted resource impact for a task.
func (e *Engine) Predict(task string, complexity int) *ResourceImpact {
	return e.model.Predict(task, complexity)
}

// GetStats returns statistics for all observed tasks.
func (e *Engine) GetStats() *AllStats {
	return e.model.GetStats()
}

// GetTaskStats returns statistics for a specific task.
func (e *Engine) GetTaskStats(task string) *TaskStats {
	return e.model.GetTaskStats(task)
}

// Model returns the underlying model.
func (e *Engine) Model() Model {
	return e.model
}

// PendingCount returns the number of tasks awaiting observation.
func (e *Engine) PendingCount() int {
	e.mu.Lock()
	defer e.mu.Unlock()
	return len(e.pendingTasks)
}

// Stop gracefully shuts down the engine, waiting for pending observations.
func (e *Engine) Stop() {
	e.mu.Lock()
	if e.stopped {
		e.mu.Unlock()
		return
	}
	e.stopped = true
	e.mu.Unlock()

	// Cancel all pending goroutines
	e.cancel()

	// Wait for all goroutines to finish
	e.wg.Wait()

	e.logger.Debug("learning engine stopped")
}

// StopWithTimeout stops the engine with a timeout.
func (e *Engine) StopWithTimeout(timeout time.Duration) {
	e.mu.Lock()
	if e.stopped {
		e.mu.Unlock()
		return
	}
	e.stopped = true
	e.mu.Unlock()

	// Cancel all pending goroutines
	e.cancel()

	// Wait with timeout
	done := make(chan struct{})
	go func() {
		e.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		e.logger.Debug("learning engine stopped gracefully")
	case <-time.After(timeout):
		e.logger.Warn("learning engine stop timed out", "timeout", timeout)
	}
}

// ActiveWorkers returns the current number of active observation workers.
func (e *Engine) ActiveWorkers() int {
	return len(e.workerSem)
}

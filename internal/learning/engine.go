package learning

import (
	"log/slog"
	"sync"
	"time"

	"github.com/haskel/capfox/internal/monitor"
)

// Engine coordinates learning from task executions.
type Engine struct {
	model      Model
	aggregator *monitor.Aggregator
	logger     *slog.Logger

	observationDelay time.Duration

	mu             sync.Mutex
	pendingTasks   map[string]*pendingTask
	taskCounter    int64
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
	return &Engine{
		model:            model,
		aggregator:       aggregator,
		logger:           logger,
		observationDelay: observationDelay,
		pendingTasks:     make(map[string]*pendingTask),
	}
}

// NotifyTaskStart records that a task has started.
// It captures a baseline of system state and schedules an observation.
func (e *Engine) NotifyTaskStart(task string, complexity int) {
	e.mu.Lock()
	e.taskCounter++
	taskID := task + "_" + time.Now().Format("20060102150405") + "_" + string(rune(e.taskCounter%1000))

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

	// Schedule observation after delay
	go func() {
		time.Sleep(e.observationDelay)
		e.observe(taskID)
	}()
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

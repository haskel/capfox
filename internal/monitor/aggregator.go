package monitor

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

type Aggregator struct {
	monitors []Monitor
	state    *SystemState
	interval time.Duration
	mu       sync.RWMutex
	done     chan struct{}
	stopOnce sync.Once
	logger   *slog.Logger

	ready     bool      // true after first successful collection
	readyTime time.Time // when first collection completed
}

func NewAggregator(monitors []Monitor, interval time.Duration, logger *slog.Logger) *Aggregator {
	return &Aggregator{
		monitors: monitors,
		state:    &SystemState{},
		interval: interval,
		done:     make(chan struct{}),
		logger:   logger,
	}
}

func (a *Aggregator) Start(ctx context.Context) error {
	// Initial collection
	a.collect()

	go a.runLoop(ctx)

	a.logger.Info("aggregator started", "interval", a.interval, "monitors", len(a.monitors))
	return nil
}

func (a *Aggregator) Stop() error {
	a.stopOnce.Do(func() {
		close(a.done)

		// Close monitors that implement Closer interface
		for _, m := range a.monitors {
			if closer, ok := m.(Closer); ok {
				if err := closer.Close(); err != nil {
					a.logger.Warn("failed to close monitor",
						"monitor", m.Name(),
						"error", err,
					)
				}
			}
		}

		a.logger.Info("aggregator stopped")
	})
	return nil
}

func (a *Aggregator) GetState() *SystemState {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.state.Clone()
}

func (a *Aggregator) GetStateJSON() ([]byte, error) {
	state := a.GetState()
	return json.Marshal(state)
}

// IsReady returns true if the aggregator has collected initial metrics.
func (a *Aggregator) IsReady() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.ready
}

// ReadyTime returns when the aggregator became ready.
// Returns zero time if not ready yet.
func (a *Aggregator) ReadyTime() time.Time {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.readyTime
}

// InjectedMetrics represents metrics to inject for testing/debugging.
type InjectedMetrics struct {
	CPU         *float64 `json:"cpu,omitempty"`          // CPU usage percent
	Memory      *float64 `json:"memory,omitempty"`       // Memory usage percent
	GPUUsage    *float64 `json:"gpu_usage,omitempty"`    // GPU usage percent (first GPU)
	VRAMUsage   *float64 `json:"vram_usage,omitempty"`   // VRAM usage percent (first GPU)
	GPUIndex    int      `json:"gpu_index,omitempty"`    // Which GPU to modify (default 0)
}

// InjectMetrics overrides current metrics with injected values.
// Only non-nil fields are applied. Used for testing/debugging.
// Returns an error if gpu_index is negative.
func (a *Aggregator) InjectMetrics(metrics *InjectedMetrics) error {
	if metrics.GPUIndex < 0 {
		return fmt.Errorf("gpu_index must be >= 0, got %d", metrics.GPUIndex)
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	if metrics.CPU != nil {
		a.state.CPU.UsagePercent = *metrics.CPU
		// Update all cores proportionally
		for i := range a.state.CPU.Cores {
			a.state.CPU.Cores[i] = *metrics.CPU
		}
	}

	if metrics.Memory != nil {
		a.state.Memory.UsagePercent = *metrics.Memory
		// Calculate used bytes based on percentage
		a.state.Memory.UsedBytes = uint64(float64(a.state.Memory.TotalBytes) * (*metrics.Memory / 100))
	}

	// GPU metrics
	gpuIdx := metrics.GPUIndex
	if metrics.GPUUsage != nil || metrics.VRAMUsage != nil {
		// Ensure GPU exists
		if gpuIdx >= len(a.state.GPUs) {
			// Create a synthetic GPU for testing
			for len(a.state.GPUs) <= gpuIdx {
				a.state.GPUs = append(a.state.GPUs, GPUState{
					Index:          len(a.state.GPUs),
					Name:           "Debug GPU",
					VRAMTotalBytes: 24 * 1024 * 1024 * 1024, // 24GB default
				})
			}
		}

		if metrics.GPUUsage != nil {
			a.state.GPUs[gpuIdx].UsagePercent = *metrics.GPUUsage
		}
		if metrics.VRAMUsage != nil {
			a.state.GPUs[gpuIdx].VRAMUsedBytes = uint64(
				float64(a.state.GPUs[gpuIdx].VRAMTotalBytes) * (*metrics.VRAMUsage / 100),
			)
		}
	}

	a.state.Timestamp = time.Now()
	a.logger.Debug("metrics injected",
		"cpu", metrics.CPU,
		"memory", metrics.Memory,
		"gpu_usage", metrics.GPUUsage,
		"vram_usage", metrics.VRAMUsage,
	)

	return nil
}

func (a *Aggregator) runLoop(ctx context.Context) {
	ticker := time.NewTicker(a.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			a.collect()
		case <-ctx.Done():
			return
		case <-a.done:
			return
		}
	}
}

func (a *Aggregator) collect() {
	newState := &SystemState{
		Timestamp: time.Now(),
		GPUs:      []GPUState{},
		Storage:   make(StorageState),
	}

	for _, m := range a.monitors {
		data, err := m.Collect()
		if err != nil {
			a.logger.Warn("monitor collection failed",
				"monitor", m.Name(),
				"error", err,
			)
			continue
		}

		switch m.Name() {
		case "cpu":
			if cpuState, ok := data.(*CPUState); ok {
				newState.CPU = *cpuState
			}
		case "memory":
			if memState, ok := data.(*MemoryState); ok {
				newState.Memory = *memState
			}
		case "storage":
			if storageState, ok := data.(StorageState); ok {
				newState.Storage = storageState
			}
		case "process":
			if procState, ok := data.(*ProcessState); ok {
				newState.Processes = procState.Processes
				newState.Threads = procState.Threads
				newState.ContextSwitchesPerSec = procState.ContextSwitchesPerSec
			}
		case "gpu":
			if gpuStates, ok := data.([]GPUState); ok {
				newState.GPUs = gpuStates
			}
		}
	}

	a.mu.Lock()
	a.state = newState
	if !a.ready {
		a.ready = true
		a.readyTime = time.Now()
	}
	a.mu.Unlock()
}

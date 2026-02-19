package monitor

import (
	"context"
	"encoding/json"
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
	logger   *slog.Logger
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
	close(a.done)
	a.logger.Info("aggregator stopped")
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
	a.mu.Unlock()
}

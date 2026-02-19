package learning

import "sync"

// MovingAverageModel implements Model using exponential moving average.
type MovingAverageModel struct {
	mu       sync.RWMutex
	stats    map[string]*taskState
	alpha    float64 // smoothing factor (0 < alpha <= 1)
	observer StatsObserver
}

// taskState holds internal state for a task type.
type taskState struct {
	count        int64
	avgCPUDelta  float64
	avgMemDelta  float64
	avgGPUDelta  float64
	avgVRAMDelta float64
}

// NewMovingAverageModel creates a new MovingAverageModel.
// Alpha is the smoothing factor: higher values give more weight to recent observations.
// Typical values are 0.1-0.3.
func NewMovingAverageModel(alpha float64) *MovingAverageModel {
	if alpha <= 0 || alpha > 1 {
		alpha = 0.2 // default
	}
	return &MovingAverageModel{
		stats: make(map[string]*taskState),
		alpha: alpha,
	}
}

func (m *MovingAverageModel) Name() string {
	return "moving_average"
}

func (m *MovingAverageModel) Observe(task string, complexity int, impact *ResourceImpact) {
	if impact == nil {
		return
	}

	m.mu.Lock()

	state, exists := m.stats[task]
	if !exists {
		// First observation - use values directly
		m.stats[task] = &taskState{
			count:        1,
			avgCPUDelta:  impact.CPUDelta,
			avgMemDelta:  impact.MemoryDelta,
			avgGPUDelta:  impact.GPUDelta,
			avgVRAMDelta: impact.VRAMDelta,
		}
		state = m.stats[task]
	} else {
		// Exponential moving average: new_avg = alpha * value + (1 - alpha) * old_avg
		state.count++
		state.avgCPUDelta = m.alpha*impact.CPUDelta + (1-m.alpha)*state.avgCPUDelta
		state.avgMemDelta = m.alpha*impact.MemoryDelta + (1-m.alpha)*state.avgMemDelta
		state.avgGPUDelta = m.alpha*impact.GPUDelta + (1-m.alpha)*state.avgGPUDelta
		state.avgVRAMDelta = m.alpha*impact.VRAMDelta + (1-m.alpha)*state.avgVRAMDelta
	}

	// Get stats before unlocking
	stats := &TaskStats{
		Task:         task,
		Count:        state.count,
		AvgCPUDelta:  state.avgCPUDelta,
		AvgMemDelta:  state.avgMemDelta,
		AvgGPUDelta:  state.avgGPUDelta,
		AvgVRAMDelta: state.avgVRAMDelta,
	}
	observer := m.observer

	m.mu.Unlock()

	// Notify observer outside of lock
	if observer != nil {
		observer(task, stats)
	}
}

func (m *MovingAverageModel) Predict(task string, complexity int) *ResourceImpact {
	m.mu.RLock()
	defer m.mu.RUnlock()

	state, exists := m.stats[task]
	if !exists {
		// No data for this task - return nil (unknown impact)
		return nil
	}

	// For now, complexity is not used in prediction
	// Future: could scale impact by complexity ratio
	return &ResourceImpact{
		CPUDelta:    state.avgCPUDelta,
		MemoryDelta: state.avgMemDelta,
		GPUDelta:    state.avgGPUDelta,
		VRAMDelta:   state.avgVRAMDelta,
	}
}

func (m *MovingAverageModel) GetStats() *AllStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := &AllStats{
		Tasks:      make(map[string]*TaskStats),
		TotalTasks: 0,
	}

	for task, state := range m.stats {
		result.Tasks[task] = &TaskStats{
			Task:         task,
			Count:        state.count,
			AvgCPUDelta:  state.avgCPUDelta,
			AvgMemDelta:  state.avgMemDelta,
			AvgGPUDelta:  state.avgGPUDelta,
			AvgVRAMDelta: state.avgVRAMDelta,
		}
		result.TotalTasks += state.count
	}

	return result
}

func (m *MovingAverageModel) GetTaskStats(task string) *TaskStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	state, exists := m.stats[task]
	if !exists {
		return nil
	}

	return &TaskStats{
		Task:         task,
		Count:        state.count,
		AvgCPUDelta:  state.avgCPUDelta,
		AvgMemDelta:  state.avgMemDelta,
		AvgGPUDelta:  state.avgGPUDelta,
		AvgVRAMDelta: state.avgVRAMDelta,
	}
}

func (m *MovingAverageModel) SetObserver(observer StatsObserver) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.observer = observer
}

func (m *MovingAverageModel) LoadStats(stats *AllStats) {
	if stats == nil {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	for task, ts := range stats.Tasks {
		m.stats[task] = &taskState{
			count:        ts.Count,
			avgCPUDelta:  ts.AvgCPUDelta,
			avgMemDelta:  ts.AvgMemDelta,
			avgGPUDelta:  ts.AvgGPUDelta,
			avgVRAMDelta: ts.AvgVRAMDelta,
		}
	}
}

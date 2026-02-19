package model

import (
	"encoding/json"
	"io"
	"sync"

	"github.com/haskel/capfox/internal/decision"
)

// MovingAverageModel uses exponential moving average for predictions.
// It doesn't account for complexity - just tracks average resource usage per task type.
type MovingAverageModel struct {
	alpha float64
	mu    sync.RWMutex

	// Per-task averages
	tasks map[string]*movingAverageTaskData
}

type movingAverageTaskData struct {
	Count   int64   `json:"count"`
	CPUAvg  float64 `json:"cpu_avg"`
	MemAvg  float64 `json:"mem_avg"`
	GPUAvg  float64 `json:"gpu_avg"`
	VRAMAvg float64 `json:"vram_avg"`
}

type movingAverageState struct {
	Alpha float64                           `json:"alpha"`
	Tasks map[string]*movingAverageTaskData `json:"tasks"`
}

// NewMovingAverageModel creates a new moving average model.
// Alpha is the smoothing factor (0 < alpha <= 1). Higher values give more weight to recent observations.
func NewMovingAverageModel(alpha float64) *MovingAverageModel {
	if alpha <= 0 || alpha > 1 {
		alpha = 0.2
	}
	return &MovingAverageModel{
		alpha: alpha,
		tasks: make(map[string]*movingAverageTaskData),
	}
}

// Name returns the model name.
func (m *MovingAverageModel) Name() string {
	return string(ModelTypeMovingAverage)
}

// LearningType returns online learning type.
func (m *MovingAverageModel) LearningType() LearningType {
	return LearningTypeOnline
}

// Predict returns the exponential moving average of resource usage.
// Complexity is ignored in this model.
func (m *MovingAverageModel) Predict(task string, complexity int) *decision.ResourceImpact {
	m.mu.RLock()
	defer m.mu.RUnlock()

	data, exists := m.tasks[task]
	if !exists || data.Count == 0 {
		return nil
	}

	return &decision.ResourceImpact{
		CPUDelta:    data.CPUAvg,
		MemoryDelta: data.MemAvg,
		GPUDelta:    data.GPUAvg,
		VRAMDelta:   data.VRAMAvg,
	}
}

// Observe records an observation and updates the moving average.
func (m *MovingAverageModel) Observe(task string, complexity int, impact *decision.ResourceImpact) {
	if impact == nil {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	data, exists := m.tasks[task]
	if !exists {
		// First observation - use the value directly
		m.tasks[task] = &movingAverageTaskData{
			Count:   1,
			CPUAvg:  impact.CPUDelta,
			MemAvg:  impact.MemoryDelta,
			GPUAvg:  impact.GPUDelta,
			VRAMAvg: impact.VRAMDelta,
		}
		return
	}

	// Update exponential moving average: new_avg = alpha * new_value + (1 - alpha) * old_avg
	data.Count++
	data.CPUAvg = m.alpha*impact.CPUDelta + (1-m.alpha)*data.CPUAvg
	data.MemAvg = m.alpha*impact.MemoryDelta + (1-m.alpha)*data.MemAvg
	data.GPUAvg = m.alpha*impact.GPUDelta + (1-m.alpha)*data.GPUAvg
	data.VRAMAvg = m.alpha*impact.VRAMDelta + (1-m.alpha)*data.VRAMAvg
}

// Confidence returns confidence based on observation count.
// Starts at 0 and increases with more observations, maxing at 1.0 after ~10 observations.
func (m *MovingAverageModel) Confidence(task string) float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	data, exists := m.tasks[task]
	if !exists || data.Count == 0 {
		return 0
	}

	// Confidence grows with observations, saturating at ~10 observations
	// Using: confidence = 1 - exp(-count/5)
	conf := 1.0 - fastExp(-float64(data.Count)/5.0)
	if conf > 1.0 {
		return 1.0
	}
	return conf
}

// Stats returns model statistics.
func (m *MovingAverageModel) Stats() *Stats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var totalObs int64
	taskStats := make(map[string]*TaskStats)

	for name, data := range m.tasks {
		totalObs += data.Count
		taskStats[name] = &TaskStats{
			Task:         name,
			Count:        data.Count,
			AvgCPUDelta:  data.CPUAvg,
			AvgMemDelta:  data.MemAvg,
			AvgGPUDelta:  data.GPUAvg,
			AvgVRAMDelta: data.VRAMAvg,
		}
	}

	return &Stats{
		ModelName:         m.Name(),
		LearningType:      m.LearningType().String(),
		TotalObservations: totalObs,
		Tasks:             taskStats,
	}
}

// TaskStats returns statistics for a specific task.
func (m *MovingAverageModel) TaskStats(task string) *TaskStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	data, exists := m.tasks[task]
	if !exists {
		return nil
	}

	return &TaskStats{
		Task:         task,
		Count:        data.Count,
		AvgCPUDelta:  data.CPUAvg,
		AvgMemDelta:  data.MemAvg,
		AvgGPUDelta:  data.GPUAvg,
		AvgVRAMDelta: data.VRAMAvg,
	}
}

// NeedsRetrain returns false (online learning doesn't need retraining).
func (m *MovingAverageModel) NeedsRetrain() bool {
	return false
}

// Retrain is a no-op for online learning models.
func (m *MovingAverageModel) Retrain() error {
	return nil
}

// Save serializes the model state to a writer.
func (m *MovingAverageModel) Save(w io.Writer) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	state := movingAverageState{
		Alpha: m.alpha,
		Tasks: m.tasks,
	}

	return json.NewEncoder(w).Encode(state)
}

// Load deserializes the model state from a reader.
func (m *MovingAverageModel) Load(r io.Reader) error {
	var state movingAverageState
	if err := json.NewDecoder(r).Decode(&state); err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.alpha = state.Alpha
	m.tasks = state.Tasks
	if m.tasks == nil {
		m.tasks = make(map[string]*movingAverageTaskData)
	}

	return nil
}

// fastExp is a fast approximation of exp(x) for negative x values.
func fastExp(x float64) float64 {
	// For small negative x, use 1 + x + x²/2 approximation
	if x > -0.1 {
		return 1 + x + x*x/2
	}
	// Otherwise use simple approach: exp(x) ≈ (1 + x/n)^n for n=10
	t := 1 + x/10
	return t * t * t * t * t * t * t * t * t * t
}

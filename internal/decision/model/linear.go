package model

import (
	"encoding/json"
	"io"
	"math"
	"sync"

	"github.com/haskel/capfox/internal/decision"
)

// LinearModel uses online linear regression to predict resource usage.
// Prediction: impact = A * complexity + B
// Uses incremental least squares for efficient online updates.
type LinearModel struct {
	minObservations int
	mu              sync.RWMutex

	// Per-task regression data
	tasks map[string]*linearTaskData
}

// linearTaskData holds running statistics for incremental linear regression.
// Uses Welford's algorithm for online variance and covariance calculation.
type linearTaskData struct {
	Count int64 `json:"count"`

	// Running means
	MeanX float64 `json:"mean_x"` // mean of complexity

	// Running sums for linear regression
	// CPU
	CPUMeanY float64 `json:"cpu_mean_y"`
	CPUCov   float64 `json:"cpu_cov"` // sum of (x-mean_x)(y-mean_y)
	CPUVarX  float64 `json:"cpu_var"` // sum of (x-mean_x)²

	// Memory
	MemMeanY float64 `json:"mem_mean_y"`
	MemCov   float64 `json:"mem_cov"`

	// GPU
	GPUMeanY float64 `json:"gpu_mean_y"`
	GPUCov   float64 `json:"gpu_cov"`

	// VRAM
	VRAMMeanY float64 `json:"vram_mean_y"`
	VRAMCov   float64 `json:"vram_cov"`

	// Shared variance of X (complexity)
	VarX float64 `json:"var_x"`
}

type linearState struct {
	MinObservations int                        `json:"min_observations"`
	Tasks           map[string]*linearTaskData `json:"tasks"`
}

// NewLinearModel creates a new linear regression model.
func NewLinearModel(minObs int) *LinearModel {
	if minObs < 2 {
		minObs = 2
	}
	return &LinearModel{
		minObservations: minObs,
		tasks:           make(map[string]*linearTaskData),
	}
}

// Name returns the model name.
func (m *LinearModel) Name() string {
	return string(ModelTypeLinear)
}

// LearningType returns online learning type.
func (m *LinearModel) LearningType() LearningType {
	return LearningTypeOnline
}

// Predict returns predicted resource impact based on linear regression.
func (m *LinearModel) Predict(task string, complexity int) *decision.ResourceImpact {
	m.mu.RLock()
	defer m.mu.RUnlock()

	data, exists := m.tasks[task]
	if !exists || data.Count < int64(m.minObservations) {
		return nil
	}

	x := float64(complexity)
	coefs := m.calculateCoefficients(data)

	return &decision.ResourceImpact{
		CPUDelta:    coefs.CPUA*x + coefs.CPUB,
		MemoryDelta: coefs.MemA*x + coefs.MemB,
		GPUDelta:    coefs.GPUA*x + coefs.GPUB,
		VRAMDelta:   coefs.VRAMA*x + coefs.VRAMB,
	}
}

// calculateCoefficients calculates linear regression coefficients from running statistics.
func (m *LinearModel) calculateCoefficients(data *linearTaskData) *Coefficients {
	// Linear regression: y = ax + b
	// a = Cov(x,y) / Var(x)
	// b = mean_y - a * mean_x

	coefs := &Coefficients{}

	// Avoid division by zero
	if data.VarX < 1e-10 {
		// No variance in X, use mean as prediction (b = mean_y, a = 0)
		coefs.CPUB = data.CPUMeanY
		coefs.MemB = data.MemMeanY
		coefs.GPUB = data.GPUMeanY
		coefs.VRAMB = data.VRAMMeanY
		return coefs
	}

	// CPU
	coefs.CPUA = data.CPUCov / data.VarX
	coefs.CPUB = data.CPUMeanY - coefs.CPUA*data.MeanX

	// Memory
	coefs.MemA = data.MemCov / data.VarX
	coefs.MemB = data.MemMeanY - coefs.MemA*data.MeanX

	// GPU
	coefs.GPUA = data.GPUCov / data.VarX
	coefs.GPUB = data.GPUMeanY - coefs.GPUA*data.MeanX

	// VRAM
	coefs.VRAMA = data.VRAMCov / data.VarX
	coefs.VRAMB = data.VRAMMeanY - coefs.VRAMA*data.MeanX

	return coefs
}

// Observe records an observation and updates the model incrementally.
func (m *LinearModel) Observe(task string, complexity int, impact *decision.ResourceImpact) {
	if impact == nil {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	x := float64(complexity)

	data, exists := m.tasks[task]
	if !exists {
		// First observation
		m.tasks[task] = &linearTaskData{
			Count:     1,
			MeanX:     x,
			CPUMeanY:  impact.CPUDelta,
			MemMeanY:  impact.MemoryDelta,
			GPUMeanY:  impact.GPUDelta,
			VRAMMeanY: impact.VRAMDelta,
			// Covariances and variances start at 0
		}
		return
	}

	// Incremental update using Welford's method
	n := float64(data.Count + 1)

	// Old deviations
	deltaX := x - data.MeanX
	deltaCPU := impact.CPUDelta - data.CPUMeanY
	deltaMem := impact.MemoryDelta - data.MemMeanY
	deltaGPU := impact.GPUDelta - data.GPUMeanY
	deltaVRAM := impact.VRAMDelta - data.VRAMMeanY

	// Update means
	data.MeanX += deltaX / n
	data.CPUMeanY += deltaCPU / n
	data.MemMeanY += deltaMem / n
	data.GPUMeanY += deltaGPU / n
	data.VRAMMeanY += deltaVRAM / n

	// New deviations (after mean update)
	deltaX2 := x - data.MeanX
	deltaCPU2 := impact.CPUDelta - data.CPUMeanY
	deltaMem2 := impact.MemoryDelta - data.MemMeanY
	deltaGPU2 := impact.GPUDelta - data.GPUMeanY
	deltaVRAM2 := impact.VRAMDelta - data.VRAMMeanY

	// Update variance of X
	data.VarX += deltaX * deltaX2

	// Update covariances: Cov += (x - old_mean_x)(y - new_mean_y)
	data.CPUCov += deltaX * deltaCPU2
	data.MemCov += deltaX * deltaMem2
	data.GPUCov += deltaX * deltaGPU2
	data.VRAMCov += deltaX * deltaVRAM2

	data.Count++
}

// Confidence returns confidence based on observation count and variance.
func (m *LinearModel) Confidence(task string) float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	data, exists := m.tasks[task]
	if !exists || data.Count == 0 {
		return 0
	}

	if data.Count < int64(m.minObservations) {
		return 0
	}

	// Base confidence from count: 0.5 at minObs, approaching 1.0 as count increases
	countFactor := 0.5 + 0.5*(1.0-math.Exp(-float64(data.Count-int64(m.minObservations))/10.0))

	// Penalize if no variance in X (all observations have same complexity)
	if data.VarX < 1e-10 {
		countFactor *= 0.5
	}

	return math.Min(countFactor, 1.0)
}

// Stats returns model statistics.
func (m *LinearModel) Stats() *Stats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var totalObs int64
	taskStats := make(map[string]*TaskStats)

	for name, data := range m.tasks {
		totalObs += data.Count

		var coefs *Coefficients
		if data.Count >= int64(m.minObservations) {
			coefs = m.calculateCoefficients(data)
		}

		taskStats[name] = &TaskStats{
			Task:         name,
			Count:        data.Count,
			AvgCPUDelta:  data.CPUMeanY,
			AvgMemDelta:  data.MemMeanY,
			AvgGPUDelta:  data.GPUMeanY,
			AvgVRAMDelta: data.VRAMMeanY,
			Coefficients: coefs,
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
func (m *LinearModel) TaskStats(task string) *TaskStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	data, exists := m.tasks[task]
	if !exists {
		return nil
	}

	var coefs *Coefficients
	if data.Count >= int64(m.minObservations) {
		coefs = m.calculateCoefficients(data)
	}

	return &TaskStats{
		Task:         task,
		Count:        data.Count,
		AvgCPUDelta:  data.CPUMeanY,
		AvgMemDelta:  data.MemMeanY,
		AvgGPUDelta:  data.GPUMeanY,
		AvgVRAMDelta: data.VRAMMeanY,
		Coefficients: coefs,
	}
}

// NeedsRetrain returns false (online learning doesn't need retraining).
func (m *LinearModel) NeedsRetrain() bool {
	return false
}

// Retrain is a no-op for online learning models.
func (m *LinearModel) Retrain() error {
	return nil
}

// Save serializes the model state to a writer.
func (m *LinearModel) Save(w io.Writer) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	state := linearState{
		MinObservations: m.minObservations,
		Tasks:           m.tasks,
	}

	return json.NewEncoder(w).Encode(state)
}

// Load deserializes the model state from a reader.
func (m *LinearModel) Load(r io.Reader) error {
	var state linearState
	if err := json.NewDecoder(r).Decode(&state); err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.minObservations = state.MinObservations
	m.tasks = state.Tasks
	if m.tasks == nil {
		m.tasks = make(map[string]*linearTaskData)
	}

	return nil
}

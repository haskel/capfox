package model

import (
	"encoding/json"
	"io"
	"sync"
	"time"

	"github.com/haskel/capfox/internal/decision"
)

// GradientBoostConfig holds configuration for gradient boosting model.
type GradientBoostConfig struct {
	NEstimators     int
	MaxDepth        int
	RetrainInterval time.Duration
	MinObservations int
	MaxBufferSize   int
}

// GradientBoostModel uses gradient boosting for predictions.
// This is a batch learning model - observations are buffered and the model
// is retrained periodically.
type GradientBoostModel struct {
	config GradientBoostConfig
	mu     sync.RWMutex

	// Per-task data
	tasks map[string]*gradientBoostTaskData

	// Last retrain time
	lastRetrain time.Time
}

type gradientBoostTaskData struct {
	Count int64 `json:"count"`

	// Buffered observations for batch training
	Observations []boostObservation `json:"observations"`

	// Current model coefficients (simplified - using linear approximation for now)
	// In a real implementation, this would be the boosted trees
	CPUCoefs  []float64 `json:"cpu_coefs"`
	MemCoefs  []float64 `json:"mem_coefs"`
	GPUCoefs  []float64 `json:"gpu_coefs"`
	VRAMCoefs []float64 `json:"vram_coefs"`

	// Statistics
	SumCPU  float64 `json:"sum_cpu"`
	SumMem  float64 `json:"sum_mem"`
	SumGPU  float64 `json:"sum_gpu"`
	SumVRAM float64 `json:"sum_vram"`

	// Track if model needs retraining
	needsRetrain bool
}

type boostObservation struct {
	Complexity int     `json:"x"`
	CPUDelta   float64 `json:"cpu"`
	MemDelta   float64 `json:"mem"`
	GPUDelta   float64 `json:"gpu"`
	VRAMDelta  float64 `json:"vram"`
}

type gradientBoostState struct {
	Config      GradientBoostConfig               `json:"config"`
	Tasks       map[string]*gradientBoostTaskData `json:"tasks"`
	LastRetrain time.Time                         `json:"last_retrain"`
}

// NewGradientBoostModel creates a new gradient boosting model.
func NewGradientBoostModel(cfg GradientBoostConfig) *GradientBoostModel {
	if cfg.MinObservations < 10 {
		cfg.MinObservations = 10
	}
	if cfg.MaxBufferSize < 100 {
		cfg.MaxBufferSize = 100
	}
	if cfg.RetrainInterval == 0 {
		cfg.RetrainInterval = time.Hour
	}
	return &GradientBoostModel{
		config:      cfg,
		tasks:       make(map[string]*gradientBoostTaskData),
		lastRetrain: time.Now(),
	}
}

// Name returns the model name.
func (m *GradientBoostModel) Name() string {
	return string(ModelTypeGradientBoost)
}

// LearningType returns batch learning type.
func (m *GradientBoostModel) LearningType() LearningType {
	return LearningTypeBatch
}

// Predict returns predicted resource impact.
func (m *GradientBoostModel) Predict(task string, complexity int) *decision.ResourceImpact {
	m.mu.RLock()
	defer m.mu.RUnlock()

	data, exists := m.tasks[task]
	if !exists || data.Count < int64(m.config.MinObservations) {
		return nil
	}

	if len(data.CPUCoefs) == 0 {
		return nil
	}

	x := float64(complexity)

	return &decision.ResourceImpact{
		CPUDelta:    evaluateBoostCoefs(data.CPUCoefs, x),
		MemoryDelta: evaluateBoostCoefs(data.MemCoefs, x),
		GPUDelta:    evaluateBoostCoefs(data.GPUCoefs, x),
		VRAMDelta:   evaluateBoostCoefs(data.VRAMCoefs, x),
	}
}

// evaluateBoostCoefs evaluates prediction from coefficients.
// For now, using simple linear: coefs[0] + coefs[1]*x
func evaluateBoostCoefs(coefs []float64, x float64) float64 {
	if len(coefs) == 0 {
		return 0
	}
	if len(coefs) == 1 {
		return coefs[0]
	}
	return coefs[0] + coefs[1]*x
}

// Observe records an observation in the buffer.
func (m *GradientBoostModel) Observe(task string, complexity int, impact *decision.ResourceImpact) {
	if impact == nil {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	data, exists := m.tasks[task]
	if !exists {
		data = &gradientBoostTaskData{
			Observations: make([]boostObservation, 0, 100),
		}
		m.tasks[task] = data
	}

	obs := boostObservation{
		Complexity: complexity,
		CPUDelta:   impact.CPUDelta,
		MemDelta:   impact.MemoryDelta,
		GPUDelta:   impact.GPUDelta,
		VRAMDelta:  impact.VRAMDelta,
	}

	data.Observations = append(data.Observations, obs)
	data.Count++
	data.SumCPU += impact.CPUDelta
	data.SumMem += impact.MemoryDelta
	data.SumGPU += impact.GPUDelta
	data.SumVRAM += impact.VRAMDelta

	// Keep buffer size manageable
	if len(data.Observations) > m.config.MaxBufferSize {
		// Keep the last MaxBufferSize observations
		data.Observations = data.Observations[len(data.Observations)-m.config.MaxBufferSize:]
	}

	data.needsRetrain = true
}

// Confidence returns confidence based on observation count.
func (m *GradientBoostModel) Confidence(task string) float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	data, exists := m.tasks[task]
	if !exists || data.Count == 0 {
		return 0
	}

	if data.Count < int64(m.config.MinObservations) {
		return 0
	}

	if len(data.CPUCoefs) == 0 {
		return 0 // Model not trained yet
	}

	// Confidence grows with observations
	countFactor := 1.0 - fastBoostExp(-float64(data.Count-int64(m.config.MinObservations))/50.0)
	return min(countFactor, 1.0)
}

func fastBoostExp(x float64) float64 {
	if x > -0.1 {
		return 1 + x + x*x/2
	}
	t := 1 + x/10
	return t * t * t * t * t * t * t * t * t * t
}

// Stats returns model statistics.
func (m *GradientBoostModel) Stats() *Stats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var totalObs int64
	taskStats := make(map[string]*TaskStats)

	for name, data := range m.tasks {
		totalObs += data.Count

		avgCPU := 0.0
		avgMem := 0.0
		avgGPU := 0.0
		avgVRAM := 0.0
		if data.Count > 0 {
			avgCPU = data.SumCPU / float64(data.Count)
			avgMem = data.SumMem / float64(data.Count)
			avgGPU = data.SumGPU / float64(data.Count)
			avgVRAM = data.SumVRAM / float64(data.Count)
		}

		taskStats[name] = &TaskStats{
			Task:         name,
			Count:        data.Count,
			AvgCPUDelta:  avgCPU,
			AvgMemDelta:  avgMem,
			AvgGPUDelta:  avgGPU,
			AvgVRAMDelta: avgVRAM,
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
func (m *GradientBoostModel) TaskStats(task string) *TaskStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	data, exists := m.tasks[task]
	if !exists {
		return nil
	}

	avgCPU := 0.0
	avgMem := 0.0
	avgGPU := 0.0
	avgVRAM := 0.0
	if data.Count > 0 {
		avgCPU = data.SumCPU / float64(data.Count)
		avgMem = data.SumMem / float64(data.Count)
		avgGPU = data.SumGPU / float64(data.Count)
		avgVRAM = data.SumVRAM / float64(data.Count)
	}

	return &TaskStats{
		Task:         task,
		Count:        data.Count,
		AvgCPUDelta:  avgCPU,
		AvgMemDelta:  avgMem,
		AvgGPUDelta:  avgGPU,
		AvgVRAMDelta: avgVRAM,
	}
}

// NeedsRetrain checks if any task needs retraining.
func (m *GradientBoostModel) NeedsRetrain() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Check if retrain interval has passed
	if time.Since(m.lastRetrain) < m.config.RetrainInterval {
		return false
	}

	// Check if any task needs retraining
	for _, data := range m.tasks {
		if data.needsRetrain && data.Count >= int64(m.config.MinObservations) {
			return true
		}
	}

	return false
}

// Retrain retrains the model using buffered observations.
// In a real implementation, this would use actual gradient boosting.
// For now, using simple linear regression as a placeholder.
func (m *GradientBoostModel) Retrain() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, data := range m.tasks {
		if !data.needsRetrain || len(data.Observations) < m.config.MinObservations {
			continue
		}

		// Simple linear regression for now
		n := float64(len(data.Observations))
		var sumX, sumX2 float64
		var sumCPU, sumXCPU float64
		var sumMem, sumXMem float64
		var sumGPU, sumXGPU float64
		var sumVRAM, sumXVRAM float64

		for _, obs := range data.Observations {
			x := float64(obs.Complexity)
			sumX += x
			sumX2 += x * x
			sumCPU += obs.CPUDelta
			sumXCPU += x * obs.CPUDelta
			sumMem += obs.MemDelta
			sumXMem += x * obs.MemDelta
			sumGPU += obs.GPUDelta
			sumXGPU += x * obs.GPUDelta
			sumVRAM += obs.VRAMDelta
			sumXVRAM += x * obs.VRAMDelta
		}

		denom := n*sumX2 - sumX*sumX
		if denom < 1e-10 {
			// No variance, use mean
			data.CPUCoefs = []float64{sumCPU / n}
			data.MemCoefs = []float64{sumMem / n}
			data.GPUCoefs = []float64{sumGPU / n}
			data.VRAMCoefs = []float64{sumVRAM / n}
		} else {
			// Linear regression: y = a + b*x
			// b = (n*sum(xy) - sum(x)*sum(y)) / (n*sum(x²) - sum(x)²)
			// a = mean(y) - b*mean(x)
			meanX := sumX / n

			bCPU := (n*sumXCPU - sumX*sumCPU) / denom
			aCPU := sumCPU/n - bCPU*meanX
			data.CPUCoefs = []float64{aCPU, bCPU}

			bMem := (n*sumXMem - sumX*sumMem) / denom
			aMem := sumMem/n - bMem*meanX
			data.MemCoefs = []float64{aMem, bMem}

			bGPU := (n*sumXGPU - sumX*sumGPU) / denom
			aGPU := sumGPU/n - bGPU*meanX
			data.GPUCoefs = []float64{aGPU, bGPU}

			bVRAM := (n*sumXVRAM - sumX*sumVRAM) / denom
			aVRAM := sumVRAM/n - bVRAM*meanX
			data.VRAMCoefs = []float64{aVRAM, bVRAM}
		}

		data.needsRetrain = false
	}

	m.lastRetrain = time.Now()
	return nil
}

// Save serializes the model state to a writer.
func (m *GradientBoostModel) Save(w io.Writer) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	state := gradientBoostState{
		Config:      m.config,
		Tasks:       m.tasks,
		LastRetrain: m.lastRetrain,
	}

	return json.NewEncoder(w).Encode(state)
}

// Load deserializes the model state from a reader.
func (m *GradientBoostModel) Load(r io.Reader) error {
	var state gradientBoostState
	if err := json.NewDecoder(r).Decode(&state); err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.config = state.Config
	m.tasks = state.Tasks
	m.lastRetrain = state.LastRetrain
	if m.tasks == nil {
		m.tasks = make(map[string]*gradientBoostTaskData)
	}

	return nil
}

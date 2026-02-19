package model

import (
	"encoding/json"
	"io"
	"math"
	"sync"

	"github.com/haskel/capfox/internal/decision"
)

// PolynomialModel uses polynomial regression for predictions.
// Prediction: impact = a0 + a1*x + a2*x² + ... + an*x^n
// Uses a simplified approach with buffered observations.
type PolynomialModel struct {
	degree          int
	minObservations int
	mu              sync.RWMutex

	// Per-task polynomial data
	tasks map[string]*polynomialTaskData
}

// polynomialTaskData holds observations for polynomial regression.
type polynomialTaskData struct {
	Count int64 `json:"count"`

	// Buffered observations (limited size for memory)
	Observations []observation `json:"observations"`

	// Cached coefficients (recalculated on Observe)
	CPUCoefs  []float64 `json:"cpu_coefs"`
	MemCoefs  []float64 `json:"mem_coefs"`
	GPUCoefs  []float64 `json:"gpu_coefs"`
	VRAMCoefs []float64 `json:"vram_coefs"`

	// Statistics
	SumCPU  float64 `json:"sum_cpu"`
	SumMem  float64 `json:"sum_mem"`
	SumGPU  float64 `json:"sum_gpu"`
	SumVRAM float64 `json:"sum_vram"`
}

type observation struct {
	Complexity int     `json:"x"`
	CPUDelta   float64 `json:"cpu"`
	MemDelta   float64 `json:"mem"`
	GPUDelta   float64 `json:"gpu"`
	VRAMDelta  float64 `json:"vram"`
}

type polynomialState struct {
	Degree          int                             `json:"degree"`
	MinObservations int                             `json:"min_observations"`
	Tasks           map[string]*polynomialTaskData `json:"tasks"`
}

const maxPolynomialObservations = 1000

// NewPolynomialModel creates a new polynomial regression model.
func NewPolynomialModel(degree, minObs int) *PolynomialModel {
	if degree < 1 {
		degree = 2
	}
	if degree > 5 {
		degree = 5 // Limit for numerical stability
	}
	if minObs < degree+1 {
		minObs = degree + 1
	}
	return &PolynomialModel{
		degree:          degree,
		minObservations: minObs,
		tasks:           make(map[string]*polynomialTaskData),
	}
}

// Name returns the model name.
func (m *PolynomialModel) Name() string {
	return string(ModelTypePolynomial)
}

// LearningType returns online learning type.
func (m *PolynomialModel) LearningType() LearningType {
	return LearningTypeOnline
}

// Predict returns predicted resource impact based on polynomial regression.
func (m *PolynomialModel) Predict(task string, complexity int) *decision.ResourceImpact {
	m.mu.RLock()
	defer m.mu.RUnlock()

	data, exists := m.tasks[task]
	if !exists || data.Count < int64(m.minObservations) {
		return nil
	}

	x := float64(complexity)

	return &decision.ResourceImpact{
		CPUDelta:    evaluatePolynomial(data.CPUCoefs, x),
		MemoryDelta: evaluatePolynomial(data.MemCoefs, x),
		GPUDelta:    evaluatePolynomial(data.GPUCoefs, x),
		VRAMDelta:   evaluatePolynomial(data.VRAMCoefs, x),
	}
}

// evaluatePolynomial evaluates polynomial at point x.
// coefs[0] + coefs[1]*x + coefs[2]*x² + ...
func evaluatePolynomial(coefs []float64, x float64) float64 {
	if len(coefs) == 0 {
		return 0
	}

	result := 0.0
	xPow := 1.0
	for _, c := range coefs {
		result += c * xPow
		xPow *= x
	}
	return result
}

// Observe records an observation and updates the model.
func (m *PolynomialModel) Observe(task string, complexity int, impact *decision.ResourceImpact) {
	if impact == nil {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	data, exists := m.tasks[task]
	if !exists {
		data = &polynomialTaskData{
			Observations: make([]observation, 0, 100),
		}
		m.tasks[task] = data
	}

	obs := observation{
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
	if len(data.Observations) > maxPolynomialObservations {
		// Keep the last maxPolynomialObservations
		data.Observations = data.Observations[len(data.Observations)-maxPolynomialObservations:]
	}

	// Recalculate coefficients if we have enough observations
	if data.Count >= int64(m.minObservations) {
		m.fitPolynomial(data)
	}
}

// fitPolynomial fits polynomial coefficients using least squares.
func (m *PolynomialModel) fitPolynomial(data *polynomialTaskData) {
	n := len(data.Observations)
	degree := m.degree

	// Build Vandermonde matrix: X[i][j] = x_i^j
	// And solve X'X * coefs = X'y for each resource

	// Build X'X matrix (symmetric, size degree+1)
	// X'X[i][j] = sum(x^(i+j))
	xtx := make([][]float64, degree+1)
	for i := range xtx {
		xtx[i] = make([]float64, degree+1)
	}

	// Build X'y vectors
	xtyCPU := make([]float64, degree+1)
	xtyMem := make([]float64, degree+1)
	xtyGPU := make([]float64, degree+1)
	xtyVRAM := make([]float64, degree+1)

	// Fill matrices
	for _, obs := range data.Observations {
		x := float64(obs.Complexity)
		xPows := make([]float64, 2*degree+1)
		xPows[0] = 1
		for p := 1; p <= 2*degree; p++ {
			xPows[p] = xPows[p-1] * x
		}

		for i := 0; i <= degree; i++ {
			for j := 0; j <= degree; j++ {
				xtx[i][j] += xPows[i+j]
			}
			xtyCPU[i] += xPows[i] * obs.CPUDelta
			xtyMem[i] += xPows[i] * obs.MemDelta
			xtyGPU[i] += xPows[i] * obs.GPUDelta
			xtyVRAM[i] += xPows[i] * obs.VRAMDelta
		}
	}

	// Solve using Gaussian elimination with regularization
	lambda := 1e-6 * float64(n) // Ridge regularization for stability
	for i := 0; i <= degree; i++ {
		xtx[i][i] += lambda
	}

	data.CPUCoefs = solveLinearSystem(xtx, xtyCPU)
	data.MemCoefs = solveLinearSystem(copyMatrix(xtx), xtyMem)
	data.GPUCoefs = solveLinearSystem(copyMatrix(xtx), xtyGPU)
	data.VRAMCoefs = solveLinearSystem(copyMatrix(xtx), xtyVRAM)
}

// copyMatrix creates a deep copy of a matrix.
func copyMatrix(m [][]float64) [][]float64 {
	result := make([][]float64, len(m))
	for i := range m {
		result[i] = make([]float64, len(m[i]))
		copy(result[i], m[i])
	}
	return result
}

// solveLinearSystem solves Ax = b using Gaussian elimination with partial pivoting.
func solveLinearSystem(A [][]float64, b []float64) []float64 {
	n := len(b)
	if n == 0 || len(A) != n {
		return nil
	}

	// Forward elimination with partial pivoting
	for k := 0; k < n; k++ {
		// Find pivot
		maxIdx := k
		maxVal := math.Abs(A[k][k])
		for i := k + 1; i < n; i++ {
			if math.Abs(A[i][k]) > maxVal {
				maxIdx = i
				maxVal = math.Abs(A[i][k])
			}
		}

		// Swap rows
		if maxIdx != k {
			A[k], A[maxIdx] = A[maxIdx], A[k]
			b[k], b[maxIdx] = b[maxIdx], b[k]
		}

		// Check for singular matrix
		if math.Abs(A[k][k]) < 1e-12 {
			// Return zero coefficients for singular matrix
			return make([]float64, n)
		}

		// Eliminate
		for i := k + 1; i < n; i++ {
			factor := A[i][k] / A[k][k]
			for j := k; j < n; j++ {
				A[i][j] -= factor * A[k][j]
			}
			b[i] -= factor * b[k]
		}
	}

	// Back substitution
	x := make([]float64, n)
	for i := n - 1; i >= 0; i-- {
		x[i] = b[i]
		for j := i + 1; j < n; j++ {
			x[i] -= A[i][j] * x[j]
		}
		x[i] /= A[i][i]
	}

	return x
}

// Confidence returns confidence based on observation count and model fit.
func (m *PolynomialModel) Confidence(task string) float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	data, exists := m.tasks[task]
	if !exists || data.Count == 0 {
		return 0
	}

	if data.Count < int64(m.minObservations) {
		return 0
	}

	// Confidence based on count and degree
	countFactor := 1.0 - math.Exp(-float64(data.Count-int64(m.minObservations))/20.0)

	// Lower confidence for higher degree (more prone to overfitting)
	degreePenalty := 1.0 - 0.1*float64(m.degree-1)
	if degreePenalty < 0.5 {
		degreePenalty = 0.5
	}

	return math.Min(countFactor*degreePenalty, 1.0)
}

// Stats returns model statistics.
func (m *PolynomialModel) Stats() *Stats {
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
func (m *PolynomialModel) TaskStats(task string) *TaskStats {
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

// NeedsRetrain returns false (online learning doesn't need retraining).
func (m *PolynomialModel) NeedsRetrain() bool {
	return false
}

// Retrain is a no-op for online learning models.
func (m *PolynomialModel) Retrain() error {
	return nil
}

// Save serializes the model state to a writer.
func (m *PolynomialModel) Save(w io.Writer) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	state := polynomialState{
		Degree:          m.degree,
		MinObservations: m.minObservations,
		Tasks:           m.tasks,
	}

	return json.NewEncoder(w).Encode(state)
}

// Load deserializes the model state from a reader.
func (m *PolynomialModel) Load(r io.Reader) error {
	var state polynomialState
	if err := json.NewDecoder(r).Decode(&state); err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.degree = state.Degree
	m.minObservations = state.MinObservations
	m.tasks = state.Tasks
	if m.tasks == nil {
		m.tasks = make(map[string]*polynomialTaskData)
	}

	return nil
}

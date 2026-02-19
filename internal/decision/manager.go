package decision

import (
	"sync"

	"github.com/haskel/capfox/internal/monitor"
)

// Strategy defines the interface for decision-making strategies.
// This is duplicated here to avoid import cycles.
// The actual implementation is in decision/strategy package.
type Strategy interface {
	Name() string
	Decide(ctx *Context) *Result
}

// PredictionModel defines the interface for resource impact prediction.
// This is duplicated here to avoid import cycles.
// The actual implementation is in decision/model package.
type PredictionModel interface {
	Name() string
	Predict(task string, complexity int) *ResourceImpact
	Observe(task string, complexity int, impact *ResourceImpact)
	Confidence(task string) float64
}

// Manager coordinates decision making.
type Manager struct {
	strategy   Strategy
	model      PredictionModel
	aggregator *monitor.Aggregator
	thresholds *ThresholdsConfig

	// For queue-aware strategy
	mu           sync.RWMutex
	pendingTasks []PendingTask
}

// ManagerConfig holds manager configuration.
type ManagerConfig struct {
	Thresholds   *ThresholdsConfig
	SafetyBuffer float64
}

// NewManager creates a new decision manager.
func NewManager(
	strategy Strategy,
	model PredictionModel,
	aggregator *monitor.Aggregator,
	cfg ManagerConfig,
) *Manager {
	return &Manager{
		strategy:     strategy,
		model:        model,
		aggregator:   aggregator,
		thresholds:   cfg.Thresholds,
		pendingTasks: make([]PendingTask, 0),
	}
}

// Decide makes a decision about whether a task can run.
func (m *Manager) Decide(task string, complexity int, resources *ResourceEstimate) *Result {
	// Build context
	ctx := NewContext(task, complexity).
		WithResources(resources).
		WithCurrentState(m.aggregator.GetState()).
		WithThresholds(m.thresholds)

	// Get prediction from model
	if m.model != nil {
		prediction := m.model.Predict(task, complexity)
		ctx.WithPrediction(prediction)
	}

	// Add pending tasks for queue-aware strategy
	m.mu.RLock()
	ctx.WithPendingTasks(m.pendingTasks)
	m.mu.RUnlock()

	// Delegate to strategy
	return m.strategy.Decide(ctx)
}

// AddPendingTask adds a task to the pending list.
func (m *Manager) AddPendingTask(task PendingTask) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.pendingTasks = append(m.pendingTasks, task)
}

// RemovePendingTask removes a task from the pending list.
func (m *Manager) RemovePendingTask(taskID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, t := range m.pendingTasks {
		if t.Task == taskID {
			m.pendingTasks = append(m.pendingTasks[:i], m.pendingTasks[i+1:]...)
			return
		}
	}
}

// PendingCount returns the number of pending tasks.
func (m *Manager) PendingCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.pendingTasks)
}

// Strategy returns the current strategy.
func (m *Manager) Strategy() Strategy {
	return m.strategy
}

// Model returns the current model.
func (m *Manager) Model() PredictionModel {
	return m.model
}

// UpdateThresholds updates the threshold configuration.
func (m *Manager) UpdateThresholds(thresholds *ThresholdsConfig) {
	m.thresholds = thresholds
}

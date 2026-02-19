package model

import (
	"io"

	"github.com/haskel/capfox/internal/decision"
)

// NoopModel is a model that doesn't predict anything.
// Used with threshold strategy when prediction is not needed.
type NoopModel struct{}

// NewNoopModel creates a new noop model.
func NewNoopModel() *NoopModel {
	return &NoopModel{}
}

// Name returns the model name.
func (m *NoopModel) Name() string {
	return string(ModelTypeNone)
}

// LearningType returns the learning type.
func (m *NoopModel) LearningType() LearningType {
	return LearningTypeOnline
}

// Predict always returns nil (no prediction).
func (m *NoopModel) Predict(task string, complexity int) *decision.ResourceImpact {
	return nil
}

// Observe is a no-op for this model.
func (m *NoopModel) Observe(task string, complexity int, impact *decision.ResourceImpact) {
	// noop
}

// Confidence always returns 0.
func (m *NoopModel) Confidence(task string) float64 {
	return 0
}

// Stats returns empty stats.
func (m *NoopModel) Stats() *Stats {
	return &Stats{
		ModelName:    m.Name(),
		LearningType: m.LearningType().String(),
		Tasks:        make(map[string]*TaskStats),
	}
}

// TaskStats returns nil for any task.
func (m *NoopModel) TaskStats(task string) *TaskStats {
	return nil
}

// NeedsRetrain always returns false.
func (m *NoopModel) NeedsRetrain() bool {
	return false
}

// Retrain is a no-op.
func (m *NoopModel) Retrain() error {
	return nil
}

// Save is a no-op.
func (m *NoopModel) Save(w io.Writer) error {
	return nil
}

// Load is a no-op.
func (m *NoopModel) Load(r io.Reader) error {
	return nil
}

package learning

import (
	"github.com/haskel/capfox/internal/decision"
	"github.com/haskel/capfox/internal/decision/model"
)

// ModelAdapter wraps decision/model.PredictionModel to implement the old learning.Model interface.
// This allows gradual migration to the new model system.
type ModelAdapter struct {
	model    model.PredictionModel
	observer StatsObserver
}

// NewModelAdapter creates a new adapter wrapping a PredictionModel.
func NewModelAdapter(m model.PredictionModel) *ModelAdapter {
	return &ModelAdapter{model: m}
}

// Name returns the model name.
func (a *ModelAdapter) Name() string {
	return a.model.Name()
}

// Observe records an observation.
func (a *ModelAdapter) Observe(task string, complexity int, impact *ResourceImpact) {
	if impact == nil {
		return
	}

	// Convert to decision.ResourceImpact
	decisionImpact := &decision.ResourceImpact{
		CPUDelta:    impact.CPUDelta,
		MemoryDelta: impact.MemoryDelta,
		GPUDelta:    impact.GPUDelta,
		VRAMDelta:   impact.VRAMDelta,
	}

	a.model.Observe(task, complexity, decisionImpact)

	// Notify observer if set
	if a.observer != nil {
		stats := a.GetTaskStats(task)
		a.observer(task, stats)
	}
}

// Predict returns predicted resource impact.
func (a *ModelAdapter) Predict(task string, complexity int) *ResourceImpact {
	prediction := a.model.Predict(task, complexity)
	if prediction == nil {
		return nil
	}

	return &ResourceImpact{
		CPUDelta:    prediction.CPUDelta,
		MemoryDelta: prediction.MemoryDelta,
		GPUDelta:    prediction.GPUDelta,
		VRAMDelta:   prediction.VRAMDelta,
	}
}

// GetStats returns statistics for all tasks.
func (a *ModelAdapter) GetStats() *AllStats {
	modelStats := a.model.Stats()
	if modelStats == nil {
		return &AllStats{Tasks: make(map[string]*TaskStats)}
	}

	tasks := make(map[string]*TaskStats)
	var total int64

	for name, ts := range modelStats.Tasks {
		tasks[name] = &TaskStats{
			Task:         ts.Task,
			Count:        ts.Count,
			AvgCPUDelta:  ts.AvgCPUDelta,
			AvgMemDelta:  ts.AvgMemDelta,
			AvgGPUDelta:  ts.AvgGPUDelta,
			AvgVRAMDelta: ts.AvgVRAMDelta,
		}
		total += ts.Count
	}

	return &AllStats{
		Tasks:      tasks,
		TotalTasks: total,
	}
}

// GetTaskStats returns statistics for a specific task.
func (a *ModelAdapter) GetTaskStats(task string) *TaskStats {
	ts := a.model.TaskStats(task)
	if ts == nil {
		return nil
	}

	return &TaskStats{
		Task:         ts.Task,
		Count:        ts.Count,
		AvgCPUDelta:  ts.AvgCPUDelta,
		AvgMemDelta:  ts.AvgMemDelta,
		AvgGPUDelta:  ts.AvgGPUDelta,
		AvgVRAMDelta: ts.AvgVRAMDelta,
	}
}

// SetObserver sets a callback for stats changes.
func (a *ModelAdapter) SetObserver(observer StatsObserver) {
	a.observer = observer
}

// LoadStats loads previously saved statistics.
// Note: This is a no-op for the new model system - use model.Load() instead.
func (a *ModelAdapter) LoadStats(stats *AllStats) {
	// The new model system uses Save/Load for persistence.
	// This method is kept for backwards compatibility but does nothing.
}

// Underlying returns the wrapped PredictionModel.
func (a *ModelAdapter) Underlying() model.PredictionModel {
	return a.model
}

// ConvertImpact converts learning.ResourceImpact to decision.ResourceImpact.
func ConvertImpact(impact *ResourceImpact) *decision.ResourceImpact {
	if impact == nil {
		return nil
	}
	return &decision.ResourceImpact{
		CPUDelta:    impact.CPUDelta,
		MemoryDelta: impact.MemoryDelta,
		GPUDelta:    impact.GPUDelta,
		VRAMDelta:   impact.VRAMDelta,
	}
}

// ConvertImpactBack converts decision.ResourceImpact to learning.ResourceImpact.
func ConvertImpactBack(impact *decision.ResourceImpact) *ResourceImpact {
	if impact == nil {
		return nil
	}
	return &ResourceImpact{
		CPUDelta:    impact.CPUDelta,
		MemoryDelta: impact.MemoryDelta,
		GPUDelta:    impact.GPUDelta,
		VRAMDelta:   impact.VRAMDelta,
	}
}

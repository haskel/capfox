package learning

// Model is the interface for learning models that predict resource impact.
type Model interface {
	// Name returns the model name.
	Name() string

	// Observe records an observation of task impact.
	Observe(task string, complexity int, impact *ResourceImpact)

	// Predict returns predicted resource impact for a task.
	Predict(task string, complexity int) *ResourceImpact

	// GetStats returns statistics for all observed tasks.
	GetStats() *AllStats

	// GetTaskStats returns statistics for a specific task.
	GetTaskStats(task string) *TaskStats

	// SetObserver sets a callback that will be called when stats change.
	SetObserver(observer StatsObserver)

	// LoadStats loads previously saved statistics into the model.
	LoadStats(stats *AllStats)
}

// StatsObserver is called when task statistics are updated.
type StatsObserver func(task string, stats *TaskStats)

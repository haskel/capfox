package model

import (
	"io"

	"github.com/haskel/capfox/internal/decision"
)

// LearningType defines how the model learns.
type LearningType int

const (
	// LearningTypeOnline - model updates immediately on Observe().
	LearningTypeOnline LearningType = iota
	// LearningTypeBatch - model updates periodically via Retrain().
	LearningTypeBatch
)

// String returns string representation of learning type.
func (t LearningType) String() string {
	switch t {
	case LearningTypeOnline:
		return "online"
	case LearningTypeBatch:
		return "batch"
	default:
		return "unknown"
	}
}

// ModelType represents the type of prediction model.
type ModelType string

const (
	ModelTypeNone          ModelType = "none"
	ModelTypeMovingAverage ModelType = "moving_average"
	ModelTypeLinear        ModelType = "linear"
)

// IsValid checks if the model type is valid.
func (m ModelType) IsValid() bool {
	switch m {
	case ModelTypeNone, ModelTypeMovingAverage, ModelTypeLinear:
		return true
	}
	return false
}

// String returns string representation.
func (m ModelType) String() string {
	return string(m)
}

// PredictionModel defines the interface for resource impact prediction.
type PredictionModel interface {
	// Name returns the model name.
	Name() string

	// LearningType returns how this model learns (online or batch).
	LearningType() LearningType

	// Predict predicts the resource impact for a task.
	// Returns nil if there's insufficient data for prediction.
	Predict(task string, complexity int) *decision.ResourceImpact

	// Observe records an observation for training.
	Observe(task string, complexity int, impact *decision.ResourceImpact)

	// Confidence returns the prediction confidence for a task (0-1).
	Confidence(task string) float64

	// Stats returns model statistics.
	Stats() *Stats

	// TaskStats returns statistics for a specific task.
	TaskStats(task string) *TaskStats

	// For batch models
	NeedsRetrain() bool
	Retrain() error

	// Persistence
	Save(w io.Writer) error
	Load(r io.Reader) error
}

// Stats contains overall model statistics.
type Stats struct {
	ModelName         string               `json:"model_name"`
	LearningType      string               `json:"learning_type"`
	TotalObservations int64                `json:"total_observations"`
	Tasks             map[string]*TaskStats `json:"tasks"`
}

// TaskStats contains statistics for a specific task type.
type TaskStats struct {
	Task         string  `json:"task"`
	Count        int64   `json:"count"`
	AvgCPUDelta  float64 `json:"avg_cpu_delta"`
	AvgMemDelta  float64 `json:"avg_mem_delta"`
	AvgGPUDelta  float64 `json:"avg_gpu_delta,omitempty"`
	AvgVRAMDelta float64 `json:"avg_vram_delta,omitempty"`

	// For linear regression model
	Coefficients *Coefficients `json:"coefficients,omitempty"`
}

// Coefficients holds regression coefficients for a task.
type Coefficients struct {
	// CPU: impact = A * complexity + B
	CPUA float64 `json:"cpu_a,omitempty"`
	CPUB float64 `json:"cpu_b,omitempty"`

	// Memory: impact = A * complexity + B
	MemA float64 `json:"mem_a,omitempty"`
	MemB float64 `json:"mem_b,omitempty"`

	// GPU: impact = A * complexity + B
	GPUA float64 `json:"gpu_a,omitempty"`
	GPUB float64 `json:"gpu_b,omitempty"`

	// VRAM: impact = A * complexity + B
	VRAMA float64 `json:"vram_a,omitempty"`
	VRAMB float64 `json:"vram_b,omitempty"`
}

package decision

import (
	"time"

	"github.com/haskel/capfox/internal/monitor"
)

// Reason represents a reason for rejecting a task.
type Reason string

const (
	ReasonCPUOverload      Reason = "cpu_overload"
	ReasonMemoryOverload   Reason = "memory_overload"
	ReasonGPUOverload      Reason = "gpu_overload"
	ReasonVRAMOverload     Reason = "vram_overload"
	ReasonStorageLow       Reason = "storage_low"
	ReasonInsufficientData Reason = "insufficient_data"
)

// ResourceEstimate represents client's estimate of resource requirements.
type ResourceEstimate struct {
	CPU    int `json:"cpu,omitempty"`
	GPU    int `json:"gpu,omitempty"`
	Memory int `json:"memory,omitempty"`
}

// ResourceImpact represents the predicted or observed impact on resources.
type ResourceImpact struct {
	CPUDelta    float64 `json:"cpu_delta"`
	MemoryDelta float64 `json:"memory_delta"`
	GPUDelta    float64 `json:"gpu_delta,omitempty"`
	VRAMDelta   float64 `json:"vram_delta,omitempty"`
}

// PendingTask represents a task awaiting observation.
type PendingTask struct {
	Task       string
	Complexity int
	StartedAt  time.Time
	Predicted  *ResourceImpact
}

// Context contains all information needed for making a decision.
type Context struct {
	// Request data
	Task       string
	Complexity int
	Resources  *ResourceEstimate // optional, from client

	// Current system state
	CurrentState *monitor.SystemState

	// Model prediction (may be nil if no data)
	Prediction *ResourceImpact

	// Configuration
	Thresholds   *ThresholdsConfig
	SafetyBuffer float64 // for conservative strategy (e.g., 0.10 = 10%)

	// Pending tasks (for queue_aware strategy)
	PendingTasks []PendingTask
}

// ThresholdsConfig holds threshold configuration for decisions.
type ThresholdsConfig struct {
	CPU     CPUThreshold
	Memory  MemoryThreshold
	GPU     GPUThreshold
	VRAM    VRAMThreshold
	Storage StorageThreshold
}

// CPUThreshold defines CPU threshold.
type CPUThreshold struct {
	MaxPercent float64
}

// MemoryThreshold defines memory threshold.
type MemoryThreshold struct {
	MaxPercent float64
}

// GPUThreshold defines GPU threshold.
type GPUThreshold struct {
	MaxPercent float64
}

// VRAMThreshold defines VRAM threshold.
type VRAMThreshold struct {
	MaxPercent float64
}

// StorageThreshold defines storage threshold.
type StorageThreshold struct {
	MinFreeGB float64
}

// FutureState represents predicted system state after task execution.
type FutureState struct {
	CPUPercent    float64 `json:"cpu_percent"`
	MemoryPercent float64 `json:"memory_percent"`
	GPUPercent    float64 `json:"gpu_percent,omitempty"`
	VRAMPercent   float64 `json:"vram_percent,omitempty"`
}

// Result contains the decision outcome.
type Result struct {
	// Main result
	Allowed bool     `json:"allowed"`
	Reasons []Reason `json:"reasons,omitempty"`

	// Predicted state after task execution
	PredictedState *FutureState `json:"predicted,omitempty"`

	// Confidence in the decision (0-1)
	Confidence float64 `json:"confidence"`

	// Metadata
	Strategy string `json:"strategy"`
	Model    string `json:"model"`
}

// NewContext creates a new decision context.
func NewContext(task string, complexity int) *Context {
	return &Context{
		Task:       task,
		Complexity: complexity,
	}
}

// WithCurrentState sets the current system state.
func (c *Context) WithCurrentState(state *monitor.SystemState) *Context {
	c.CurrentState = state
	return c
}

// WithPrediction sets the model prediction.
func (c *Context) WithPrediction(prediction *ResourceImpact) *Context {
	c.Prediction = prediction
	return c
}

// WithThresholds sets the threshold configuration.
func (c *Context) WithThresholds(thresholds *ThresholdsConfig) *Context {
	c.Thresholds = thresholds
	return c
}

// WithSafetyBuffer sets the safety buffer for conservative strategy.
func (c *Context) WithSafetyBuffer(buffer float64) *Context {
	c.SafetyBuffer = buffer
	return c
}

// WithPendingTasks sets the pending tasks for queue_aware strategy.
func (c *Context) WithPendingTasks(tasks []PendingTask) *Context {
	c.PendingTasks = tasks
	return c
}

// WithResources sets the client's resource estimate.
func (c *Context) WithResources(resources *ResourceEstimate) *Context {
	c.Resources = resources
	return c
}

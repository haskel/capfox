package learning

import "time"

// TaskRecord represents a single task execution record.
type TaskRecord struct {
	Task       string         `json:"task"`
	Complexity int            `json:"complexity,omitempty"`
	StartedAt  time.Time      `json:"started_at"`
	Impact     *ResourceImpact `json:"impact,omitempty"`
}

// ResourceImpact represents the observed impact of a task on system resources.
type ResourceImpact struct {
	CPUDelta    float64 `json:"cpu_delta"`
	MemoryDelta float64 `json:"memory_delta"`
	GPUDelta    float64 `json:"gpu_delta,omitempty"`
	VRAMDelta   float64 `json:"vram_delta,omitempty"`
}

// TaskStats holds aggregated statistics for a specific task type.
type TaskStats struct {
	Task         string  `json:"task"`
	Count        int64   `json:"count"`
	AvgCPUDelta  float64 `json:"avg_cpu_delta"`
	AvgMemDelta  float64 `json:"avg_mem_delta"`
	AvgGPUDelta  float64 `json:"avg_gpu_delta,omitempty"`
	AvgVRAMDelta float64 `json:"avg_vram_delta,omitempty"`
}

// AllStats holds statistics for all task types.
type AllStats struct {
	Tasks      map[string]*TaskStats `json:"tasks"`
	TotalTasks int64                 `json:"total_tasks"`
}

// TaskStartRequest represents the request body for POST /task/notify.
type TaskStartRequest struct {
	Task       string `json:"task"`
	Complexity int    `json:"complexity,omitempty"`
}

// TaskStartResponse represents the response for POST /task/notify.
type TaskStartResponse struct {
	Received bool   `json:"received"`
	Task     string `json:"task"`
}

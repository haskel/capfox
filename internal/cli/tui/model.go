package tui

import (
	"time"
)

// Config holds TUI configuration
type Config struct {
	ServerURL       string
	RefreshInterval time.Duration
	User            string
	Password        string
}

// Model represents the TUI state
type Model struct {
	config Config

	// Data from API
	status *StatusData
	stats  *StatsData

	// UI state
	width       int
	height      int
	loading     bool
	err         error
	lastUpdated time.Time

	// Table scroll position
	tableOffset int
}

// StatusData represents system status from /status endpoint
type StatusData struct {
	CPU     CPUStatus     `json:"cpu"`
	Memory  MemoryStatus  `json:"memory"`
	GPUs    []GPUStatus   `json:"gpus"`
	Storage StorageStatus `json:"storage"`
	Process ProcessStatus `json:"process"`
}

type CPUStatus struct {
	UsagePercent float64 `json:"usage_percent"`
}

type MemoryStatus struct {
	UsagePercent float64 `json:"usage_percent"`
	TotalBytes   uint64  `json:"total_bytes"`
	UsedBytes    uint64  `json:"used_bytes"`
}

type GPUStatus struct {
	Index          int     `json:"index"`
	Name           string  `json:"name"`
	UsagePercent   float64 `json:"usage_percent"`
	VRAMTotalBytes uint64  `json:"vram_total_bytes"`
	VRAMUsedBytes  uint64  `json:"vram_used_bytes"`
}

type StorageStatus map[string]DiskStatus

type DiskStatus struct {
	TotalBytes uint64  `json:"total_bytes"`
	FreeBytes  uint64  `json:"free_bytes"`
	UsedBytes  uint64  `json:"used_bytes"`
	UsedPct    float64 `json:"used_percent"`
}

type ProcessStatus struct {
	TotalProcesses int `json:"total_processes"`
	TotalThreads   int `json:"total_threads"`
}

// StatsData represents task statistics from /stats endpoint
type StatsData struct {
	Tasks      map[string]*TaskStats `json:"tasks"`
	TotalTasks int64                 `json:"total_tasks"`
}

type TaskStats struct {
	Task         string  `json:"task"`
	Count        int64   `json:"count"`
	AvgCPUDelta  float64 `json:"avg_cpu_delta"`
	AvgMemDelta  float64 `json:"avg_mem_delta"`
	AvgGPUDelta  float64 `json:"avg_gpu_delta,omitempty"`
	AvgVRAMDelta float64 `json:"avg_vram_delta,omitempty"`
}

// NewModel creates a new TUI model
func NewModel(cfg Config) Model {
	return Model{
		config:  cfg,
		loading: true,
	}
}

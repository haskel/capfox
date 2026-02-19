package monitor

import "time"

type Monitor interface {
	Name() string
	Collect() (any, error)
}

type CPUState struct {
	UsagePercent float64   `json:"usage_percent"`
	Cores        []float64 `json:"cores"`
}

type MemoryState struct {
	UsedBytes    uint64  `json:"used_bytes"`
	TotalBytes   uint64  `json:"total_bytes"`
	UsagePercent float64 `json:"usage_percent"`
}

type GPUState struct {
	Index          int     `json:"index"`
	Name           string  `json:"name"`
	UsagePercent   float64 `json:"usage_percent"`
	Temperature    int     `json:"temperature"`
	VRAMUsedBytes  uint64  `json:"vram_used_bytes"`
	VRAMTotalBytes uint64  `json:"vram_total_bytes"`
}

type DiskState struct {
	UsedBytes    uint64  `json:"used_bytes"`
	TotalBytes   uint64  `json:"total_bytes"`
	UsagePercent float64 `json:"usage_percent"`
}

type StorageState map[string]DiskState

type ProcessState struct {
	Processes             int   `json:"processes"`
	Threads               int   `json:"threads"`
	ContextSwitchesPerSec int64 `json:"context_switches_per_sec"`
}

type SystemState struct {
	CPU                   CPUState     `json:"cpu"`
	Memory                MemoryState  `json:"memory"`
	GPUs                  []GPUState   `json:"gpus"`
	Storage               StorageState `json:"storage"`
	Processes             int          `json:"processes"`
	Threads               int          `json:"threads"`
	ContextSwitchesPerSec int64        `json:"context_switches_per_sec"`
	Timestamp             time.Time    `json:"timestamp"`
}

func (s *SystemState) Clone() *SystemState {
	clone := *s
	clone.CPU.Cores = make([]float64, len(s.CPU.Cores))
	copy(clone.CPU.Cores, s.CPU.Cores)
	clone.GPUs = make([]GPUState, len(s.GPUs))
	copy(clone.GPUs, s.GPUs)
	clone.Storage = make(StorageState, len(s.Storage))
	for k, v := range s.Storage {
		clone.Storage[k] = v
	}
	return &clone
}

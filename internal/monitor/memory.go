package monitor

import (
	"github.com/shirou/gopsutil/v4/mem"
)

type MemoryMonitor struct{}

func NewMemoryMonitor() *MemoryMonitor {
	return &MemoryMonitor{}
}

func (m *MemoryMonitor) Name() string {
	return "memory"
}

func (m *MemoryMonitor) Collect() (any, error) {
	v, err := mem.VirtualMemory()
	if err != nil {
		return nil, err
	}

	return &MemoryState{
		UsedBytes:    v.Used,
		TotalBytes:   v.Total,
		UsagePercent: v.UsedPercent,
	}, nil
}

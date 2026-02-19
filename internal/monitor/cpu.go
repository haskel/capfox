package monitor

import (
	"github.com/shirou/gopsutil/v4/cpu"
)

type CPUMonitor struct{}

func NewCPUMonitor() *CPUMonitor {
	return &CPUMonitor{}
}

func (m *CPUMonitor) Name() string {
	return "cpu"
}

func (m *CPUMonitor) Collect() (any, error) {
	// Get overall CPU usage
	percentages, err := cpu.Percent(0, false)
	if err != nil {
		return nil, err
	}

	var overall float64
	if len(percentages) > 0 {
		overall = percentages[0]
	}

	// Get per-core usage
	corePercentages, err := cpu.Percent(0, true)
	if err != nil {
		return nil, err
	}

	return &CPUState{
		UsagePercent: overall,
		Cores:        corePercentages,
	}, nil
}

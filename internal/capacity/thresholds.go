package capacity

import (
	"github.com/haskel/capfox/internal/config"
	"github.com/haskel/capfox/internal/monitor"
)

type Reason string

const (
	ReasonCPUOverload    Reason = "cpu_overload"
	ReasonMemoryOverload Reason = "memory_overload"
	ReasonGPUOverload    Reason = "gpu_overload"
	ReasonVRAMOverload   Reason = "vram_overload"
	ReasonStorageLow     Reason = "storage_low"
)

type ThresholdChecker struct {
	thresholds config.ThresholdsConfig
}

func NewThresholdChecker(thresholds config.ThresholdsConfig) *ThresholdChecker {
	return &ThresholdChecker{thresholds: thresholds}
}

func (c *ThresholdChecker) Check(state *monitor.SystemState) []Reason {
	var reasons []Reason

	if state.CPU.UsagePercent > c.thresholds.CPU.MaxPercent {
		reasons = append(reasons, ReasonCPUOverload)
	}

	if state.Memory.UsagePercent > c.thresholds.Memory.MaxPercent {
		reasons = append(reasons, ReasonMemoryOverload)
	}

	// Check GPU thresholds
	for _, gpu := range state.GPUs {
		if gpu.UsagePercent > c.thresholds.GPU.MaxPercent {
			reasons = append(reasons, ReasonGPUOverload)
			break
		}

		vramPercent := float64(gpu.VRAMUsedBytes) / float64(gpu.VRAMTotalBytes) * 100
		if gpu.VRAMTotalBytes > 0 && vramPercent > c.thresholds.VRAM.MaxPercent {
			reasons = append(reasons, ReasonVRAMOverload)
			break
		}
	}

	// Check storage thresholds
	for _, disk := range state.Storage {
		freeGB := float64(disk.TotalBytes-disk.UsedBytes) / (1024 * 1024 * 1024)
		if freeGB < c.thresholds.Storage.MinFreeGB {
			reasons = append(reasons, ReasonStorageLow)
			break
		}
	}

	return reasons
}

func (c *ThresholdChecker) UpdateThresholds(thresholds config.ThresholdsConfig) {
	c.thresholds = thresholds
}

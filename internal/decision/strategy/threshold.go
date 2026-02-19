package strategy

import (
	"github.com/haskel/capfox/internal/decision"
)

// ThresholdStrategy makes decisions based on current resource thresholds.
// It does not use predictions - only current system state.
type ThresholdStrategy struct{}

// NewThresholdStrategy creates a new threshold strategy.
func NewThresholdStrategy() *ThresholdStrategy {
	return &ThresholdStrategy{}
}

// Name returns the strategy name.
func (s *ThresholdStrategy) Name() string {
	return string(StrategyTypeThreshold)
}

// Decide checks if current system state exceeds any thresholds.
func (s *ThresholdStrategy) Decide(ctx *decision.Context) *decision.Result {
	result := &decision.Result{
		Allowed:    true,
		Strategy:   s.Name(),
		Model:      "none",
		Confidence: 1.0, // Threshold-based decisions are deterministic
	}

	if ctx == nil || ctx.CurrentState == nil || ctx.Thresholds == nil {
		return result
	}

	reasons := s.checkThresholds(ctx)
	if len(reasons) > 0 {
		result.Allowed = false
		result.Reasons = reasons
	}

	return result
}

// checkThresholds checks all resource thresholds and returns violations.
func (s *ThresholdStrategy) checkThresholds(ctx *decision.Context) []decision.Reason {
	var reasons []decision.Reason

	state := ctx.CurrentState
	thresholds := ctx.Thresholds

	// Check CPU threshold
	if state.CPU.UsagePercent > thresholds.CPU.MaxPercent {
		reasons = append(reasons, decision.ReasonCPUOverload)
	}

	// Check Memory threshold
	if state.Memory.UsagePercent > thresholds.Memory.MaxPercent {
		reasons = append(reasons, decision.ReasonMemoryOverload)
	}

	// Check GPU thresholds
	for _, gpu := range state.GPUs {
		if gpu.UsagePercent > thresholds.GPU.MaxPercent {
			reasons = append(reasons, decision.ReasonGPUOverload)
			break
		}

		// Check VRAM
		if gpu.VRAMTotalBytes > 0 {
			vramPercent := float64(gpu.VRAMUsedBytes) / float64(gpu.VRAMTotalBytes) * 100
			if vramPercent > thresholds.VRAM.MaxPercent {
				reasons = append(reasons, decision.ReasonVRAMOverload)
				break
			}
		}
	}

	// Check storage thresholds
	for _, disk := range state.Storage {
		freeGB := float64(disk.TotalBytes-disk.UsedBytes) / (1024 * 1024 * 1024)
		if freeGB < thresholds.Storage.MinFreeGB {
			reasons = append(reasons, decision.ReasonStorageLow)
			break
		}
	}

	return reasons
}

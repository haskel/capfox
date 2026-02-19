package strategy

import (
	"github.com/haskel/capfox/internal/decision"
	"github.com/haskel/capfox/internal/decision/model"
)

// ConservativeStrategy makes decisions with an additional safety buffer.
// It adds a percentage buffer to predictions, making it more cautious than PredictiveStrategy.
// Example: with 10% buffer, a predicted 70% CPU usage is treated as 77% (70 * 1.1).
type ConservativeStrategy struct {
	model           model.PredictionModel
	safetyBuffer    float64 // e.g., 0.10 for 10% buffer
	minObservations int
	fallback        Strategy
}

// NewConservativeStrategy creates a new conservative strategy.
// safetyBuffer is a fraction (e.g., 0.10 for 10%).
func NewConservativeStrategy(m model.PredictionModel, safetyBuffer float64, minObs int, fallback Strategy) *ConservativeStrategy {
	if fallback == nil {
		fallback = NewThresholdStrategy()
	}
	return &ConservativeStrategy{
		model:           m,
		safetyBuffer:    safetyBuffer,
		minObservations: minObs,
		fallback:        fallback,
	}
}

// Name returns the strategy name.
func (s *ConservativeStrategy) Name() string {
	return string(StrategyTypeConservative)
}

// Decide makes a decision with safety buffer applied to predictions.
func (s *ConservativeStrategy) Decide(ctx *decision.Context) *decision.Result {
	if ctx == nil || ctx.CurrentState == nil || ctx.Thresholds == nil {
		return &decision.Result{
			Allowed:    true,
			Strategy:   s.Name(),
			Model:      s.model.Name(),
			Confidence: 0.0,
		}
	}

	// Check if we have enough data for prediction
	confidence := s.model.Confidence(ctx.Task)
	if confidence == 0 || ctx.Prediction == nil {
		// Fallback to threshold strategy
		result := s.fallback.Decide(ctx)
		result.Reasons = append(result.Reasons, decision.ReasonInsufficientData)
		return result
	}

	// Calculate future state with safety buffer
	futureState := s.calculateFutureStateWithBuffer(ctx)

	// Check if buffered future state exceeds thresholds
	reasons := s.checkFutureThresholds(futureState, ctx.Thresholds)

	result := &decision.Result{
		Allowed:        len(reasons) == 0,
		Reasons:        reasons,
		PredictedState: futureState,
		Confidence:     confidence,
		Strategy:       s.Name(),
		Model:          s.model.Name(),
	}

	return result
}

// calculateFutureStateWithBuffer calculates predicted state with safety buffer applied.
func (s *ConservativeStrategy) calculateFutureStateWithBuffer(ctx *decision.Context) *decision.FutureState {
	state := ctx.CurrentState
	prediction := ctx.Prediction

	// Apply buffer multiplier to predictions
	// Buffer increases the predicted impact, making the strategy more conservative
	bufferMultiplier := 1.0 + s.safetyBuffer

	bufferedCPUDelta := prediction.CPUDelta * bufferMultiplier
	bufferedMemoryDelta := prediction.MemoryDelta * bufferMultiplier
	bufferedGPUDelta := prediction.GPUDelta * bufferMultiplier
	bufferedVRAMDelta := prediction.VRAMDelta * bufferMultiplier

	future := &decision.FutureState{
		CPUPercent:    state.CPU.UsagePercent + bufferedCPUDelta,
		MemoryPercent: state.Memory.UsagePercent + bufferedMemoryDelta,
	}

	// GPU prediction
	if len(state.GPUs) > 0 {
		future.GPUPercent = state.GPUs[0].UsagePercent + bufferedGPUDelta

		if state.GPUs[0].VRAMTotalBytes > 0 {
			currentVRAMPercent := float64(state.GPUs[0].VRAMUsedBytes) / float64(state.GPUs[0].VRAMTotalBytes) * 100
			future.VRAMPercent = currentVRAMPercent + bufferedVRAMDelta
		}
	}

	// Ensure values are within bounds [0, 100]
	future.CPUPercent = clamp(future.CPUPercent, 0, 100)
	future.MemoryPercent = clamp(future.MemoryPercent, 0, 100)
	future.GPUPercent = clamp(future.GPUPercent, 0, 100)
	future.VRAMPercent = clamp(future.VRAMPercent, 0, 100)

	return future
}

// checkFutureThresholds checks if predicted future state exceeds thresholds.
func (s *ConservativeStrategy) checkFutureThresholds(future *decision.FutureState, thresholds *decision.ThresholdsConfig) []decision.Reason {
	var reasons []decision.Reason

	if future.CPUPercent > thresholds.CPU.MaxPercent {
		reasons = append(reasons, decision.ReasonCPUOverload)
	}

	if future.MemoryPercent > thresholds.Memory.MaxPercent {
		reasons = append(reasons, decision.ReasonMemoryOverload)
	}

	if future.GPUPercent > thresholds.GPU.MaxPercent {
		reasons = append(reasons, decision.ReasonGPUOverload)
	}

	if future.VRAMPercent > thresholds.VRAM.MaxPercent {
		reasons = append(reasons, decision.ReasonVRAMOverload)
	}

	return reasons
}

package strategy

import (
	"github.com/haskel/capfox/internal/decision"
	"github.com/haskel/capfox/internal/decision/model"
)

// PredictiveStrategy makes decisions based on predicted future resource state.
// Logic: future_state = current_state + predicted_impact
// Falls back to threshold strategy when insufficient data.
type PredictiveStrategy struct {
	model           model.PredictionModel
	minObservations int
	fallback        Strategy
}

// NewPredictiveStrategy creates a new predictive strategy.
func NewPredictiveStrategy(m model.PredictionModel, minObs int, fallback Strategy) *PredictiveStrategy {
	if fallback == nil {
		fallback = NewThresholdStrategy()
	}
	return &PredictiveStrategy{
		model:           m,
		minObservations: minObs,
		fallback:        fallback,
	}
}

// Name returns the strategy name.
func (s *PredictiveStrategy) Name() string {
	return string(StrategyTypePredictive)
}

// Decide makes a decision based on predicted future state.
func (s *PredictiveStrategy) Decide(ctx *decision.Context) *decision.Result {
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

	// Calculate future state
	futureState := s.calculateFutureState(ctx)

	// Check if future state exceeds thresholds
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

// calculateFutureState calculates predicted system state after task execution.
func (s *PredictiveStrategy) calculateFutureState(ctx *decision.Context) *decision.FutureState {
	state := ctx.CurrentState
	prediction := ctx.Prediction

	future := &decision.FutureState{
		CPUPercent:    state.CPU.UsagePercent + prediction.CPUDelta,
		MemoryPercent: state.Memory.UsagePercent + prediction.MemoryDelta,
	}

	// GPU prediction (use first GPU for now)
	if len(state.GPUs) > 0 {
		future.GPUPercent = state.GPUs[0].UsagePercent + prediction.GPUDelta

		if state.GPUs[0].VRAMTotalBytes > 0 {
			currentVRAMPercent := float64(state.GPUs[0].VRAMUsedBytes) / float64(state.GPUs[0].VRAMTotalBytes) * 100
			future.VRAMPercent = currentVRAMPercent + prediction.VRAMDelta
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
func (s *PredictiveStrategy) checkFutureThresholds(future *decision.FutureState, thresholds *decision.ThresholdsConfig) []decision.Reason {
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

// clamp limits value to the range [min, max].
func clamp(value, min, max float64) float64 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

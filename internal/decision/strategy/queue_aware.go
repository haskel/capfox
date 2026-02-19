package strategy

import (
	"github.com/haskel/capfox/internal/decision"
	"github.com/haskel/capfox/internal/decision/model"
)

// QueueAwareStrategy makes decisions considering currently pending tasks.
// It sums the predicted impact of all pending tasks to estimate the true future state.
type QueueAwareStrategy struct {
	model           model.PredictionModel
	minObservations int
	fallback        Strategy
}

// NewQueueAwareStrategy creates a new queue-aware strategy.
func NewQueueAwareStrategy(m model.PredictionModel, minObs int, fallback Strategy) *QueueAwareStrategy {
	if fallback == nil {
		fallback = NewThresholdStrategy()
	}
	return &QueueAwareStrategy{
		model:           m,
		minObservations: minObs,
		fallback:        fallback,
	}
}

// Name returns the strategy name.
func (s *QueueAwareStrategy) Name() string {
	return string(StrategyTypeQueueAware)
}

// Decide makes a decision considering pending tasks.
func (s *QueueAwareStrategy) Decide(ctx *decision.Context) *decision.Result {
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

	// Calculate aggregate pending task impact
	pendingImpact := s.calculatePendingImpact(ctx)

	// Calculate future state = current + pending tasks impact + new task prediction
	futureState := s.calculateFutureState(ctx, pendingImpact)

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

// calculatePendingImpact sums the predicted impact of all pending tasks.
func (s *QueueAwareStrategy) calculatePendingImpact(ctx *decision.Context) *decision.ResourceImpact {
	impact := &decision.ResourceImpact{}

	for _, task := range ctx.PendingTasks {
		// Use stored prediction if available
		if task.Predicted != nil {
			impact.CPUDelta += task.Predicted.CPUDelta
			impact.MemoryDelta += task.Predicted.MemoryDelta
			impact.GPUDelta += task.Predicted.GPUDelta
			impact.VRAMDelta += task.Predicted.VRAMDelta
		} else {
			// Otherwise, get fresh prediction from model
			prediction := s.model.Predict(task.Task, task.Complexity)
			if prediction != nil {
				impact.CPUDelta += prediction.CPUDelta
				impact.MemoryDelta += prediction.MemoryDelta
				impact.GPUDelta += prediction.GPUDelta
				impact.VRAMDelta += prediction.VRAMDelta
			}
		}
	}

	return impact
}

// calculateFutureState calculates predicted state considering pending tasks.
func (s *QueueAwareStrategy) calculateFutureState(ctx *decision.Context, pendingImpact *decision.ResourceImpact) *decision.FutureState {
	state := ctx.CurrentState
	prediction := ctx.Prediction

	// Future = Current + Pending Tasks Impact + New Task Prediction
	future := &decision.FutureState{
		CPUPercent:    state.CPU.UsagePercent + pendingImpact.CPUDelta + prediction.CPUDelta,
		MemoryPercent: state.Memory.UsagePercent + pendingImpact.MemoryDelta + prediction.MemoryDelta,
	}

	// GPU prediction
	if len(state.GPUs) > 0 {
		future.GPUPercent = state.GPUs[0].UsagePercent + pendingImpact.GPUDelta + prediction.GPUDelta

		if state.GPUs[0].VRAMTotalBytes > 0 {
			currentVRAMPercent := float64(state.GPUs[0].VRAMUsedBytes) / float64(state.GPUs[0].VRAMTotalBytes) * 100
			future.VRAMPercent = currentVRAMPercent + pendingImpact.VRAMDelta + prediction.VRAMDelta
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
func (s *QueueAwareStrategy) checkFutureThresholds(future *decision.FutureState, thresholds *decision.ThresholdsConfig) []decision.Reason {
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

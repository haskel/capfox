package strategy

import "github.com/haskel/capfox/internal/decision"

// Strategy defines the interface for decision-making strategies.
type Strategy interface {
	// Name returns the strategy name.
	Name() string

	// Decide makes a decision based on the given context.
	Decide(ctx *decision.Context) *decision.Result
}

// StrategyType represents the type of decision strategy.
type StrategyType string

const (
	StrategyTypeThreshold    StrategyType = "threshold"
	StrategyTypePredictive   StrategyType = "predictive"
	StrategyTypeConservative StrategyType = "conservative"
	StrategyTypeQueueAware   StrategyType = "queue_aware"
)

// IsValid checks if the strategy type is valid.
func (s StrategyType) IsValid() bool {
	switch s {
	case StrategyTypeThreshold, StrategyTypePredictive,
		StrategyTypeConservative, StrategyTypeQueueAware:
		return true
	}
	return false
}

// String returns string representation.
func (s StrategyType) String() string {
	return string(s)
}

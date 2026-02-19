package strategy

import (
	"fmt"

	"github.com/haskel/capfox/internal/decision/model"
)

// Config holds strategy configuration.
type Config struct {
	Type             StrategyType
	SafetyBufferPct  float64 // for conservative strategy (e.g., 10 for 10%)
	FallbackStrategy StrategyType
	MinObservations  int
}

// Factory creates decision strategies.
type Factory struct {
	model  model.PredictionModel
	config Config
}

// NewFactory creates a new strategy factory.
func NewFactory(m model.PredictionModel, cfg Config) *Factory {
	return &Factory{
		model:  m,
		config: cfg,
	}
}

// Create creates a strategy based on configuration.
func (f *Factory) Create() (Strategy, error) {
	return f.CreateByType(f.config.Type)
}

// CreateByType creates a strategy of the specified type.
func (f *Factory) CreateByType(strategyType StrategyType) (Strategy, error) {
	switch strategyType {
	case StrategyTypeThreshold:
		return NewThresholdStrategy(), nil

	case StrategyTypePredictive:
		fallback := f.createFallback()
		return NewPredictiveStrategy(f.model, f.config.MinObservations, fallback), nil

	case StrategyTypeConservative:
		fallback := f.createFallback()
		return NewConservativeStrategy(f.model, f.config.SafetyBufferPct/100, f.config.MinObservations, fallback), nil

	case StrategyTypeQueueAware:
		fallback := f.createFallback()
		return NewQueueAwareStrategy(f.model, f.config.MinObservations, fallback), nil

	default:
		return nil, fmt.Errorf("unknown strategy type: %s", strategyType)
	}
}

// createFallback creates a fallback strategy.
func (f *Factory) createFallback() Strategy {
	if f.config.FallbackStrategy == "" || f.config.FallbackStrategy == StrategyTypeThreshold {
		return NewThresholdStrategy()
	}
	// For non-threshold fallback, create it (but avoid infinite recursion)
	fallback, _ := f.CreateByType(f.config.FallbackStrategy)
	return fallback
}

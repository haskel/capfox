package model

import (
	"fmt"
	"time"
)

// Config holds model configuration.
type Config struct {
	Type            ModelType
	MinObservations int

	// MovingAverage params
	Alpha float64

	// Polynomial params
	Degree int

	// GradientBoosting params
	NEstimators     int
	MaxDepth        int
	RetrainInterval time.Duration
	MaxBufferSize   int
}

// DefaultConfig returns default model configuration.
func DefaultConfig() Config {
	return Config{
		Type:            ModelTypeLinear,
		MinObservations: 5,
		Alpha:           0.2,
		Degree:          2,
		NEstimators:     100,
		MaxDepth:        5,
		RetrainInterval: time.Hour,
		MaxBufferSize:   10000,
	}
}

// Factory creates prediction models.
type Factory struct {
	config Config
}

// NewFactory creates a new model factory.
func NewFactory(cfg Config) *Factory {
	return &Factory{config: cfg}
}

// Create creates a model based on configuration.
func (f *Factory) Create() (PredictionModel, error) {
	return f.CreateByType(f.config.Type)
}

// CreateByType creates a model of the specified type.
func (f *Factory) CreateByType(modelType ModelType) (PredictionModel, error) {
	switch modelType {
	case ModelTypeNone:
		return NewNoopModel(), nil

	case ModelTypeMovingAverage:
		return NewMovingAverageModel(f.config.Alpha), nil

	case ModelTypeLinear:
		return NewLinearModel(f.config.MinObservations), nil

	case ModelTypePolynomial:
		return NewPolynomialModel(f.config.Degree, f.config.MinObservations), nil

	case ModelTypeGradientBoost:
		return NewGradientBoostModel(GradientBoostConfig{
			NEstimators:     f.config.NEstimators,
			MaxDepth:        f.config.MaxDepth,
			RetrainInterval: f.config.RetrainInterval,
			MinObservations: f.config.MinObservations,
			MaxBufferSize:   f.config.MaxBufferSize,
		}), nil

	default:
		return nil, fmt.Errorf("unknown model type: %s", modelType)
	}
}

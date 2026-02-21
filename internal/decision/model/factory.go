package model

import (
	"fmt"
)

// Config holds model configuration.
type Config struct {
	Type            ModelType
	MinObservations int

	// MovingAverage params
	Alpha float64
}

// DefaultConfig returns default model configuration.
func DefaultConfig() Config {
	return Config{
		Type:            ModelTypeLinear,
		MinObservations: 5,
		Alpha:           0.2,
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

	default:
		return nil, fmt.Errorf("unknown model type: %s", modelType)
	}
}

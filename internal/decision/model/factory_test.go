package model

import (
	"testing"
	"time"
)

func TestFactoryCreateByType(t *testing.T) {
	factory := NewFactory(Config{
		Type:            ModelTypeLinear,
		MinObservations: 5,
		Alpha:           0.2,
		Degree:          2,
		NEstimators:     100,
		MaxDepth:        5,
		RetrainInterval: time.Hour,
		MaxBufferSize:   10000,
	})

	tests := []struct {
		modelType    ModelType
		expectError  bool
		expectedName string
		learningType LearningType
	}{
		{ModelTypeNone, false, "none", LearningTypeOnline},
		{ModelTypeMovingAverage, false, "moving_average", LearningTypeOnline},
		{ModelTypeLinear, false, "linear", LearningTypeOnline},
		{ModelTypePolynomial, false, "polynomial", LearningTypeOnline},
		{ModelTypeGradientBoost, false, "gradient_boosting", LearningTypeBatch},
		{ModelType("invalid"), true, "", LearningTypeOnline},
	}

	for _, tt := range tests {
		t.Run(string(tt.modelType), func(t *testing.T) {
			model, err := factory.CreateByType(tt.modelType)

			if tt.expectError {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if model.Name() != tt.expectedName {
				t.Errorf("expected name '%s', got '%s'", tt.expectedName, model.Name())
			}

			if model.LearningType() != tt.learningType {
				t.Errorf("expected learning type %v, got %v",
					tt.learningType, model.LearningType())
			}
		})
	}
}

func TestNoopModel(t *testing.T) {
	m := NewNoopModel()

	if m.Name() != "none" {
		t.Errorf("expected name 'none', got '%s'", m.Name())
	}

	if m.LearningType() != LearningTypeOnline {
		t.Errorf("expected online learning type")
	}

	// Predict should return nil
	prediction := m.Predict("test", 100)
	if prediction != nil {
		t.Error("expected nil prediction")
	}

	// Confidence should be 0
	if m.Confidence("test") != 0 {
		t.Errorf("expected confidence 0, got %f", m.Confidence("test"))
	}

	// NeedsRetrain should be false
	if m.NeedsRetrain() {
		t.Error("expected NeedsRetrain to be false")
	}

	// Stats should return empty stats
	stats := m.Stats()
	if stats.ModelName != "none" {
		t.Errorf("expected model name 'none', got '%s'", stats.ModelName)
	}
	if len(stats.Tasks) != 0 {
		t.Error("expected empty tasks map")
	}

	// TaskStats should return nil
	taskStats := m.TaskStats("test")
	if taskStats != nil {
		t.Error("expected nil task stats")
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Type != ModelTypeLinear {
		t.Errorf("expected default type 'linear', got '%s'", cfg.Type)
	}

	if cfg.MinObservations != 5 {
		t.Errorf("expected MinObservations 5, got %d", cfg.MinObservations)
	}

	if cfg.Alpha != 0.2 {
		t.Errorf("expected Alpha 0.2, got %f", cfg.Alpha)
	}

	if cfg.RetrainInterval != time.Hour {
		t.Errorf("expected RetrainInterval 1h, got %v", cfg.RetrainInterval)
	}
}

func TestMovingAverageModelCreation(t *testing.T) {
	// Test with valid alpha
	m := NewMovingAverageModel(0.3)
	if m.Name() != "moving_average" {
		t.Errorf("expected name 'moving_average', got '%s'", m.Name())
	}

	// Test with invalid alpha (should default to 0.2)
	m2 := NewMovingAverageModel(0)
	if m2.alpha != 0.2 {
		t.Errorf("expected default alpha 0.2, got %f", m2.alpha)
	}

	m3 := NewMovingAverageModel(1.5)
	if m3.alpha != 0.2 {
		t.Errorf("expected default alpha 0.2, got %f", m3.alpha)
	}
}

func TestLinearModelCreation(t *testing.T) {
	m := NewLinearModel(10)
	if m.Name() != "linear" {
		t.Errorf("expected name 'linear', got '%s'", m.Name())
	}
	if m.minObservations != 10 {
		t.Errorf("expected minObservations 10, got %d", m.minObservations)
	}
}

func TestPolynomialModelCreation(t *testing.T) {
	m := NewPolynomialModel(3, 10)
	if m.Name() != "polynomial" {
		t.Errorf("expected name 'polynomial', got '%s'", m.Name())
	}
	if m.degree != 3 {
		t.Errorf("expected degree 3, got %d", m.degree)
	}
}

func TestGradientBoostModelCreation(t *testing.T) {
	cfg := GradientBoostConfig{
		NEstimators:     50,
		MaxDepth:        3,
		RetrainInterval: 30 * time.Minute,
		MinObservations: 20,
		MaxBufferSize:   5000,
	}
	m := NewGradientBoostModel(cfg)

	if m.Name() != "gradient_boosting" {
		t.Errorf("expected name 'gradient_boosting', got '%s'", m.Name())
	}
	if m.LearningType() != LearningTypeBatch {
		t.Error("expected batch learning type")
	}
}

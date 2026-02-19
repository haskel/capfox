package strategy

import (
	"testing"

	"github.com/haskel/capfox/internal/decision/model"
)

func TestFactoryCreateByType(t *testing.T) {
	// Create a noop model for testing
	noopModel := model.NewNoopModel()

	factory := NewFactory(noopModel, Config{
		Type:             StrategyTypePredictive,
		SafetyBufferPct:  10,
		FallbackStrategy: StrategyTypeThreshold,
		MinObservations:  5,
	})

	tests := []struct {
		strategyType StrategyType
		expectError  bool
		expectedName string
	}{
		{StrategyTypeThreshold, false, "threshold"},
		{StrategyTypePredictive, false, "predictive"},
		{StrategyTypeConservative, false, "conservative"},
		{StrategyTypeQueueAware, false, "queue_aware"},
		{StrategyType("invalid"), true, ""},
	}

	for _, tt := range tests {
		t.Run(string(tt.strategyType), func(t *testing.T) {
			strategy, err := factory.CreateByType(tt.strategyType)

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

			if strategy.Name() != tt.expectedName {
				t.Errorf("expected name '%s', got '%s'", tt.expectedName, strategy.Name())
			}
		})
	}
}

func TestThresholdStrategyDecide(t *testing.T) {
	strategy := NewThresholdStrategy()

	if strategy.Name() != "threshold" {
		t.Errorf("expected name 'threshold', got '%s'", strategy.Name())
	}

	// Test that it returns a result (placeholder implementation)
	result := strategy.Decide(nil)
	if result == nil {
		t.Fatal("expected result, got nil")
	}
	if result.Strategy != "threshold" {
		t.Errorf("expected strategy 'threshold', got '%s'", result.Strategy)
	}
}

func TestPredictiveStrategyDecide(t *testing.T) {
	noopModel := model.NewNoopModel()
	strategy := NewPredictiveStrategy(noopModel, 5, nil)

	if strategy.Name() != "predictive" {
		t.Errorf("expected name 'predictive', got '%s'", strategy.Name())
	}

	result := strategy.Decide(nil)
	if result == nil {
		t.Fatal("expected result, got nil")
	}
	if result.Strategy != "predictive" {
		t.Errorf("expected strategy 'predictive', got '%s'", result.Strategy)
	}
}

func TestConservativeStrategyDecide(t *testing.T) {
	noopModel := model.NewNoopModel()
	strategy := NewConservativeStrategy(noopModel, 0.1, 5, nil)

	if strategy.Name() != "conservative" {
		t.Errorf("expected name 'conservative', got '%s'", strategy.Name())
	}

	result := strategy.Decide(nil)
	if result == nil {
		t.Fatal("expected result, got nil")
	}
	if result.Strategy != "conservative" {
		t.Errorf("expected strategy 'conservative', got '%s'", result.Strategy)
	}
}

func TestQueueAwareStrategyDecide(t *testing.T) {
	noopModel := model.NewNoopModel()
	strategy := NewQueueAwareStrategy(noopModel, 5, nil)

	if strategy.Name() != "queue_aware" {
		t.Errorf("expected name 'queue_aware', got '%s'", strategy.Name())
	}

	result := strategy.Decide(nil)
	if result == nil {
		t.Error("expected result, got nil")
	}
	if result.Strategy != "queue_aware" {
		t.Errorf("expected strategy 'queue_aware', got '%s'", result.Strategy)
	}
}

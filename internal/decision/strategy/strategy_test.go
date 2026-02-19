package strategy

import (
	"testing"
)

func TestStrategyTypeIsValid(t *testing.T) {
	tests := []struct {
		strategyType StrategyType
		valid        bool
	}{
		{StrategyTypeThreshold, true},
		{StrategyTypePredictive, true},
		{StrategyTypeConservative, true},
		{StrategyTypeQueueAware, true},
		{StrategyType("invalid"), false},
		{StrategyType(""), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.strategyType), func(t *testing.T) {
			if tt.strategyType.IsValid() != tt.valid {
				t.Errorf("StrategyType(%s).IsValid() = %v, want %v",
					tt.strategyType, tt.strategyType.IsValid(), tt.valid)
			}
		})
	}
}

func TestStrategyTypeString(t *testing.T) {
	tests := []struct {
		strategyType StrategyType
		expected     string
	}{
		{StrategyTypeThreshold, "threshold"},
		{StrategyTypePredictive, "predictive"},
		{StrategyTypeConservative, "conservative"},
		{StrategyTypeQueueAware, "queue_aware"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if tt.strategyType.String() != tt.expected {
				t.Errorf("StrategyType.String() = %s, want %s",
					tt.strategyType.String(), tt.expected)
			}
		})
	}
}

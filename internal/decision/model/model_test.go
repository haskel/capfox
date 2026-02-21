package model

import (
	"testing"
)

func TestModelTypeIsValid(t *testing.T) {
	tests := []struct {
		modelType ModelType
		valid     bool
	}{
		{ModelTypeNone, true},
		{ModelTypeMovingAverage, true},
		{ModelTypeLinear, true},
		{ModelType("invalid"), false},
		{ModelType(""), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.modelType), func(t *testing.T) {
			if tt.modelType.IsValid() != tt.valid {
				t.Errorf("ModelType(%s).IsValid() = %v, want %v",
					tt.modelType, tt.modelType.IsValid(), tt.valid)
			}
		})
	}
}

func TestModelTypeString(t *testing.T) {
	tests := []struct {
		modelType ModelType
		expected  string
	}{
		{ModelTypeNone, "none"},
		{ModelTypeMovingAverage, "moving_average"},
		{ModelTypeLinear, "linear"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if tt.modelType.String() != tt.expected {
				t.Errorf("ModelType.String() = %s, want %s",
					tt.modelType.String(), tt.expected)
			}
		})
	}
}

func TestLearningTypeString(t *testing.T) {
	tests := []struct {
		learningType LearningType
		expected     string
	}{
		{LearningTypeOnline, "online"},
		{LearningTypeBatch, "batch"},
		{LearningType(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if tt.learningType.String() != tt.expected {
				t.Errorf("LearningType.String() = %s, want %s",
					tt.learningType.String(), tt.expected)
			}
		})
	}
}

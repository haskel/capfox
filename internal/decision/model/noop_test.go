package model

import (
	"bytes"
	"testing"

	"github.com/haskel/capfox/internal/decision"
)

func TestNoopModel_Name(t *testing.T) {
	m := NewNoopModel()
	if m.Name() != "none" {
		t.Errorf("expected name 'none', got '%s'", m.Name())
	}
}

func TestNoopModel_LearningType(t *testing.T) {
	m := NewNoopModel()
	if m.LearningType() != LearningTypeOnline {
		t.Error("expected online learning type")
	}
}

func TestNoopModel_Predict(t *testing.T) {
	m := NewNoopModel()
	prediction := m.Predict("test", 100)
	if prediction != nil {
		t.Error("expected nil prediction")
	}
}

func TestNoopModel_Observe(t *testing.T) {
	m := NewNoopModel()
	// Should not panic
	m.Observe("test", 100, &decision.ResourceImpact{CPUDelta: 10})
}

func TestNoopModel_Confidence(t *testing.T) {
	m := NewNoopModel()
	if m.Confidence("test") != 0 {
		t.Errorf("expected confidence 0, got %f", m.Confidence("test"))
	}
}

func TestNoopModel_Stats(t *testing.T) {
	m := NewNoopModel()
	stats := m.Stats()
	if stats.ModelName != "none" {
		t.Errorf("expected model name 'none', got '%s'", stats.ModelName)
	}
	if len(stats.Tasks) != 0 {
		t.Error("expected empty tasks map")
	}
}

func TestNoopModel_TaskStats(t *testing.T) {
	m := NewNoopModel()
	taskStats := m.TaskStats("test")
	if taskStats != nil {
		t.Error("expected nil task stats")
	}
}

func TestNoopModel_NeedsRetrain(t *testing.T) {
	m := NewNoopModel()
	if m.NeedsRetrain() {
		t.Error("expected NeedsRetrain to be false")
	}
}

func TestNoopModel_SaveLoad(t *testing.T) {
	m := NewNoopModel()
	var buf bytes.Buffer
	if err := m.Save(&buf); err != nil {
		t.Errorf("Save error: %v", err)
	}
	if err := m.Load(&buf); err != nil {
		t.Errorf("Load error: %v", err)
	}
}

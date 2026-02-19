package model

import (
	"bytes"
	"math"
	"testing"

	"github.com/haskel/capfox/internal/decision"
)

func TestMovingAverageModel_Name(t *testing.T) {
	m := NewMovingAverageModel(0.3)
	if m.Name() != "moving_average" {
		t.Errorf("expected name 'moving_average', got '%s'", m.Name())
	}
}

func TestMovingAverageModel_LearningType(t *testing.T) {
	m := NewMovingAverageModel(0.3)
	if m.LearningType() != LearningTypeOnline {
		t.Error("expected online learning type")
	}
}

func TestMovingAverageModel_InvalidAlpha(t *testing.T) {
	// Alpha <= 0 should default to 0.2
	m1 := NewMovingAverageModel(0)
	if m1.alpha != 0.2 {
		t.Errorf("expected default alpha 0.2, got %f", m1.alpha)
	}

	m2 := NewMovingAverageModel(-0.5)
	if m2.alpha != 0.2 {
		t.Errorf("expected default alpha 0.2, got %f", m2.alpha)
	}

	// Alpha > 1 should default to 0.2
	m3 := NewMovingAverageModel(1.5)
	if m3.alpha != 0.2 {
		t.Errorf("expected default alpha 0.2, got %f", m3.alpha)
	}
}

func TestMovingAverageModel_PredictNoData(t *testing.T) {
	m := NewMovingAverageModel(0.3)
	prediction := m.Predict("unknown_task", 100)
	if prediction != nil {
		t.Error("expected nil prediction for unknown task")
	}
}

func TestMovingAverageModel_SingleObservation(t *testing.T) {
	m := NewMovingAverageModel(0.3)

	impact := &decision.ResourceImpact{
		CPUDelta:    10.0,
		MemoryDelta: 20.0,
		GPUDelta:    30.0,
		VRAMDelta:   40.0,
	}
	m.Observe("test", 100, impact)

	prediction := m.Predict("test", 200) // Complexity is ignored
	if prediction == nil {
		t.Fatal("expected prediction")
	}

	// First observation should be the exact value
	if prediction.CPUDelta != 10.0 {
		t.Errorf("expected CPU 10.0, got %f", prediction.CPUDelta)
	}
	if prediction.MemoryDelta != 20.0 {
		t.Errorf("expected Memory 20.0, got %f", prediction.MemoryDelta)
	}
}

func TestMovingAverageModel_MultipleObservations(t *testing.T) {
	alpha := 0.5
	m := NewMovingAverageModel(alpha)

	// First observation: CPU = 10
	m.Observe("test", 100, &decision.ResourceImpact{CPUDelta: 10.0})

	// Second observation: CPU = 20
	// EMA = alpha * new + (1-alpha) * old = 0.5 * 20 + 0.5 * 10 = 15
	m.Observe("test", 100, &decision.ResourceImpact{CPUDelta: 20.0})

	prediction := m.Predict("test", 100)
	if prediction == nil {
		t.Fatal("expected prediction")
	}

	expected := 15.0
	if math.Abs(prediction.CPUDelta-expected) > 0.001 {
		t.Errorf("expected CPU %f, got %f", expected, prediction.CPUDelta)
	}
}

func TestMovingAverageModel_ConfidenceGrows(t *testing.T) {
	m := NewMovingAverageModel(0.3)

	// No data - confidence 0
	if m.Confidence("test") != 0 {
		t.Errorf("expected confidence 0 for unknown task")
	}

	// One observation - some confidence
	m.Observe("test", 100, &decision.ResourceImpact{CPUDelta: 10.0})
	conf1 := m.Confidence("test")
	if conf1 == 0 {
		t.Error("expected non-zero confidence after 1 observation")
	}

	// More observations - higher confidence
	for i := 0; i < 10; i++ {
		m.Observe("test", 100, &decision.ResourceImpact{CPUDelta: 10.0})
	}
	conf2 := m.Confidence("test")
	if conf2 <= conf1 {
		t.Errorf("expected confidence to grow, got %f <= %f", conf2, conf1)
	}

	// Confidence should be close to 1 after many observations
	if conf2 < 0.8 {
		t.Errorf("expected confidence > 0.8 after 11 observations, got %f", conf2)
	}
}

func TestMovingAverageModel_Stats(t *testing.T) {
	m := NewMovingAverageModel(0.3)

	m.Observe("task1", 100, &decision.ResourceImpact{CPUDelta: 10.0, MemoryDelta: 20.0})
	m.Observe("task2", 200, &decision.ResourceImpact{CPUDelta: 30.0, MemoryDelta: 40.0})

	stats := m.Stats()
	if stats.ModelName != "moving_average" {
		t.Errorf("expected model name 'moving_average', got '%s'", stats.ModelName)
	}
	if stats.TotalObservations != 2 {
		t.Errorf("expected 2 total observations, got %d", stats.TotalObservations)
	}
	if len(stats.Tasks) != 2 {
		t.Errorf("expected 2 tasks, got %d", len(stats.Tasks))
	}
}

func TestMovingAverageModel_TaskStats(t *testing.T) {
	m := NewMovingAverageModel(0.3)

	// Unknown task
	if m.TaskStats("unknown") != nil {
		t.Error("expected nil for unknown task")
	}

	m.Observe("test", 100, &decision.ResourceImpact{CPUDelta: 10.0})
	taskStats := m.TaskStats("test")
	if taskStats == nil {
		t.Fatal("expected task stats")
	}
	if taskStats.Task != "test" {
		t.Errorf("expected task 'test', got '%s'", taskStats.Task)
	}
	if taskStats.Count != 1 {
		t.Errorf("expected count 1, got %d", taskStats.Count)
	}
}

func TestMovingAverageModel_SaveLoad(t *testing.T) {
	m1 := NewMovingAverageModel(0.3)
	m1.Observe("test", 100, &decision.ResourceImpact{CPUDelta: 10.0, MemoryDelta: 20.0})
	m1.Observe("test", 100, &decision.ResourceImpact{CPUDelta: 15.0, MemoryDelta: 25.0})

	// Save
	var buf bytes.Buffer
	if err := m1.Save(&buf); err != nil {
		t.Fatalf("Save error: %v", err)
	}

	// Load into new model
	m2 := NewMovingAverageModel(0.5) // Different alpha - should be overwritten
	if err := m2.Load(&buf); err != nil {
		t.Fatalf("Load error: %v", err)
	}

	// Verify state was restored
	if m2.alpha != 0.3 {
		t.Errorf("expected alpha 0.3, got %f", m2.alpha)
	}

	prediction := m2.Predict("test", 100)
	if prediction == nil {
		t.Fatal("expected prediction after load")
	}

	// Should match original model's prediction
	p1 := m1.Predict("test", 100)
	if math.Abs(prediction.CPUDelta-p1.CPUDelta) > 0.001 {
		t.Errorf("prediction mismatch: %f vs %f", prediction.CPUDelta, p1.CPUDelta)
	}
}

func TestMovingAverageModel_NilObservation(t *testing.T) {
	m := NewMovingAverageModel(0.3)
	// Should not panic
	m.Observe("test", 100, nil)

	// Should still have no data
	if m.Predict("test", 100) != nil {
		t.Error("expected nil prediction after nil observation")
	}
}

func TestMovingAverageModel_IgnoresComplexity(t *testing.T) {
	m := NewMovingAverageModel(0.3)

	// Observe with different complexities
	m.Observe("test", 100, &decision.ResourceImpact{CPUDelta: 10.0})
	m.Observe("test", 200, &decision.ResourceImpact{CPUDelta: 20.0})
	m.Observe("test", 300, &decision.ResourceImpact{CPUDelta: 30.0})

	// Predictions should be the same regardless of complexity
	p1 := m.Predict("test", 100)
	p2 := m.Predict("test", 500)

	if p1.CPUDelta != p2.CPUDelta {
		t.Errorf("predictions differ for different complexities: %f vs %f", p1.CPUDelta, p2.CPUDelta)
	}
}

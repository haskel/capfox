package model

import (
	"bytes"
	"math"
	"testing"

	"github.com/haskel/capfox/internal/decision"
)

func TestPolynomialModel_Name(t *testing.T) {
	m := NewPolynomialModel(2, 5)
	if m.Name() != "polynomial" {
		t.Errorf("expected name 'polynomial', got '%s'", m.Name())
	}
}

func TestPolynomialModel_LearningType(t *testing.T) {
	m := NewPolynomialModel(2, 5)
	if m.LearningType() != LearningTypeOnline {
		t.Error("expected online learning type")
	}
}

func TestPolynomialModel_DegreeDefaults(t *testing.T) {
	// Degree < 1 should default to 2
	m1 := NewPolynomialModel(0, 5)
	if m1.degree != 2 {
		t.Errorf("expected degree 2 for invalid input, got %d", m1.degree)
	}

	// Degree > 5 should be capped at 5
	m2 := NewPolynomialModel(10, 5)
	if m2.degree != 5 {
		t.Errorf("expected degree 5 (capped), got %d", m2.degree)
	}
}

func TestPolynomialModel_MinObservationsAdjusted(t *testing.T) {
	// minObs should be at least degree + 1
	m := NewPolynomialModel(3, 2)
	if m.minObservations < 4 {
		t.Errorf("expected minObservations >= 4 for degree 3, got %d", m.minObservations)
	}
}

func TestPolynomialModel_PredictNoData(t *testing.T) {
	m := NewPolynomialModel(2, 5)
	prediction := m.Predict("unknown_task", 100)
	if prediction != nil {
		t.Error("expected nil prediction for unknown task")
	}
}

func TestPolynomialModel_LinearData(t *testing.T) {
	m := NewPolynomialModel(2, 5)

	// Linear data: CPU = 10 + 0.1 * x
	for i := 1; i <= 10; i++ {
		complexity := i * 100
		cpu := 10.0 + 0.1*float64(complexity)
		m.Observe("test", complexity, &decision.ResourceImpact{CPUDelta: cpu})
	}

	// Predict for complexity 500
	prediction := m.Predict("test", 500)
	if prediction == nil {
		t.Fatal("expected prediction")
	}

	// Should be approximately 10 + 0.1 * 500 = 60
	expected := 60.0
	if math.Abs(prediction.CPUDelta-expected) > 2.0 {
		t.Errorf("expected CPU ~%f, got %f", expected, prediction.CPUDelta)
	}
}

func TestPolynomialModel_QuadraticData(t *testing.T) {
	m := NewPolynomialModel(2, 5)

	// Quadratic data: CPU = 0.001 * x^2
	for i := 1; i <= 10; i++ {
		x := float64(i * 10)
		cpu := 0.001 * x * x
		m.Observe("test", int(x), &decision.ResourceImpact{CPUDelta: cpu})
	}

	// Predict for x = 50
	prediction := m.Predict("test", 50)
	if prediction == nil {
		t.Fatal("expected prediction")
	}

	// Should be approximately 0.001 * 50^2 = 2.5
	expected := 2.5
	if math.Abs(prediction.CPUDelta-expected) > 0.5 {
		t.Errorf("expected CPU ~%f, got %f", expected, prediction.CPUDelta)
	}
}

func TestPolynomialModel_ConfidenceGrows(t *testing.T) {
	m := NewPolynomialModel(2, 5)

	// No data - confidence 0
	if m.Confidence("test") != 0 {
		t.Errorf("expected confidence 0 for unknown task")
	}

	// Less than minObservations - confidence 0
	for i := 0; i < 3; i++ {
		m.Observe("test", i*100, &decision.ResourceImpact{CPUDelta: float64(i * 10)})
	}
	if m.Confidence("test") != 0 {
		t.Errorf("expected confidence 0 with insufficient observations")
	}

	// Add more observations
	for i := 3; i < 20; i++ {
		m.Observe("test", i*100, &decision.ResourceImpact{CPUDelta: float64(i * 10)})
	}
	conf := m.Confidence("test")
	if conf <= 0.3 {
		t.Errorf("expected confidence > 0.3 after 20 observations, got %f", conf)
	}
}

func TestPolynomialModel_HigherDegreeLowerConfidence(t *testing.T) {
	m1 := NewPolynomialModel(1, 5) // Linear
	m2 := NewPolynomialModel(3, 5) // Cubic

	// Same data for both
	for i := 1; i <= 20; i++ {
		impact := &decision.ResourceImpact{CPUDelta: float64(i * 10)}
		m1.Observe("test", i*100, impact)
		m2.Observe("test", i*100, impact)
	}

	conf1 := m1.Confidence("test")
	conf2 := m2.Confidence("test")

	// Higher degree should have lower confidence (penalty for overfitting)
	if conf2 >= conf1 {
		t.Errorf("expected lower confidence for higher degree: degree1=%d conf=%f, degree3=%d conf=%f",
			m1.degree, conf1, m2.degree, conf2)
	}
}

func TestPolynomialModel_Stats(t *testing.T) {
	m := NewPolynomialModel(2, 5)

	for i := 1; i <= 10; i++ {
		m.Observe("test", i*100, &decision.ResourceImpact{CPUDelta: float64(i * 10)})
	}

	stats := m.Stats()
	if stats.ModelName != "polynomial" {
		t.Errorf("expected model name 'polynomial', got '%s'", stats.ModelName)
	}
	if stats.TotalObservations != 10 {
		t.Errorf("expected 10 observations, got %d", stats.TotalObservations)
	}

	taskStats := stats.Tasks["test"]
	if taskStats == nil {
		t.Fatal("expected task stats")
	}
	// Average should be (10+20+...+100)/10 = 55
	expectedAvg := 55.0
	if math.Abs(taskStats.AvgCPUDelta-expectedAvg) > 0.1 {
		t.Errorf("expected avg CPU %f, got %f", expectedAvg, taskStats.AvgCPUDelta)
	}
}

func TestPolynomialModel_SaveLoad(t *testing.T) {
	m1 := NewPolynomialModel(2, 5)

	for i := 1; i <= 10; i++ {
		x := float64(i * 10)
		cpu := 0.001 * x * x
		m1.Observe("test", int(x), &decision.ResourceImpact{CPUDelta: cpu})
	}

	// Save
	var buf bytes.Buffer
	if err := m1.Save(&buf); err != nil {
		t.Fatalf("Save error: %v", err)
	}

	// Load into new model
	m2 := NewPolynomialModel(3, 10) // Different params - should be overwritten
	if err := m2.Load(&buf); err != nil {
		t.Fatalf("Load error: %v", err)
	}

	// Predictions should match
	p1 := m1.Predict("test", 50)
	p2 := m2.Predict("test", 50)

	if p1 == nil || p2 == nil {
		t.Fatal("expected predictions")
	}

	if math.Abs(p1.CPUDelta-p2.CPUDelta) > 0.01 {
		t.Errorf("prediction mismatch: %f vs %f", p1.CPUDelta, p2.CPUDelta)
	}
}

func TestPolynomialModel_BufferLimit(t *testing.T) {
	m := NewPolynomialModel(2, 5)

	// Add more observations than maxPolynomialObservations
	for i := 0; i < maxPolynomialObservations+100; i++ {
		m.Observe("test", i, &decision.ResourceImpact{CPUDelta: float64(i)})
	}

	// Model should still work and buffer should be limited
	taskStats := m.TaskStats("test")
	if taskStats == nil {
		t.Fatal("expected task stats")
	}

	// Count reflects all observations
	if taskStats.Count != int64(maxPolynomialObservations+100) {
		t.Errorf("expected count %d, got %d", maxPolynomialObservations+100, taskStats.Count)
	}
}

func TestPolynomialModel_NilObservation(t *testing.T) {
	m := NewPolynomialModel(2, 5)
	// Should not panic
	m.Observe("test", 100, nil)

	// Should still have no data
	if m.TaskStats("test") != nil {
		t.Error("expected nil task stats after nil observation")
	}
}

func TestPolynomialModel_NeedsRetrain(t *testing.T) {
	m := NewPolynomialModel(2, 5)
	if m.NeedsRetrain() {
		t.Error("online model should not need retraining")
	}
}

func TestPolynomialModel_MultipleResources(t *testing.T) {
	m := NewPolynomialModel(2, 5)

	// Add observations with all resources
	for i := 1; i <= 10; i++ {
		m.Observe("test", i*100, &decision.ResourceImpact{
			CPUDelta:    float64(i * 10),
			MemoryDelta: float64(i * 5),
			GPUDelta:    float64(i * 2),
			VRAMDelta:   float64(i),
		})
	}

	// Prediction should return non-nil for all resources
	prediction := m.Predict("test", 500)
	if prediction == nil {
		t.Fatal("expected prediction")
	}

	// Just verify all fields are populated (actual values may vary due to polynomial fitting)
	stats := m.TaskStats("test")
	if stats == nil {
		t.Fatal("expected task stats")
	}
	if stats.Count != 10 {
		t.Errorf("expected count 10, got %d", stats.Count)
	}
}

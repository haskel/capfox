package model

import (
	"bytes"
	"math"
	"testing"

	"github.com/haskel/capfox/internal/decision"
)

func TestLinearModel_Name(t *testing.T) {
	m := NewLinearModel(5)
	if m.Name() != "linear" {
		t.Errorf("expected name 'linear', got '%s'", m.Name())
	}
}

func TestLinearModel_LearningType(t *testing.T) {
	m := NewLinearModel(5)
	if m.LearningType() != LearningTypeOnline {
		t.Error("expected online learning type")
	}
}

func TestLinearModel_MinObservationsDefault(t *testing.T) {
	m := NewLinearModel(0)
	if m.minObservations != 2 {
		t.Errorf("expected minObservations 2 for invalid input, got %d", m.minObservations)
	}
}

func TestLinearModel_PredictNoData(t *testing.T) {
	m := NewLinearModel(5)
	prediction := m.Predict("unknown_task", 100)
	if prediction != nil {
		t.Error("expected nil prediction for unknown task")
	}
}

func TestLinearModel_PredictInsufficientData(t *testing.T) {
	m := NewLinearModel(5)

	// Add only 3 observations (less than minObservations)
	for i := 0; i < 3; i++ {
		m.Observe("test", i*100, &decision.ResourceImpact{CPUDelta: float64(i * 10)})
	}

	prediction := m.Predict("test", 100)
	if prediction != nil {
		t.Error("expected nil prediction with insufficient data")
	}
}

func TestLinearModel_LinearPrediction(t *testing.T) {
	m := NewLinearModel(5)

	// Perfect linear relationship: CPU = 0.1 * complexity
	// Data points: (100, 10), (200, 20), (300, 30), (400, 40), (500, 50)
	for i := 1; i <= 5; i++ {
		complexity := i * 100
		cpu := float64(i * 10)
		m.Observe("test", complexity, &decision.ResourceImpact{CPUDelta: cpu})
	}

	// Predict for complexity 600 - should be approximately 60
	prediction := m.Predict("test", 600)
	if prediction == nil {
		t.Fatal("expected prediction")
	}

	expected := 60.0
	if math.Abs(prediction.CPUDelta-expected) > 0.5 {
		t.Errorf("expected CPU %f, got %f", expected, prediction.CPUDelta)
	}
}

func TestLinearModel_WithIntercept(t *testing.T) {
	m := NewLinearModel(5)

	// Linear relationship with intercept: CPU = 5 + 0.1 * complexity
	// Data points: (100, 15), (200, 25), (300, 35), (400, 45), (500, 55)
	for i := 1; i <= 5; i++ {
		complexity := i * 100
		cpu := 5 + float64(i*10)
		m.Observe("test", complexity, &decision.ResourceImpact{CPUDelta: cpu})
	}

	// Predict for complexity 600 - should be approximately 5 + 60 = 65
	prediction := m.Predict("test", 600)
	if prediction == nil {
		t.Fatal("expected prediction")
	}

	expected := 65.0
	if math.Abs(prediction.CPUDelta-expected) > 0.5 {
		t.Errorf("expected CPU %f, got %f", expected, prediction.CPUDelta)
	}
}

func TestLinearModel_ConstantX(t *testing.T) {
	m := NewLinearModel(5)

	// All observations have the same complexity - should use mean
	for i := 0; i < 5; i++ {
		m.Observe("test", 100, &decision.ResourceImpact{CPUDelta: float64(10 + i)})
	}

	prediction := m.Predict("test", 100)
	if prediction == nil {
		t.Fatal("expected prediction")
	}

	// Mean of 10, 11, 12, 13, 14 = 12
	expected := 12.0
	if math.Abs(prediction.CPUDelta-expected) > 0.5 {
		t.Errorf("expected CPU %f, got %f", expected, prediction.CPUDelta)
	}
}

func TestLinearModel_ConfidenceGrows(t *testing.T) {
	m := NewLinearModel(5)

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
	for i := 3; i < 10; i++ {
		m.Observe("test", i*100, &decision.ResourceImpact{CPUDelta: float64(i * 10)})
	}
	conf := m.Confidence("test")
	if conf <= 0.5 {
		t.Errorf("expected confidence > 0.5 after 10 observations, got %f", conf)
	}
}

func TestLinearModel_MultipleResources(t *testing.T) {
	m := NewLinearModel(5)

	// Different linear relationships for each resource
	for i := 1; i <= 10; i++ {
		complexity := i * 100
		m.Observe("test", complexity, &decision.ResourceImpact{
			CPUDelta:    float64(i * 10),                // CPU = 0.1 * x
			MemoryDelta: float64(5 + i*5),               // Mem = 5 + 0.05 * x
			GPUDelta:    float64(i * 2),                 // GPU = 0.02 * x
			VRAMDelta:   float64(10 + i),                // VRAM = 10 + 0.01 * x
		})
	}

	prediction := m.Predict("test", 500)
	if prediction == nil {
		t.Fatal("expected prediction")
	}

	// Check predictions
	if math.Abs(prediction.CPUDelta-50.0) > 1.0 {
		t.Errorf("expected CPU ~50, got %f", prediction.CPUDelta)
	}
	if math.Abs(prediction.MemoryDelta-30.0) > 1.0 {
		t.Errorf("expected Memory ~30, got %f", prediction.MemoryDelta)
	}
	if math.Abs(prediction.GPUDelta-10.0) > 1.0 {
		t.Errorf("expected GPU ~10, got %f", prediction.GPUDelta)
	}
}

func TestLinearModel_Stats(t *testing.T) {
	m := NewLinearModel(5)

	for i := 1; i <= 10; i++ {
		m.Observe("test", i*100, &decision.ResourceImpact{CPUDelta: float64(i * 10)})
	}

	stats := m.Stats()
	if stats.ModelName != "linear" {
		t.Errorf("expected model name 'linear', got '%s'", stats.ModelName)
	}
	if stats.TotalObservations != 10 {
		t.Errorf("expected 10 observations, got %d", stats.TotalObservations)
	}

	taskStats := stats.Tasks["test"]
	if taskStats == nil {
		t.Fatal("expected task stats")
	}
	if taskStats.Coefficients == nil {
		t.Error("expected coefficients for task with enough observations")
	}
}

func TestLinearModel_TaskStatsWithCoefficients(t *testing.T) {
	m := NewLinearModel(5)

	// Perfect linear: CPU = 0.1 * x
	for i := 1; i <= 5; i++ {
		m.Observe("test", i*100, &decision.ResourceImpact{CPUDelta: float64(i * 10)})
	}

	taskStats := m.TaskStats("test")
	if taskStats == nil {
		t.Fatal("expected task stats")
	}

	coefs := taskStats.Coefficients
	if coefs == nil {
		t.Fatal("expected coefficients")
	}

	// Coefficient A should be approximately 0.1
	if math.Abs(coefs.CPUA-0.1) > 0.01 {
		t.Errorf("expected CPU coefficient A ~0.1, got %f", coefs.CPUA)
	}

	// Coefficient B should be approximately 0
	if math.Abs(coefs.CPUB) > 0.5 {
		t.Errorf("expected CPU coefficient B ~0, got %f", coefs.CPUB)
	}
}

func TestLinearModel_SaveLoad(t *testing.T) {
	m1 := NewLinearModel(5)

	for i := 1; i <= 10; i++ {
		m1.Observe("test", i*100, &decision.ResourceImpact{CPUDelta: float64(i * 10)})
	}

	// Save
	var buf bytes.Buffer
	if err := m1.Save(&buf); err != nil {
		t.Fatalf("Save error: %v", err)
	}

	// Load into new model
	m2 := NewLinearModel(10) // Different minObs - should be overwritten
	if err := m2.Load(&buf); err != nil {
		t.Fatalf("Load error: %v", err)
	}

	// Predictions should match
	p1 := m1.Predict("test", 600)
	p2 := m2.Predict("test", 600)

	if p1 == nil || p2 == nil {
		t.Fatal("expected predictions")
	}

	if math.Abs(p1.CPUDelta-p2.CPUDelta) > 0.001 {
		t.Errorf("prediction mismatch: %f vs %f", p1.CPUDelta, p2.CPUDelta)
	}
}

func TestLinearModel_NilObservation(t *testing.T) {
	m := NewLinearModel(5)
	// Should not panic
	m.Observe("test", 100, nil)

	// Should still have no data
	if m.TaskStats("test") != nil {
		t.Error("expected nil task stats after nil observation")
	}
}

func TestLinearModel_NeedsRetrain(t *testing.T) {
	m := NewLinearModel(5)
	if m.NeedsRetrain() {
		t.Error("online model should not need retraining")
	}
}

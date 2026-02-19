package model

import (
	"bytes"
	"math"
	"testing"
	"time"

	"github.com/haskel/capfox/internal/decision"
)

func TestGradientBoostModel_Name(t *testing.T) {
	cfg := GradientBoostConfig{MinObservations: 10}
	m := NewGradientBoostModel(cfg)
	if m.Name() != "gradient_boosting" {
		t.Errorf("expected name 'gradient_boosting', got '%s'", m.Name())
	}
}

func TestGradientBoostModel_LearningType(t *testing.T) {
	cfg := GradientBoostConfig{MinObservations: 10}
	m := NewGradientBoostModel(cfg)
	if m.LearningType() != LearningTypeBatch {
		t.Error("expected batch learning type")
	}
}

func TestGradientBoostModel_ConfigDefaults(t *testing.T) {
	cfg := GradientBoostConfig{} // All zeros
	m := NewGradientBoostModel(cfg)

	if m.config.MinObservations != 10 {
		t.Errorf("expected MinObservations 10, got %d", m.config.MinObservations)
	}
	if m.config.MaxBufferSize != 100 {
		t.Errorf("expected MaxBufferSize 100, got %d", m.config.MaxBufferSize)
	}
	if m.config.RetrainInterval != time.Hour {
		t.Errorf("expected RetrainInterval 1h, got %v", m.config.RetrainInterval)
	}
}

func TestGradientBoostModel_PredictNoData(t *testing.T) {
	cfg := GradientBoostConfig{MinObservations: 5}
	m := NewGradientBoostModel(cfg)

	prediction := m.Predict("unknown_task", 100)
	if prediction != nil {
		t.Error("expected nil prediction for unknown task")
	}
}

func TestGradientBoostModel_PredictBeforeRetrain(t *testing.T) {
	cfg := GradientBoostConfig{MinObservations: 5}
	m := NewGradientBoostModel(cfg)

	// Add observations but don't retrain
	for i := 0; i < 10; i++ {
		m.Observe("test", i*100, &decision.ResourceImpact{CPUDelta: float64(i * 10)})
	}

	// Prediction should be nil (no coefficients yet)
	prediction := m.Predict("test", 100)
	if prediction != nil {
		t.Error("expected nil prediction before retrain")
	}
}

func TestGradientBoostModel_PredictAfterRetrain(t *testing.T) {
	cfg := GradientBoostConfig{MinObservations: 5}
	m := NewGradientBoostModel(cfg)

	// Linear data: CPU = 0.1 * x
	for i := 1; i <= 10; i++ {
		m.Observe("test", i*100, &decision.ResourceImpact{CPUDelta: float64(i * 10)})
	}

	// Retrain
	if err := m.Retrain(); err != nil {
		t.Fatalf("Retrain error: %v", err)
	}

	// Now prediction should work
	prediction := m.Predict("test", 500)
	if prediction == nil {
		t.Fatal("expected prediction after retrain")
	}

	// Should be approximately 50
	expected := 50.0
	if math.Abs(prediction.CPUDelta-expected) > 5.0 {
		t.Errorf("expected CPU ~%f, got %f", expected, prediction.CPUDelta)
	}
}

func TestGradientBoostModel_NeedsRetrain(t *testing.T) {
	cfg := GradientBoostConfig{
		MinObservations: 5,
		RetrainInterval: 100 * time.Millisecond,
	}
	m := NewGradientBoostModel(cfg)

	// No data - should not need retrain
	if m.NeedsRetrain() {
		t.Error("expected no retrain needed with no data")
	}

	// Add some observations
	for i := 0; i < 10; i++ {
		m.Observe("test", i*100, &decision.ResourceImpact{CPUDelta: float64(i * 10)})
	}

	// Just added data, but retrain interval hasn't passed
	if m.NeedsRetrain() {
		t.Error("expected no retrain needed before interval")
	}

	// Wait for interval
	time.Sleep(150 * time.Millisecond)

	// Now should need retrain
	if !m.NeedsRetrain() {
		t.Error("expected retrain needed after interval")
	}

	// After retrain, should not need retrain
	_ = m.Retrain()
	if m.NeedsRetrain() {
		t.Error("expected no retrain needed after retrain")
	}
}

func TestGradientBoostModel_ConfidenceBeforeRetrain(t *testing.T) {
	cfg := GradientBoostConfig{MinObservations: 5}
	m := NewGradientBoostModel(cfg)

	// Add observations
	for i := 0; i < 10; i++ {
		m.Observe("test", i*100, &decision.ResourceImpact{CPUDelta: float64(i * 10)})
	}

	// Confidence should be 0 before retrain
	if m.Confidence("test") != 0 {
		t.Error("expected confidence 0 before retrain")
	}
}

func TestGradientBoostModel_ConfidenceAfterRetrain(t *testing.T) {
	cfg := GradientBoostConfig{MinObservations: 5}
	m := NewGradientBoostModel(cfg)

	for i := 1; i <= 20; i++ {
		m.Observe("test", i*100, &decision.ResourceImpact{CPUDelta: float64(i * 10)})
	}

	_ = m.Retrain()

	conf := m.Confidence("test")
	if conf <= 0 {
		t.Errorf("expected positive confidence after retrain, got %f", conf)
	}
}

func TestGradientBoostModel_Stats(t *testing.T) {
	cfg := GradientBoostConfig{MinObservations: 5}
	m := NewGradientBoostModel(cfg)

	for i := 1; i <= 10; i++ {
		m.Observe("test", i*100, &decision.ResourceImpact{CPUDelta: float64(i * 10)})
	}

	stats := m.Stats()
	if stats.ModelName != "gradient_boosting" {
		t.Errorf("expected model name 'gradient_boosting', got '%s'", stats.ModelName)
	}
	if stats.LearningType != "batch" {
		t.Errorf("expected learning type 'batch', got '%s'", stats.LearningType)
	}
	if stats.TotalObservations != 10 {
		t.Errorf("expected 10 observations, got %d", stats.TotalObservations)
	}
}

func TestGradientBoostModel_BufferLimit(t *testing.T) {
	cfg := GradientBoostConfig{
		MinObservations: 5,
		MaxBufferSize:   50,
	}
	m := NewGradientBoostModel(cfg)

	// Add more observations than buffer size
	for i := 0; i < 100; i++ {
		m.Observe("test", i, &decision.ResourceImpact{CPUDelta: float64(i)})
	}

	// Count should be 100
	taskStats := m.TaskStats("test")
	if taskStats.Count != 100 {
		t.Errorf("expected count 100, got %d", taskStats.Count)
	}

	// But internal buffer should be limited (checked implicitly by successful operations)
	_ = m.Retrain()
	if m.Predict("test", 50) == nil {
		t.Error("expected prediction after retrain")
	}
}

func TestGradientBoostModel_SaveLoad(t *testing.T) {
	cfg := GradientBoostConfig{MinObservations: 5}
	m1 := NewGradientBoostModel(cfg)

	for i := 1; i <= 10; i++ {
		m1.Observe("test", i*100, &decision.ResourceImpact{CPUDelta: float64(i * 10)})
	}
	_ = m1.Retrain()

	// Save
	var buf bytes.Buffer
	if err := m1.Save(&buf); err != nil {
		t.Fatalf("Save error: %v", err)
	}

	// Load into new model
	m2 := NewGradientBoostModel(GradientBoostConfig{})
	if err := m2.Load(&buf); err != nil {
		t.Fatalf("Load error: %v", err)
	}

	// Predictions should match
	p1 := m1.Predict("test", 500)
	p2 := m2.Predict("test", 500)

	if p1 == nil || p2 == nil {
		t.Fatal("expected predictions")
	}

	if math.Abs(p1.CPUDelta-p2.CPUDelta) > 0.001 {
		t.Errorf("prediction mismatch: %f vs %f", p1.CPUDelta, p2.CPUDelta)
	}
}

func TestGradientBoostModel_NilObservation(t *testing.T) {
	cfg := GradientBoostConfig{MinObservations: 5}
	m := NewGradientBoostModel(cfg)

	// Should not panic
	m.Observe("test", 100, nil)

	// Should have no data
	if m.TaskStats("test") != nil {
		t.Error("expected nil task stats after nil observation")
	}
}

func TestGradientBoostModel_ConstantData(t *testing.T) {
	cfg := GradientBoostConfig{MinObservations: 5}
	m := NewGradientBoostModel(cfg)

	// All same complexity - should fallback to mean
	for i := 0; i < 10; i++ {
		m.Observe("test", 100, &decision.ResourceImpact{CPUDelta: float64(10 + i)})
	}

	m.Retrain()

	prediction := m.Predict("test", 100)
	if prediction == nil {
		t.Fatal("expected prediction")
	}

	// Mean of 10, 11, ..., 19 = 14.5
	expected := 14.5
	if math.Abs(prediction.CPUDelta-expected) > 1.0 {
		t.Errorf("expected CPU ~%f, got %f", expected, prediction.CPUDelta)
	}
}

func TestGradientBoostModel_MultipleResources(t *testing.T) {
	cfg := GradientBoostConfig{MinObservations: 5}
	m := NewGradientBoostModel(cfg)

	for i := 1; i <= 10; i++ {
		m.Observe("test", i*100, &decision.ResourceImpact{
			CPUDelta:    float64(i * 10),
			MemoryDelta: float64(i * 5),
			GPUDelta:    float64(i * 2),
			VRAMDelta:   float64(i),
		})
	}

	m.Retrain()

	prediction := m.Predict("test", 500)
	if prediction == nil {
		t.Fatal("expected prediction")
	}

	// Check all resources predicted
	if math.Abs(prediction.CPUDelta-50.0) > 5.0 {
		t.Errorf("expected CPU ~50, got %f", prediction.CPUDelta)
	}
	if math.Abs(prediction.MemoryDelta-25.0) > 5.0 {
		t.Errorf("expected Memory ~25, got %f", prediction.MemoryDelta)
	}
	if math.Abs(prediction.GPUDelta-10.0) > 3.0 {
		t.Errorf("expected GPU ~10, got %f", prediction.GPUDelta)
	}
}

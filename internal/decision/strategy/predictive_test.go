package strategy

import (
	"io"
	"testing"

	"github.com/haskel/capfox/internal/decision"
	"github.com/haskel/capfox/internal/decision/model"
	"github.com/haskel/capfox/internal/monitor"
)

// mockModel implements model.PredictionModel for testing
type mockModel struct {
	name       string
	prediction *decision.ResourceImpact
	confidence float64
}

func newMockModel(name string, prediction *decision.ResourceImpact, confidence float64) *mockModel {
	return &mockModel{
		name:       name,
		prediction: prediction,
		confidence: confidence,
	}
}

func (m *mockModel) Name() string                                 { return m.name }
func (m *mockModel) LearningType() model.LearningType             { return model.LearningTypeOnline }
func (m *mockModel) Predict(task string, complexity int) *decision.ResourceImpact { return m.prediction }
func (m *mockModel) Observe(task string, complexity int, impact *decision.ResourceImpact) {}
func (m *mockModel) Confidence(task string) float64               { return m.confidence }
func (m *mockModel) Stats() *model.Stats                          { return &model.Stats{ModelName: m.name} }
func (m *mockModel) TaskStats(task string) *model.TaskStats       { return nil }
func (m *mockModel) NeedsRetrain() bool                           { return false }
func (m *mockModel) Retrain() error                               { return nil }
func (m *mockModel) Save(w io.Writer) error                       { return nil }
func (m *mockModel) Load(r io.Reader) error                       { return nil }

func TestPredictiveStrategy_Name(t *testing.T) {
	m := newMockModel("test", nil, 0)
	s := NewPredictiveStrategy(m, 5, nil)
	if s.Name() != "predictive" {
		t.Errorf("expected name 'predictive', got '%s'", s.Name())
	}
}

func TestPredictiveStrategy_Decide_NilContext(t *testing.T) {
	m := newMockModel("test", nil, 0)
	s := NewPredictiveStrategy(m, 5, nil)
	result := s.Decide(nil)

	if !result.Allowed {
		t.Error("expected allowed=true for nil context")
	}
}

func TestPredictiveStrategy_Decide_FallbackOnNoData(t *testing.T) {
	m := newMockModel("test", nil, 0) // Zero confidence
	s := NewPredictiveStrategy(m, 5, nil)

	ctx := decision.NewContext("test", 100).
		WithCurrentState(&monitor.SystemState{
			CPU:    monitor.CPUState{UsagePercent: 50.0},
			Memory: monitor.MemoryState{UsagePercent: 40.0},
		}).
		WithThresholds(&decision.ThresholdsConfig{
			CPU:     decision.CPUThreshold{MaxPercent: 80.0},
			Memory:  decision.MemoryThreshold{MaxPercent: 80.0},
			Storage: decision.StorageThreshold{MinFreeGB: 0},
		})

	result := s.Decide(ctx)

	// Should fallback to threshold strategy
	if !result.Allowed {
		t.Error("expected allowed=true (fallback to threshold)")
	}
	if !containsReason(result.Reasons, decision.ReasonInsufficientData) {
		t.Error("expected ReasonInsufficientData in reasons")
	}
}

func TestPredictiveStrategy_Decide_AllowsWhenPredictedBelowThreshold(t *testing.T) {
	prediction := &decision.ResourceImpact{
		CPUDelta:    10.0, // 50 + 10 = 60% < 80%
		MemoryDelta: 10.0, // 40 + 10 = 50% < 80%
	}
	m := newMockModel("test", prediction, 0.9)
	s := NewPredictiveStrategy(m, 5, nil)

	ctx := decision.NewContext("test", 100).
		WithCurrentState(&monitor.SystemState{
			CPU:    monitor.CPUState{UsagePercent: 50.0},
			Memory: monitor.MemoryState{UsagePercent: 40.0},
		}).
		WithThresholds(&decision.ThresholdsConfig{
			CPU:     decision.CPUThreshold{MaxPercent: 80.0},
			Memory:  decision.MemoryThreshold{MaxPercent: 80.0},
			Storage: decision.StorageThreshold{MinFreeGB: 0},
		}).
		WithPrediction(prediction)

	result := s.Decide(ctx)

	if !result.Allowed {
		t.Errorf("expected allowed=true, got reasons: %v", result.Reasons)
	}
	if result.PredictedState == nil {
		t.Error("expected predicted state")
	}
	if result.PredictedState.CPUPercent != 60.0 {
		t.Errorf("expected CPU 60%%, got %f%%", result.PredictedState.CPUPercent)
	}
	if result.Confidence != 0.9 {
		t.Errorf("expected confidence 0.9, got %f", result.Confidence)
	}
}

func TestPredictiveStrategy_Decide_RejectsWhenPredictedExceedsThreshold(t *testing.T) {
	prediction := &decision.ResourceImpact{
		CPUDelta:    35.0, // 50 + 35 = 85% > 80%
		MemoryDelta: 10.0,
	}
	m := newMockModel("test", prediction, 0.9)
	s := NewPredictiveStrategy(m, 5, nil)

	ctx := decision.NewContext("test", 100).
		WithCurrentState(&monitor.SystemState{
			CPU:    monitor.CPUState{UsagePercent: 50.0},
			Memory: monitor.MemoryState{UsagePercent: 40.0},
		}).
		WithThresholds(&decision.ThresholdsConfig{
			CPU:     decision.CPUThreshold{MaxPercent: 80.0},
			Memory:  decision.MemoryThreshold{MaxPercent: 80.0},
			Storage: decision.StorageThreshold{MinFreeGB: 0},
		}).
		WithPrediction(prediction)

	result := s.Decide(ctx)

	if result.Allowed {
		t.Error("expected allowed=false for predicted CPU overload")
	}
	if !containsReason(result.Reasons, decision.ReasonCPUOverload) {
		t.Error("expected ReasonCPUOverload in reasons")
	}
	if result.PredictedState == nil {
		t.Error("expected predicted state")
	}
	if result.PredictedState.CPUPercent != 85.0 {
		t.Errorf("expected CPU 85%%, got %f%%", result.PredictedState.CPUPercent)
	}
}

func TestPredictiveStrategy_Decide_GPUPrediction(t *testing.T) {
	prediction := &decision.ResourceImpact{
		CPUDelta:    5.0,
		MemoryDelta: 5.0,
		GPUDelta:    40.0, // 50 + 40 = 90% > 80%
		VRAMDelta:   10.0,
	}
	m := newMockModel("test", prediction, 0.85)
	s := NewPredictiveStrategy(m, 5, nil)

	ctx := decision.NewContext("test", 100).
		WithCurrentState(&monitor.SystemState{
			CPU:    monitor.CPUState{UsagePercent: 50.0},
			Memory: monitor.MemoryState{UsagePercent: 40.0},
			GPUs: []monitor.GPUState{
				{UsagePercent: 50.0, VRAMUsedBytes: 5000, VRAMTotalBytes: 10000},
			},
		}).
		WithThresholds(&decision.ThresholdsConfig{
			CPU:     decision.CPUThreshold{MaxPercent: 80.0},
			Memory:  decision.MemoryThreshold{MaxPercent: 80.0},
			GPU:     decision.GPUThreshold{MaxPercent: 80.0},
			VRAM:    decision.VRAMThreshold{MaxPercent: 80.0},
			Storage: decision.StorageThreshold{MinFreeGB: 0},
		}).
		WithPrediction(prediction)

	result := s.Decide(ctx)

	if result.Allowed {
		t.Error("expected allowed=false for predicted GPU overload")
	}
	if !containsReason(result.Reasons, decision.ReasonGPUOverload) {
		t.Error("expected ReasonGPUOverload in reasons")
	}
	if result.PredictedState.GPUPercent != 90.0 {
		t.Errorf("expected GPU 90%%, got %f%%", result.PredictedState.GPUPercent)
	}
}

func TestPredictiveStrategy_Decide_ClampsPrediction(t *testing.T) {
	prediction := &decision.ResourceImpact{
		CPUDelta:    60.0, // 90 + 60 = 150% should be clamped to 100%
		MemoryDelta: 5.0,
	}
	m := newMockModel("test", prediction, 0.9)
	s := NewPredictiveStrategy(m, 5, nil)

	ctx := decision.NewContext("test", 100).
		WithCurrentState(&monitor.SystemState{
			CPU:    monitor.CPUState{UsagePercent: 90.0},
			Memory: monitor.MemoryState{UsagePercent: 40.0},
		}).
		WithThresholds(&decision.ThresholdsConfig{
			CPU:     decision.CPUThreshold{MaxPercent: 80.0},
			Memory:  decision.MemoryThreshold{MaxPercent: 80.0},
			Storage: decision.StorageThreshold{MinFreeGB: 0},
		}).
		WithPrediction(prediction)

	result := s.Decide(ctx)

	if result.PredictedState.CPUPercent != 100.0 {
		t.Errorf("expected CPU clamped to 100%%, got %f%%", result.PredictedState.CPUPercent)
	}
}

package strategy

import (
	"testing"

	"github.com/haskel/capfox/internal/decision"
	"github.com/haskel/capfox/internal/monitor"
)

func TestConservativeStrategy_Name(t *testing.T) {
	m := newMockModel("test", nil, 0)
	s := NewConservativeStrategy(m, 0.1, 5, nil)
	if s.Name() != "conservative" {
		t.Errorf("expected name 'conservative', got '%s'", s.Name())
	}
}

func TestConservativeStrategy_Decide_NilContext(t *testing.T) {
	m := newMockModel("test", nil, 0)
	s := NewConservativeStrategy(m, 0.1, 5, nil)
	result := s.Decide(nil)

	if !result.Allowed {
		t.Error("expected allowed=true for nil context")
	}
}

func TestConservativeStrategy_Decide_FallbackOnNoData(t *testing.T) {
	m := newMockModel("test", nil, 0) // Zero confidence
	s := NewConservativeStrategy(m, 0.1, 5, nil)

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

	if !result.Allowed {
		t.Error("expected allowed=true (fallback to threshold)")
	}
	if !containsReason(result.Reasons, decision.ReasonInsufficientData) {
		t.Error("expected ReasonInsufficientData in reasons")
	}
}

func TestConservativeStrategy_Decide_AppliesBuffer(t *testing.T) {
	// Without buffer: 50 + 20 = 70% < 80% (allowed)
	// With 20% buffer: 50 + (20 * 1.2) = 50 + 24 = 74% < 80% (still allowed)
	prediction := &decision.ResourceImpact{
		CPUDelta:    20.0,
		MemoryDelta: 10.0,
	}
	m := newMockModel("test", prediction, 0.9)
	s := NewConservativeStrategy(m, 0.2, 5, nil) // 20% buffer

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
		t.Error("expected allowed=true with buffered prediction")
	}
	// 50 + (20 * 1.2) = 74%
	if result.PredictedState.CPUPercent != 74.0 {
		t.Errorf("expected CPU 74%%, got %f%%", result.PredictedState.CPUPercent)
	}
}

func TestConservativeStrategy_Decide_MoreStrictThanPredictive(t *testing.T) {
	// Predictive: 60 + 20 = 80% <= 80% (allowed)
	// Conservative with 10% buffer: 60 + (20 * 1.1) = 60 + 22 = 82% > 80% (rejected)
	prediction := &decision.ResourceImpact{
		CPUDelta:    20.0,
		MemoryDelta: 10.0,
	}
	mPredictive := newMockModel("test", prediction, 0.9)
	mConservative := newMockModel("test", prediction, 0.9)

	predictive := NewPredictiveStrategy(mPredictive, 5, nil)
	conservative := NewConservativeStrategy(mConservative, 0.1, 5, nil) // 10% buffer

	ctx := decision.NewContext("test", 100).
		WithCurrentState(&monitor.SystemState{
			CPU:    monitor.CPUState{UsagePercent: 60.0},
			Memory: monitor.MemoryState{UsagePercent: 40.0},
		}).
		WithThresholds(&decision.ThresholdsConfig{
			CPU:     decision.CPUThreshold{MaxPercent: 80.0},
			Memory:  decision.MemoryThreshold{MaxPercent: 80.0},
			Storage: decision.StorageThreshold{MinFreeGB: 0},
		}).
		WithPrediction(prediction)

	predictiveResult := predictive.Decide(ctx)
	conservativeResult := conservative.Decide(ctx)

	if !predictiveResult.Allowed {
		t.Error("expected predictive to allow")
	}
	if conservativeResult.Allowed {
		t.Error("expected conservative to reject (more strict)")
	}

	// Verify predicted states
	// Predictive: 60 + 20 = 80
	if predictiveResult.PredictedState.CPUPercent != 80.0 {
		t.Errorf("predictive: expected CPU 80%%, got %f%%", predictiveResult.PredictedState.CPUPercent)
	}
	// Conservative: 60 + 22 = 82
	if conservativeResult.PredictedState.CPUPercent != 82.0 {
		t.Errorf("conservative: expected CPU 82%%, got %f%%", conservativeResult.PredictedState.CPUPercent)
	}
}

func TestConservativeStrategy_Decide_BufferAppliedToAllResources(t *testing.T) {
	prediction := &decision.ResourceImpact{
		CPUDelta:    10.0,
		MemoryDelta: 10.0,
		GPUDelta:    10.0,
		VRAMDelta:   10.0,
	}
	m := newMockModel("test", prediction, 0.9)
	s := NewConservativeStrategy(m, 0.5, 5, nil) // 50% buffer

	ctx := decision.NewContext("test", 100).
		WithCurrentState(&monitor.SystemState{
			CPU:    monitor.CPUState{UsagePercent: 50.0},
			Memory: monitor.MemoryState{UsagePercent: 50.0},
			GPUs: []monitor.GPUState{
				{UsagePercent: 50.0, VRAMUsedBytes: 5000, VRAMTotalBytes: 10000}, // 50% VRAM
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

	// All should be: 50 + (10 * 1.5) = 50 + 15 = 65%
	expected := 65.0
	if result.PredictedState.CPUPercent != expected {
		t.Errorf("expected CPU %f%%, got %f%%", expected, result.PredictedState.CPUPercent)
	}
	if result.PredictedState.MemoryPercent != expected {
		t.Errorf("expected Memory %f%%, got %f%%", expected, result.PredictedState.MemoryPercent)
	}
	if result.PredictedState.GPUPercent != expected {
		t.Errorf("expected GPU %f%%, got %f%%", expected, result.PredictedState.GPUPercent)
	}
	if result.PredictedState.VRAMPercent != expected {
		t.Errorf("expected VRAM %f%%, got %f%%", expected, result.PredictedState.VRAMPercent)
	}
}

func TestConservativeStrategy_Decide_RejectsWithBuffer(t *testing.T) {
	// Without buffer: 70 + 10 = 80% <= 80% (would be allowed)
	// With 10% buffer: 70 + (10 * 1.1) = 70 + 11 = 81% > 80% (rejected)
	prediction := &decision.ResourceImpact{
		CPUDelta:    10.0,
		MemoryDelta: 5.0,
	}
	m := newMockModel("test", prediction, 0.9)
	s := NewConservativeStrategy(m, 0.1, 5, nil) // 10% buffer

	ctx := decision.NewContext("test", 100).
		WithCurrentState(&monitor.SystemState{
			CPU:    monitor.CPUState{UsagePercent: 70.0},
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
		t.Error("expected allowed=false with buffer applied")
	}
	if !containsReason(result.Reasons, decision.ReasonCPUOverload) {
		t.Error("expected ReasonCPUOverload in reasons")
	}
	// 70 + 11 = 81
	if result.PredictedState.CPUPercent != 81.0 {
		t.Errorf("expected CPU 81%%, got %f%%", result.PredictedState.CPUPercent)
	}
}

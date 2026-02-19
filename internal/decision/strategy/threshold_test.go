package strategy

import (
	"testing"

	"github.com/haskel/capfox/internal/decision"
	"github.com/haskel/capfox/internal/monitor"
)

func TestThresholdStrategy_Name(t *testing.T) {
	s := NewThresholdStrategy()
	if s.Name() != "threshold" {
		t.Errorf("expected name 'threshold', got '%s'", s.Name())
	}
}

func TestThresholdStrategy_Decide_NilContext(t *testing.T) {
	s := NewThresholdStrategy()
	result := s.Decide(nil)

	if !result.Allowed {
		t.Error("expected allowed=true for nil context")
	}
	if result.Strategy != "threshold" {
		t.Errorf("expected strategy 'threshold', got '%s'", result.Strategy)
	}
}

func TestThresholdStrategy_Decide_NilState(t *testing.T) {
	s := NewThresholdStrategy()
	ctx := decision.NewContext("test", 100)
	result := s.Decide(ctx)

	if !result.Allowed {
		t.Error("expected allowed=true for nil state")
	}
}

func TestThresholdStrategy_Decide_AllowsBelowThresholds(t *testing.T) {
	s := NewThresholdStrategy()

	ctx := decision.NewContext("test", 100).
		WithCurrentState(&monitor.SystemState{
			CPU:    monitor.CPUState{UsagePercent: 50.0},
			Memory: monitor.MemoryState{UsagePercent: 40.0},
			GPUs: []monitor.GPUState{
				{UsagePercent: 30.0, VRAMUsedBytes: 5000, VRAMTotalBytes: 10000},
			},
			Storage: monitor.StorageState{
				"/": {UsedBytes: 100 * 1024 * 1024 * 1024, TotalBytes: 200 * 1024 * 1024 * 1024},
			},
		}).
		WithThresholds(&decision.ThresholdsConfig{
			CPU:     decision.CPUThreshold{MaxPercent: 80.0},
			Memory:  decision.MemoryThreshold{MaxPercent: 80.0},
			GPU:     decision.GPUThreshold{MaxPercent: 80.0},
			VRAM:    decision.VRAMThreshold{MaxPercent: 80.0},
			Storage: decision.StorageThreshold{MinFreeGB: 10.0},
		})

	result := s.Decide(ctx)

	if !result.Allowed {
		t.Error("expected allowed=true when below thresholds")
	}
	if len(result.Reasons) != 0 {
		t.Errorf("expected no reasons, got %v", result.Reasons)
	}
	if result.Confidence != 1.0 {
		t.Errorf("expected confidence 1.0, got %f", result.Confidence)
	}
}

func TestThresholdStrategy_Decide_RejectsCPUOverload(t *testing.T) {
	s := NewThresholdStrategy()

	ctx := decision.NewContext("test", 100).
		WithCurrentState(&monitor.SystemState{
			CPU:    monitor.CPUState{UsagePercent: 90.0},
			Memory: monitor.MemoryState{UsagePercent: 40.0},
		}).
		WithThresholds(&decision.ThresholdsConfig{
			CPU:     decision.CPUThreshold{MaxPercent: 80.0},
			Memory:  decision.MemoryThreshold{MaxPercent: 80.0},
			Storage: decision.StorageThreshold{MinFreeGB: 0},
		})

	result := s.Decide(ctx)

	if result.Allowed {
		t.Error("expected allowed=false for CPU overload")
	}
	if !containsReason(result.Reasons, decision.ReasonCPUOverload) {
		t.Error("expected ReasonCPUOverload in reasons")
	}
}

func TestThresholdStrategy_Decide_RejectsMemoryOverload(t *testing.T) {
	s := NewThresholdStrategy()

	ctx := decision.NewContext("test", 100).
		WithCurrentState(&monitor.SystemState{
			CPU:    monitor.CPUState{UsagePercent: 50.0},
			Memory: monitor.MemoryState{UsagePercent: 95.0},
		}).
		WithThresholds(&decision.ThresholdsConfig{
			CPU:     decision.CPUThreshold{MaxPercent: 80.0},
			Memory:  decision.MemoryThreshold{MaxPercent: 80.0},
			Storage: decision.StorageThreshold{MinFreeGB: 0},
		})

	result := s.Decide(ctx)

	if result.Allowed {
		t.Error("expected allowed=false for memory overload")
	}
	if !containsReason(result.Reasons, decision.ReasonMemoryOverload) {
		t.Error("expected ReasonMemoryOverload in reasons")
	}
}

func TestThresholdStrategy_Decide_RejectsGPUOverload(t *testing.T) {
	s := NewThresholdStrategy()

	ctx := decision.NewContext("test", 100).
		WithCurrentState(&monitor.SystemState{
			CPU:    monitor.CPUState{UsagePercent: 50.0},
			Memory: monitor.MemoryState{UsagePercent: 40.0},
			GPUs: []monitor.GPUState{
				{UsagePercent: 95.0, VRAMUsedBytes: 5000, VRAMTotalBytes: 10000},
			},
		}).
		WithThresholds(&decision.ThresholdsConfig{
			CPU:     decision.CPUThreshold{MaxPercent: 80.0},
			Memory:  decision.MemoryThreshold{MaxPercent: 80.0},
			GPU:     decision.GPUThreshold{MaxPercent: 80.0},
			VRAM:    decision.VRAMThreshold{MaxPercent: 80.0},
			Storage: decision.StorageThreshold{MinFreeGB: 0},
		})

	result := s.Decide(ctx)

	if result.Allowed {
		t.Error("expected allowed=false for GPU overload")
	}
	if !containsReason(result.Reasons, decision.ReasonGPUOverload) {
		t.Error("expected ReasonGPUOverload in reasons")
	}
}

func TestThresholdStrategy_Decide_RejectsVRAMOverload(t *testing.T) {
	s := NewThresholdStrategy()

	ctx := decision.NewContext("test", 100).
		WithCurrentState(&monitor.SystemState{
			CPU:    monitor.CPUState{UsagePercent: 50.0},
			Memory: monitor.MemoryState{UsagePercent: 40.0},
			GPUs: []monitor.GPUState{
				{UsagePercent: 50.0, VRAMUsedBytes: 9000, VRAMTotalBytes: 10000}, // 90% VRAM
			},
		}).
		WithThresholds(&decision.ThresholdsConfig{
			CPU:     decision.CPUThreshold{MaxPercent: 80.0},
			Memory:  decision.MemoryThreshold{MaxPercent: 80.0},
			GPU:     decision.GPUThreshold{MaxPercent: 80.0},
			VRAM:    decision.VRAMThreshold{MaxPercent: 80.0},
			Storage: decision.StorageThreshold{MinFreeGB: 0},
		})

	result := s.Decide(ctx)

	if result.Allowed {
		t.Error("expected allowed=false for VRAM overload")
	}
	if !containsReason(result.Reasons, decision.ReasonVRAMOverload) {
		t.Error("expected ReasonVRAMOverload in reasons")
	}
}

func TestThresholdStrategy_Decide_RejectsStorageLow(t *testing.T) {
	s := NewThresholdStrategy()

	// 5GB free (195GB used out of 200GB)
	ctx := decision.NewContext("test", 100).
		WithCurrentState(&monitor.SystemState{
			CPU:    monitor.CPUState{UsagePercent: 50.0},
			Memory: monitor.MemoryState{UsagePercent: 40.0},
			Storage: monitor.StorageState{
				"/": {UsedBytes: 195 * 1024 * 1024 * 1024, TotalBytes: 200 * 1024 * 1024 * 1024},
			},
		}).
		WithThresholds(&decision.ThresholdsConfig{
			CPU:     decision.CPUThreshold{MaxPercent: 80.0},
			Memory:  decision.MemoryThreshold{MaxPercent: 80.0},
			Storage: decision.StorageThreshold{MinFreeGB: 10.0},
		})

	result := s.Decide(ctx)

	if result.Allowed {
		t.Error("expected allowed=false for low storage")
	}
	if !containsReason(result.Reasons, decision.ReasonStorageLow) {
		t.Error("expected ReasonStorageLow in reasons")
	}
}

func TestThresholdStrategy_Decide_MultipleReasons(t *testing.T) {
	s := NewThresholdStrategy()

	ctx := decision.NewContext("test", 100).
		WithCurrentState(&monitor.SystemState{
			CPU:    monitor.CPUState{UsagePercent: 90.0},
			Memory: monitor.MemoryState{UsagePercent: 95.0},
		}).
		WithThresholds(&decision.ThresholdsConfig{
			CPU:     decision.CPUThreshold{MaxPercent: 80.0},
			Memory:  decision.MemoryThreshold{MaxPercent: 80.0},
			Storage: decision.StorageThreshold{MinFreeGB: 0},
		})

	result := s.Decide(ctx)

	if result.Allowed {
		t.Error("expected allowed=false")
	}
	if len(result.Reasons) != 2 {
		t.Errorf("expected 2 reasons, got %d", len(result.Reasons))
	}
	if !containsReason(result.Reasons, decision.ReasonCPUOverload) {
		t.Error("expected ReasonCPUOverload in reasons")
	}
	if !containsReason(result.Reasons, decision.ReasonMemoryOverload) {
		t.Error("expected ReasonMemoryOverload in reasons")
	}
}

// Helper function
func containsReason(reasons []decision.Reason, target decision.Reason) bool {
	for _, r := range reasons {
		if r == target {
			return true
		}
	}
	return false
}

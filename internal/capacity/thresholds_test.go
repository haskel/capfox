package capacity

import (
	"testing"

	"github.com/haskel/capfox/internal/config"
	"github.com/haskel/capfox/internal/monitor"
)

func defaultThresholds() config.ThresholdsConfig {
	return config.ThresholdsConfig{
		CPU:     config.CPUThreshold{MaxPercent: 80},
		Memory:  config.MemoryThreshold{MaxPercent: 85},
		GPU:     config.GPUThreshold{MaxPercent: 90},
		VRAM:    config.VRAMThreshold{MaxPercent: 85},
		Storage: config.StorageThreshold{MinFreeGB: 10},
	}
}

func TestThresholdChecker_AllOK(t *testing.T) {
	checker := NewThresholdChecker(defaultThresholds())

	state := &monitor.SystemState{
		CPU:    monitor.CPUState{UsagePercent: 50},
		Memory: monitor.MemoryState{UsagePercent: 50},
		GPUs:   []monitor.GPUState{},
		Storage: monitor.StorageState{
			"/": {UsedBytes: 100 * 1024 * 1024 * 1024, TotalBytes: 500 * 1024 * 1024 * 1024},
		},
	}

	reasons := checker.Check(state)

	if len(reasons) != 0 {
		t.Errorf("expected no reasons, got %v", reasons)
	}
}

func TestThresholdChecker_CPUOverload(t *testing.T) {
	checker := NewThresholdChecker(defaultThresholds())

	state := &monitor.SystemState{
		CPU:     monitor.CPUState{UsagePercent: 85},
		Memory:  monitor.MemoryState{UsagePercent: 50},
		Storage: monitor.StorageState{},
	}

	reasons := checker.Check(state)

	if len(reasons) != 1 || reasons[0] != ReasonCPUOverload {
		t.Errorf("expected [cpu_overload], got %v", reasons)
	}
}

func TestThresholdChecker_MemoryOverload(t *testing.T) {
	checker := NewThresholdChecker(defaultThresholds())

	state := &monitor.SystemState{
		CPU:     monitor.CPUState{UsagePercent: 50},
		Memory:  monitor.MemoryState{UsagePercent: 90},
		Storage: monitor.StorageState{},
	}

	reasons := checker.Check(state)

	if len(reasons) != 1 || reasons[0] != ReasonMemoryOverload {
		t.Errorf("expected [memory_overload], got %v", reasons)
	}
}

func TestThresholdChecker_StorageLow(t *testing.T) {
	checker := NewThresholdChecker(defaultThresholds())

	// 5 GB free (less than 10 GB threshold)
	state := &monitor.SystemState{
		CPU:    monitor.CPUState{UsagePercent: 50},
		Memory: monitor.MemoryState{UsagePercent: 50},
		Storage: monitor.StorageState{
			"/": {
				UsedBytes:  495 * 1024 * 1024 * 1024,
				TotalBytes: 500 * 1024 * 1024 * 1024,
			},
		},
	}

	reasons := checker.Check(state)

	if len(reasons) != 1 || reasons[0] != ReasonStorageLow {
		t.Errorf("expected [storage_low], got %v", reasons)
	}
}

func TestThresholdChecker_GPUOverload(t *testing.T) {
	checker := NewThresholdChecker(defaultThresholds())

	state := &monitor.SystemState{
		CPU:    monitor.CPUState{UsagePercent: 50},
		Memory: monitor.MemoryState{UsagePercent: 50},
		GPUs: []monitor.GPUState{
			{Index: 0, UsagePercent: 95},
		},
		Storage: monitor.StorageState{},
	}

	reasons := checker.Check(state)

	if len(reasons) != 1 || reasons[0] != ReasonGPUOverload {
		t.Errorf("expected [gpu_overload], got %v", reasons)
	}
}

func TestThresholdChecker_VRAMOverload(t *testing.T) {
	checker := NewThresholdChecker(defaultThresholds())

	state := &monitor.SystemState{
		CPU:    monitor.CPUState{UsagePercent: 50},
		Memory: monitor.MemoryState{UsagePercent: 50},
		GPUs: []monitor.GPUState{
			{
				Index:          0,
				UsagePercent:   50,
				VRAMUsedBytes:  9 * 1024 * 1024 * 1024,
				VRAMTotalBytes: 10 * 1024 * 1024 * 1024,
			},
		},
		Storage: monitor.StorageState{},
	}

	reasons := checker.Check(state)

	if len(reasons) != 1 || reasons[0] != ReasonVRAMOverload {
		t.Errorf("expected [vram_overload], got %v", reasons)
	}
}

func TestThresholdChecker_MultipleReasons(t *testing.T) {
	checker := NewThresholdChecker(defaultThresholds())

	state := &monitor.SystemState{
		CPU:     monitor.CPUState{UsagePercent: 90},
		Memory:  monitor.MemoryState{UsagePercent: 90},
		Storage: monitor.StorageState{},
	}

	reasons := checker.Check(state)

	if len(reasons) != 2 {
		t.Errorf("expected 2 reasons, got %v", reasons)
	}

	hasCPU := false
	hasMemory := false
	for _, r := range reasons {
		if r == ReasonCPUOverload {
			hasCPU = true
		}
		if r == ReasonMemoryOverload {
			hasMemory = true
		}
	}

	if !hasCPU || !hasMemory {
		t.Errorf("expected cpu_overload and memory_overload, got %v", reasons)
	}
}

func TestThresholdChecker_UpdateThresholds(t *testing.T) {
	checker := NewThresholdChecker(defaultThresholds())

	state := &monitor.SystemState{
		CPU:     monitor.CPUState{UsagePercent: 85},
		Memory:  monitor.MemoryState{UsagePercent: 50},
		Storage: monitor.StorageState{},
	}

	// Initially should fail
	reasons := checker.Check(state)
	if len(reasons) != 1 {
		t.Errorf("expected 1 reason, got %v", reasons)
	}

	// Update threshold to 90%
	newThresholds := defaultThresholds()
	newThresholds.CPU.MaxPercent = 90
	checker.UpdateThresholds(newThresholds)

	// Now should pass
	reasons = checker.Check(state)
	if len(reasons) != 0 {
		t.Errorf("expected no reasons after update, got %v", reasons)
	}
}

func TestThresholdChecker_ConcurrentAccess(t *testing.T) {
	checker := NewThresholdChecker(defaultThresholds())

	state := &monitor.SystemState{
		CPU:     monitor.CPUState{UsagePercent: 75},
		Memory:  monitor.MemoryState{UsagePercent: 50},
		Storage: monitor.StorageState{},
	}

	// Run concurrent Check and UpdateThresholds
	done := make(chan bool)

	// Start readers
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				_ = checker.Check(state)
			}
			done <- true
		}()
	}

	// Start writers
	for i := 0; i < 5; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				newThresholds := defaultThresholds()
				newThresholds.CPU.MaxPercent = float64(70 + j%20)
				checker.UpdateThresholds(newThresholds)
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 15; i++ {
		<-done
	}
}

func TestThresholdChecker_GetThresholds(t *testing.T) {
	original := defaultThresholds()
	checker := NewThresholdChecker(original)

	got := checker.GetThresholds()
	if got.CPU.MaxPercent != original.CPU.MaxPercent {
		t.Errorf("expected CPU threshold %f, got %f", original.CPU.MaxPercent, got.CPU.MaxPercent)
	}

	// Update and verify
	newThresholds := defaultThresholds()
	newThresholds.CPU.MaxPercent = 95
	checker.UpdateThresholds(newThresholds)

	got = checker.GetThresholds()
	if got.CPU.MaxPercent != 95 {
		t.Errorf("expected CPU threshold 95, got %f", got.CPU.MaxPercent)
	}
}

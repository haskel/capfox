package monitor

import (
	"testing"
)

func TestGPUMonitor_Name(t *testing.T) {
	m := NewGPUMonitor()
	if m.Name() != "gpu" {
		t.Errorf("expected name 'gpu', got %s", m.Name())
	}
}

func TestGPUMonitor_GracefulDegradation(t *testing.T) {
	m := NewGPUMonitor()

	// Even without NVML, should not fail
	data, err := m.Collect()
	if err != nil {
		t.Fatalf("collect should not fail: %v", err)
	}

	states, ok := data.([]GPUState)
	if !ok {
		t.Fatalf("expected []GPUState, got %T", data)
	}

	// Without NVML, should return empty slice
	if !m.Available() && len(states) != 0 {
		t.Errorf("expected empty slice when NVML not available, got %d", len(states))
	}
}

func TestGPUMonitor_Close(t *testing.T) {
	m := NewGPUMonitor()

	err := m.Close()
	if err != nil {
		t.Errorf("close should not fail: %v", err)
	}
}

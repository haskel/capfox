package monitor

import (
	"testing"
)

func TestCPUMonitor_Name(t *testing.T) {
	m := NewCPUMonitor()
	if m.Name() != "cpu" {
		t.Errorf("expected name 'cpu', got %s", m.Name())
	}
}

func TestCPUMonitor_Collect(t *testing.T) {
	m := NewCPUMonitor()

	data, err := m.Collect()
	if err != nil {
		t.Fatalf("failed to collect CPU data: %v", err)
	}

	state, ok := data.(*CPUState)
	if !ok {
		t.Fatalf("expected *CPUState, got %T", data)
	}

	if state.UsagePercent < 0 || state.UsagePercent > 100 {
		t.Errorf("invalid CPU usage percent: %f", state.UsagePercent)
	}

	if len(state.Cores) == 0 {
		t.Error("expected at least one core")
	}

	for i, core := range state.Cores {
		if core < 0 || core > 100 {
			t.Errorf("invalid core %d usage: %f", i, core)
		}
	}
}

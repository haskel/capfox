package monitor

import (
	"testing"
)

func TestMemoryMonitor_Name(t *testing.T) {
	m := NewMemoryMonitor()
	if m.Name() != "memory" {
		t.Errorf("expected name 'memory', got %s", m.Name())
	}
}

func TestMemoryMonitor_Collect(t *testing.T) {
	m := NewMemoryMonitor()

	data, err := m.Collect()
	if err != nil {
		t.Fatalf("failed to collect memory data: %v", err)
	}

	state, ok := data.(*MemoryState)
	if !ok {
		t.Fatalf("expected *MemoryState, got %T", data)
	}

	if state.TotalBytes == 0 {
		t.Error("total bytes should not be zero")
	}

	if state.UsedBytes > state.TotalBytes {
		t.Errorf("used bytes (%d) should not exceed total (%d)", state.UsedBytes, state.TotalBytes)
	}

	if state.UsagePercent < 0 || state.UsagePercent > 100 {
		t.Errorf("invalid memory usage percent: %f", state.UsagePercent)
	}
}

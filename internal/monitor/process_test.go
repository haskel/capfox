package monitor

import (
	"testing"
)

func TestProcessMonitor_Name(t *testing.T) {
	m := NewProcessMonitor()
	if m.Name() != "process" {
		t.Errorf("expected name 'process', got %s", m.Name())
	}
}

func TestProcessMonitor_Collect(t *testing.T) {
	m := NewProcessMonitor()

	data, err := m.Collect()
	if err != nil {
		t.Fatalf("failed to collect process data: %v", err)
	}

	state, ok := data.(*ProcessState)
	if !ok {
		t.Fatalf("expected *ProcessState, got %T", data)
	}

	if state.Processes <= 0 {
		t.Error("expected at least one process")
	}

	if state.Threads <= 0 {
		t.Error("expected at least one thread")
	}

	// Context switches per sec can be 0 on first call
	if state.ContextSwitchesPerSec < 0 {
		t.Errorf("context switches should not be negative: %d", state.ContextSwitchesPerSec)
	}
}

func TestProcessMonitor_CollectTwice(t *testing.T) {
	m := NewProcessMonitor()

	// First call
	_, err := m.Collect()
	if err != nil {
		t.Fatalf("first collect failed: %v", err)
	}

	// Second call - should calculate context switches rate
	data, err := m.Collect()
	if err != nil {
		t.Fatalf("second collect failed: %v", err)
	}

	state, ok := data.(*ProcessState)
	if !ok {
		t.Fatalf("expected *ProcessState, got %T", data)
	}

	// After second call, we should have some data
	if state.Processes <= 0 {
		t.Error("expected at least one process")
	}
}

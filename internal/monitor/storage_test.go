package monitor

import (
	"testing"
)

func TestStorageMonitor_Name(t *testing.T) {
	m := NewStorageMonitor(nil)
	if m.Name() != "storage" {
		t.Errorf("expected name 'storage', got %s", m.Name())
	}
}

func TestStorageMonitor_DefaultPath(t *testing.T) {
	m := NewStorageMonitor(nil)
	if len(m.paths) != 1 || m.paths[0] != "/" {
		t.Errorf("expected default path ['/'], got %v", m.paths)
	}
}

func TestStorageMonitor_CustomPaths(t *testing.T) {
	paths := []string{"/", "/tmp"}
	m := NewStorageMonitor(paths)
	if len(m.paths) != 2 {
		t.Errorf("expected 2 paths, got %d", len(m.paths))
	}
}

func TestStorageMonitor_Collect(t *testing.T) {
	m := NewStorageMonitor([]string{"/"})

	data, err := m.Collect()
	if err != nil {
		t.Fatalf("failed to collect storage data: %v", err)
	}

	state, ok := data.(StorageState)
	if !ok {
		t.Fatalf("expected StorageState, got %T", data)
	}

	diskState, exists := state["/"]
	if !exists {
		t.Fatal("expected '/' in storage state")
	}

	if diskState.TotalBytes == 0 {
		t.Error("total bytes should not be zero")
	}

	if diskState.UsedBytes > diskState.TotalBytes {
		t.Errorf("used bytes (%d) should not exceed total (%d)", diskState.UsedBytes, diskState.TotalBytes)
	}

	if diskState.UsagePercent < 0 || diskState.UsagePercent > 100 {
		t.Errorf("invalid storage usage percent: %f", diskState.UsagePercent)
	}
}

func TestStorageMonitor_NonExistentPath(t *testing.T) {
	m := NewStorageMonitor([]string{"/nonexistent/path/that/does/not/exist"})

	data, err := m.Collect()
	if err != nil {
		t.Fatalf("collect should not fail: %v", err)
	}

	state, ok := data.(StorageState)
	if !ok {
		t.Fatalf("expected StorageState, got %T", data)
	}

	// Non-existent path should be skipped
	if len(state) != 0 {
		t.Errorf("expected empty state for non-existent path, got %d entries", len(state))
	}
}

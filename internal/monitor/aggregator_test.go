package monitor

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
}

type mockMonitor struct {
	name string
	data any
	err  error
}

func (m *mockMonitor) Name() string {
	return m.name
}

func (m *mockMonitor) Collect() (any, error) {
	return m.data, m.err
}

func TestAggregator_GetState(t *testing.T) {
	monitors := []Monitor{
		&mockMonitor{
			name: "cpu",
			data: &CPUState{UsagePercent: 50.0, Cores: []float64{40.0, 60.0}},
		},
		&mockMonitor{
			name: "memory",
			data: &MemoryState{UsedBytes: 1024, TotalBytes: 2048, UsagePercent: 50.0},
		},
	}

	agg := NewAggregator(monitors, time.Second, testLogger())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := agg.Start(ctx)
	if err != nil {
		t.Fatalf("failed to start aggregator: %v", err)
	}

	state := agg.GetState()

	if state.CPU.UsagePercent != 50.0 {
		t.Errorf("expected CPU usage 50.0, got %f", state.CPU.UsagePercent)
	}

	if len(state.CPU.Cores) != 2 {
		t.Errorf("expected 2 cores, got %d", len(state.CPU.Cores))
	}

	if state.Memory.UsagePercent != 50.0 {
		t.Errorf("expected memory usage 50.0, got %f", state.Memory.UsagePercent)
	}

	agg.Stop()
}

func TestAggregator_StateClone(t *testing.T) {
	state := &SystemState{
		CPU: CPUState{
			UsagePercent: 50.0,
			Cores:        []float64{40.0, 60.0},
		},
		GPUs: []GPUState{
			{Index: 0, Name: "GPU0"},
		},
		Storage: StorageState{
			"/": {UsedBytes: 100, TotalBytes: 200},
		},
	}

	clone := state.Clone()

	// Modify original
	state.CPU.Cores[0] = 100.0
	state.GPUs[0].Name = "Modified"
	state.Storage["/tmp"] = DiskState{}

	// Clone should be unchanged
	if clone.CPU.Cores[0] != 40.0 {
		t.Errorf("clone cores modified: %f", clone.CPU.Cores[0])
	}

	if clone.GPUs[0].Name != "GPU0" {
		t.Errorf("clone GPU name modified: %s", clone.GPUs[0].Name)
	}

	if _, exists := clone.Storage["/tmp"]; exists {
		t.Error("clone storage should not have /tmp")
	}
}

func TestAggregator_GetStateJSON(t *testing.T) {
	monitors := []Monitor{
		&mockMonitor{
			name: "cpu",
			data: &CPUState{UsagePercent: 25.5, Cores: []float64{25.5}},
		},
	}

	agg := NewAggregator(monitors, time.Second, testLogger())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	agg.Start(ctx)
	defer agg.Stop()

	jsonData, err := agg.GetStateJSON()
	if err != nil {
		t.Fatalf("failed to get JSON: %v", err)
	}

	if len(jsonData) == 0 {
		t.Error("expected non-empty JSON")
	}

	// Check that JSON contains expected fields
	jsonStr := string(jsonData)
	if !contains(jsonStr, "usage_percent") {
		t.Error("JSON should contain usage_percent")
	}
	if !contains(jsonStr, "25.5") {
		t.Error("JSON should contain CPU value 25.5")
	}
}

func TestAggregator_IntegrationWithRealMonitors(t *testing.T) {
	monitors := []Monitor{
		NewCPUMonitor(),
		NewMemoryMonitor(),
		NewStorageMonitor([]string{"/"}),
		NewProcessMonitor(),
		NewGPUMonitor(),
	}

	agg := NewAggregator(monitors, 100*time.Millisecond, testLogger())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := agg.Start(ctx)
	if err != nil {
		t.Fatalf("failed to start aggregator: %v", err)
	}

	// Wait for at least one collection
	time.Sleep(150 * time.Millisecond)

	state := agg.GetState()

	// Verify CPU data
	if state.CPU.UsagePercent < 0 || state.CPU.UsagePercent > 100 {
		t.Errorf("invalid CPU usage: %f", state.CPU.UsagePercent)
	}

	// Verify Memory data
	if state.Memory.TotalBytes == 0 {
		t.Error("memory total should not be zero")
	}

	// Verify Storage data
	if len(state.Storage) == 0 {
		t.Error("expected at least one storage entry")
	}

	// Verify Process data
	if state.Processes == 0 {
		t.Error("expected at least one process")
	}

	// Verify Timestamp
	if state.Timestamp.IsZero() {
		t.Error("timestamp should not be zero")
	}

	agg.Stop()
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

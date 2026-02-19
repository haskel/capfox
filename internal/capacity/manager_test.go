package capacity

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/haskel/capfox/internal/monitor"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
}

type mockMonitor struct {
	name string
	data any
}

func (m *mockMonitor) Name() string {
	return m.name
}

func (m *mockMonitor) Collect() (any, error) {
	return m.data, nil
}

func testAggregator(cpuPercent, memPercent float64) *monitor.Aggregator {
	monitors := []monitor.Monitor{
		&mockMonitor{
			name: "cpu",
			data: &monitor.CPUState{UsagePercent: cpuPercent},
		},
		&mockMonitor{
			name: "memory",
			data: &monitor.MemoryState{
				UsagePercent: memPercent,
				UsedBytes:    1024,
				TotalBytes:   2048,
			},
		},
		&mockMonitor{
			name: "storage",
			data: monitor.StorageState{
				"/": {
					UsedBytes:  100 * 1024 * 1024 * 1024,
					TotalBytes: 500 * 1024 * 1024 * 1024,
				},
			},
		},
	}

	agg := monitor.NewAggregator(monitors, time.Second, testLogger())
	ctx := context.Background()
	_ = agg.Start(ctx)

	return agg
}

func TestManager_Ask_Allowed(t *testing.T) {
	agg := testAggregator(50, 50)
	defer agg.Stop()

	manager := NewManager(agg, defaultThresholds())

	req := AskRequest{
		Task:       "test_task",
		Complexity: 100,
	}

	resp := manager.Ask(req, false)

	if !resp.Allowed {
		t.Error("expected allowed=true")
	}

	if resp.Reasons != nil {
		t.Errorf("expected no reasons, got %v", resp.Reasons)
	}
}

func TestManager_Ask_Denied(t *testing.T) {
	agg := testAggregator(90, 50) // CPU overload
	defer agg.Stop()

	manager := NewManager(agg, defaultThresholds())

	req := AskRequest{
		Task:       "test_task",
		Complexity: 100,
	}

	resp := manager.Ask(req, false)

	if resp.Allowed {
		t.Error("expected allowed=false")
	}

	// Without withReasons, should not have reasons
	if resp.Reasons != nil {
		t.Errorf("expected no reasons without flag, got %v", resp.Reasons)
	}
}

func TestManager_Ask_WithReasons(t *testing.T) {
	agg := testAggregator(90, 90) // CPU and Memory overload
	defer agg.Stop()

	manager := NewManager(agg, defaultThresholds())

	req := AskRequest{
		Task:       "test_task",
		Complexity: 100,
	}

	resp := manager.Ask(req, true)

	if resp.Allowed {
		t.Error("expected allowed=false")
	}

	if len(resp.Reasons) != 2 {
		t.Errorf("expected 2 reasons, got %v", resp.Reasons)
	}

	hasCPU := false
	hasMemory := false
	for _, r := range resp.Reasons {
		if r == "cpu_overload" {
			hasCPU = true
		}
		if r == "memory_overload" {
			hasMemory = true
		}
	}

	if !hasCPU || !hasMemory {
		t.Errorf("expected cpu_overload and memory_overload, got %v", resp.Reasons)
	}
}

func TestManager_Ask_WithResources(t *testing.T) {
	agg := testAggregator(50, 50)
	defer agg.Stop()

	manager := NewManager(agg, defaultThresholds())

	req := AskRequest{
		Task: "test_task",
		Resources: &ResourceEstimate{
			CPU:    100,
			GPU:    50,
			Memory: 200,
		},
	}

	resp := manager.Ask(req, false)

	// Currently resources are not used in decision making
	// This will be implemented in Learning Engine phase
	if !resp.Allowed {
		t.Error("expected allowed=true")
	}
}

func TestManager_UpdateThresholds(t *testing.T) {
	agg := testAggregator(85, 50) // CPU at 85%
	defer agg.Stop()

	manager := NewManager(agg, defaultThresholds())

	req := AskRequest{Task: "test"}

	// Initially denied (threshold is 80%)
	resp := manager.Ask(req, false)
	if resp.Allowed {
		t.Error("expected denied with 80% threshold")
	}

	// Update threshold to 90%
	newThresholds := defaultThresholds()
	newThresholds.CPU.MaxPercent = 90
	manager.UpdateThresholds(newThresholds)

	// Now should be allowed
	resp = manager.Ask(req, false)
	if !resp.Allowed {
		t.Error("expected allowed with 90% threshold")
	}
}

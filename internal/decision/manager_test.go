package decision

import (
	"sync"
	"testing"

	"github.com/haskel/capfox/internal/monitor"
)

// mockStrategy implements Strategy for testing.
type mockStrategy struct{}

func (m *mockStrategy) Name() string { return "mock" }
func (m *mockStrategy) Decide(ctx *Context) *Result {
	return &Result{Allowed: true, Strategy: "mock"}
}

// mockModel implements PredictionModel for testing.
type mockModel struct{}

func (m *mockModel) Name() string                                         { return "mock" }
func (m *mockModel) Predict(task string, complexity int) *ResourceImpact  { return nil }
func (m *mockModel) Observe(task string, complexity int, impact *ResourceImpact) {}
func (m *mockModel) Confidence(task string) float64                       { return 0 }

// mockAggregator creates a minimal aggregator for testing.
func mockAggregator() *monitor.Aggregator {
	return monitor.NewAggregator(nil, 0, nil)
}

func TestManager_PendingTasks_ConcurrentAccess(t *testing.T) {
	mgr := NewManager(
		&mockStrategy{},
		&mockModel{},
		mockAggregator(),
		ManagerConfig{
			Thresholds: &ThresholdsConfig{
				CPU:    CPUThreshold{MaxPercent: 80},
				Memory: MemoryThreshold{MaxPercent: 85},
			},
		},
	)

	var wg sync.WaitGroup
	iterations := 100

	// Concurrent writers
	for i := 0; i < iterations; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			mgr.AddPendingTask(PendingTask{
				Task:       "test-task",
				Complexity: id,
			})
		}(i)
	}

	// Concurrent readers (Decide calls)
	for i := 0; i < iterations; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// This should not race with AddPendingTask
			_ = mgr.Decide("test", 10, nil)
		}()
	}

	// Concurrent removers
	for i := 0; i < iterations/2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			mgr.RemovePendingTask("test-task")
		}()
	}

	wg.Wait()

	// Verify no panic occurred and manager is still functional
	result := mgr.Decide("final-test", 5, nil)
	if result == nil {
		t.Error("expected non-nil result")
	}
}

func TestManager_AddRemovePendingTask(t *testing.T) {
	mgr := NewManager(
		&mockStrategy{},
		nil,
		mockAggregator(),
		ManagerConfig{
			Thresholds: &ThresholdsConfig{},
		},
	)

	// Initially empty
	if mgr.PendingCount() != 0 {
		t.Errorf("expected 0 pending tasks, got %d", mgr.PendingCount())
	}

	// Add tasks
	mgr.AddPendingTask(PendingTask{Task: "task1", Complexity: 10})
	mgr.AddPendingTask(PendingTask{Task: "task2", Complexity: 20})

	if mgr.PendingCount() != 2 {
		t.Errorf("expected 2 pending tasks, got %d", mgr.PendingCount())
	}

	// Remove one
	mgr.RemovePendingTask("task1")

	if mgr.PendingCount() != 1 {
		t.Errorf("expected 1 pending task, got %d", mgr.PendingCount())
	}

	// Remove non-existent (should not panic)
	mgr.RemovePendingTask("non-existent")

	if mgr.PendingCount() != 1 {
		t.Errorf("expected 1 pending task, got %d", mgr.PendingCount())
	}
}

func TestManager_UpdateThresholds(t *testing.T) {
	mgr := NewManager(
		&mockStrategy{},
		nil,
		mockAggregator(),
		ManagerConfig{
			Thresholds: &ThresholdsConfig{CPU: CPUThreshold{MaxPercent: 80}},
		},
	)

	newThresholds := &ThresholdsConfig{CPU: CPUThreshold{MaxPercent: 90}}
	mgr.UpdateThresholds(newThresholds)

	// Verify update was applied (indirect check via Decide)
	result := mgr.Decide("test", 10, nil)
	if result == nil {
		t.Error("expected non-nil result after threshold update")
	}
}

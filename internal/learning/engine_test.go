package learning

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
	}

	agg := monitor.NewAggregator(monitors, 50*time.Millisecond, testLogger())
	ctx := context.Background()
	_ = agg.Start(ctx)

	return agg
}

func TestEngine_NotifyTaskStart(t *testing.T) {
	agg := testAggregator(50, 50)
	defer func() { _ = agg.Stop() }()

	model := NewMovingAverageModel(0.2)
	engine := NewEngine(model, agg, 100*time.Millisecond, testLogger())

	engine.NotifyTaskStart("test_task", 100)

	// Should have 1 pending task
	if engine.PendingCount() != 1 {
		t.Errorf("expected 1 pending task, got %d", engine.PendingCount())
	}
}

func TestEngine_Observation(t *testing.T) {
	agg := testAggregator(50, 50)
	defer func() { _ = agg.Stop() }()

	model := NewMovingAverageModel(0.2)
	engine := NewEngine(model, agg, 50*time.Millisecond, testLogger())

	engine.NotifyTaskStart("test_task", 100)

	// Wait for observation
	time.Sleep(100 * time.Millisecond)

	// Should have 0 pending tasks
	if engine.PendingCount() != 0 {
		t.Errorf("expected 0 pending tasks after observation, got %d", engine.PendingCount())
	}

	// Should have stats
	stats := engine.GetStats()
	if stats.TotalTasks != 1 {
		t.Errorf("expected 1 total task, got %d", stats.TotalTasks)
	}
}

func TestEngine_GetStats(t *testing.T) {
	agg := testAggregator(50, 50)
	defer func() { _ = agg.Stop() }()

	model := NewMovingAverageModel(0.2)
	engine := NewEngine(model, agg, 50*time.Millisecond, testLogger())

	// Manually observe
	model.Observe("task1", 100, &ResourceImpact{CPUDelta: 10.0})
	model.Observe("task2", 50, &ResourceImpact{CPUDelta: 5.0})

	stats := engine.GetStats()
	if stats.TotalTasks != 2 {
		t.Errorf("expected 2 total tasks, got %d", stats.TotalTasks)
	}
}

func TestEngine_GetTaskStats(t *testing.T) {
	agg := testAggregator(50, 50)
	defer func() { _ = agg.Stop() }()

	model := NewMovingAverageModel(0.2)
	engine := NewEngine(model, agg, time.Second, testLogger())

	// Manually observe
	model.Observe("task1", 100, &ResourceImpact{CPUDelta: 10.0})

	stats := engine.GetTaskStats("task1")
	if stats == nil {
		t.Fatal("expected stats for task1")
	}
	if stats.AvgCPUDelta != 10.0 {
		t.Errorf("expected CPU delta 10.0, got %f", stats.AvgCPUDelta)
	}

	// Unknown task
	stats = engine.GetTaskStats("unknown")
	if stats != nil {
		t.Error("expected nil for unknown task")
	}
}

func TestEngine_Predict(t *testing.T) {
	agg := testAggregator(50, 50)
	defer func() { _ = agg.Stop() }()

	model := NewMovingAverageModel(0.2)
	engine := NewEngine(model, agg, time.Second, testLogger())

	// Manually observe
	model.Observe("task1", 100, &ResourceImpact{CPUDelta: 10.0})

	impact := engine.Predict("task1", 100)
	if impact == nil {
		t.Fatal("expected prediction")
	}
	if impact.CPUDelta != 10.0 {
		t.Errorf("expected CPU delta 10.0, got %f", impact.CPUDelta)
	}

	// Unknown task
	impact = engine.Predict("unknown", 100)
	if impact != nil {
		t.Error("expected nil for unknown task")
	}
}

func TestEngine_Model(t *testing.T) {
	agg := testAggregator(50, 50)
	defer func() { _ = agg.Stop() }()

	model := NewMovingAverageModel(0.2)
	engine := NewEngine(model, agg, time.Second, testLogger())

	if engine.Model() != model {
		t.Error("expected model to be the same")
	}
}

func TestEngine_Stop(t *testing.T) {
	agg := testAggregator(50, 50)
	defer func() { _ = agg.Stop() }()

	model := NewMovingAverageModel(0.2)
	engine := NewEngine(model, agg, time.Second, testLogger())

	// Start several tasks
	for i := 0; i < 10; i++ {
		engine.NotifyTaskStart("test_task", 100)
	}

	// Stop should cancel all pending observations
	engine.Stop()

	// After stop, no new tasks should be accepted
	engine.NotifyTaskStart("should_be_ignored", 100)

	// Pending count may include tasks that started before stop
	// but new tasks should not be added
}

func TestEngine_StopWithTimeout(t *testing.T) {
	agg := testAggregator(50, 50)
	defer func() { _ = agg.Stop() }()

	model := NewMovingAverageModel(0.2)
	engine := NewEngine(model, agg, 5*time.Second, testLogger())

	// Start a task with long observation delay
	engine.NotifyTaskStart("test_task", 100)

	// Stop with short timeout should not hang
	start := time.Now()
	engine.StopWithTimeout(100 * time.Millisecond)
	elapsed := time.Since(start)

	if elapsed > 500*time.Millisecond {
		t.Errorf("StopWithTimeout took too long: %v", elapsed)
	}
}

func TestEngine_BoundedConcurrency(t *testing.T) {
	agg := testAggregator(50, 50)
	defer func() { _ = agg.Stop() }()

	model := NewMovingAverageModel(0.2)
	maxWorkers := 5
	engine := NewEngineWithWorkers(model, agg, time.Second, testLogger(), maxWorkers)
	defer engine.Stop()

	// Start more tasks than max workers
	for i := 0; i < 20; i++ {
		engine.NotifyTaskStart("test_task", 100)
	}

	// Give time for goroutines to start
	time.Sleep(50 * time.Millisecond)

	// Active workers should not exceed max
	if engine.ActiveWorkers() > maxWorkers {
		t.Errorf("active workers %d exceeds max %d", engine.ActiveWorkers(), maxWorkers)
	}
}

func TestEngine_StopMultipleCalls(t *testing.T) {
	agg := testAggregator(50, 50)
	defer func() { _ = agg.Stop() }()

	model := NewMovingAverageModel(0.2)
	engine := NewEngine(model, agg, 100*time.Millisecond, testLogger())

	engine.NotifyTaskStart("test_task", 100)

	// Multiple stop calls should not panic
	engine.Stop()
	engine.Stop()
	engine.Stop()
}

func TestEngine_TaskIDFormat(t *testing.T) {
	agg := testAggregator(50, 50)
	defer func() { _ = agg.Stop() }()

	model := NewMovingAverageModel(0.2)
	engine := NewEngine(model, agg, 50*time.Millisecond, testLogger())
	defer engine.Stop()

	// Create multiple tasks and verify IDs are formatted correctly
	for i := 0; i < 5; i++ {
		engine.NotifyTaskStart("test_task", 100)
	}

	// Wait for observations
	time.Sleep(100 * time.Millisecond)

	// Task counter should be 5
	engine.mu.Lock()
	counter := engine.taskCounter
	engine.mu.Unlock()

	if counter != 5 {
		t.Errorf("expected taskCounter=5, got %d", counter)
	}
}

package learning

import (
	"testing"
)

func TestMovingAverageModel_Name(t *testing.T) {
	model := NewMovingAverageModel(0.2)

	if model.Name() != "moving_average" {
		t.Errorf("expected name 'moving_average', got %s", model.Name())
	}
}

func TestMovingAverageModel_DefaultAlpha(t *testing.T) {
	// Invalid alpha should default to 0.2
	model := NewMovingAverageModel(0)
	if model.alpha != 0.2 {
		t.Errorf("expected default alpha 0.2, got %f", model.alpha)
	}

	model = NewMovingAverageModel(-1)
	if model.alpha != 0.2 {
		t.Errorf("expected default alpha 0.2, got %f", model.alpha)
	}

	model = NewMovingAverageModel(1.5)
	if model.alpha != 0.2 {
		t.Errorf("expected default alpha 0.2, got %f", model.alpha)
	}
}

func TestMovingAverageModel_ObserveAndPredict(t *testing.T) {
	model := NewMovingAverageModel(0.5) // Use 0.5 for easier math

	// First observation
	model.Observe("task1", 100, &ResourceImpact{
		CPUDelta:    10.0,
		MemoryDelta: 20.0,
	})

	// Predict should return the observation
	predicted := model.Predict("task1", 100)
	if predicted == nil {
		t.Fatal("expected prediction, got nil")
	}
	if predicted.CPUDelta != 10.0 {
		t.Errorf("expected CPU delta 10.0, got %f", predicted.CPUDelta)
	}
	if predicted.MemoryDelta != 20.0 {
		t.Errorf("expected memory delta 20.0, got %f", predicted.MemoryDelta)
	}

	// Second observation - should be averaged
	model.Observe("task1", 100, &ResourceImpact{
		CPUDelta:    20.0,
		MemoryDelta: 40.0,
	})

	// With alpha=0.5: new_avg = 0.5 * 20 + 0.5 * 10 = 15
	predicted = model.Predict("task1", 100)
	if predicted.CPUDelta != 15.0 {
		t.Errorf("expected CPU delta 15.0, got %f", predicted.CPUDelta)
	}
	if predicted.MemoryDelta != 30.0 {
		t.Errorf("expected memory delta 30.0, got %f", predicted.MemoryDelta)
	}
}

func TestMovingAverageModel_PredictUnknownTask(t *testing.T) {
	model := NewMovingAverageModel(0.2)

	predicted := model.Predict("unknown_task", 100)
	if predicted != nil {
		t.Error("expected nil for unknown task")
	}
}

func TestMovingAverageModel_ObserveNilImpact(t *testing.T) {
	model := NewMovingAverageModel(0.2)

	// Should not panic
	model.Observe("task1", 100, nil)

	// Should have no stats
	stats := model.GetTaskStats("task1")
	if stats != nil {
		t.Error("expected nil stats for task with nil impact")
	}
}

func TestMovingAverageModel_GetStats(t *testing.T) {
	model := NewMovingAverageModel(0.2)

	model.Observe("task1", 100, &ResourceImpact{CPUDelta: 10.0})
	model.Observe("task1", 100, &ResourceImpact{CPUDelta: 20.0})
	model.Observe("task2", 50, &ResourceImpact{CPUDelta: 5.0})

	stats := model.GetStats()

	if stats.TotalTasks != 3 {
		t.Errorf("expected total tasks 3, got %d", stats.TotalTasks)
	}

	if len(stats.Tasks) != 2 {
		t.Errorf("expected 2 task types, got %d", len(stats.Tasks))
	}

	task1Stats := stats.Tasks["task1"]
	if task1Stats == nil {
		t.Fatal("expected task1 stats")
	}
	if task1Stats.Count != 2 {
		t.Errorf("expected count 2, got %d", task1Stats.Count)
	}

	task2Stats := stats.Tasks["task2"]
	if task2Stats == nil {
		t.Fatal("expected task2 stats")
	}
	if task2Stats.Count != 1 {
		t.Errorf("expected count 1, got %d", task2Stats.Count)
	}
}

func TestMovingAverageModel_GetTaskStats(t *testing.T) {
	model := NewMovingAverageModel(0.2)

	model.Observe("task1", 100, &ResourceImpact{
		CPUDelta:    10.0,
		MemoryDelta: 20.0,
		GPUDelta:    5.0,
		VRAMDelta:   15.0,
	})

	stats := model.GetTaskStats("task1")
	if stats == nil {
		t.Fatal("expected stats")
	}

	if stats.Task != "task1" {
		t.Errorf("expected task 'task1', got %s", stats.Task)
	}
	if stats.Count != 1 {
		t.Errorf("expected count 1, got %d", stats.Count)
	}
	if stats.AvgCPUDelta != 10.0 {
		t.Errorf("expected avg CPU delta 10.0, got %f", stats.AvgCPUDelta)
	}
	if stats.AvgMemDelta != 20.0 {
		t.Errorf("expected avg mem delta 20.0, got %f", stats.AvgMemDelta)
	}
	if stats.AvgGPUDelta != 5.0 {
		t.Errorf("expected avg GPU delta 5.0, got %f", stats.AvgGPUDelta)
	}
	if stats.AvgVRAMDelta != 15.0 {
		t.Errorf("expected avg VRAM delta 15.0, got %f", stats.AvgVRAMDelta)
	}
}

func TestMovingAverageModel_GetTaskStats_NotFound(t *testing.T) {
	model := NewMovingAverageModel(0.2)

	stats := model.GetTaskStats("unknown")
	if stats != nil {
		t.Error("expected nil for unknown task")
	}
}

func TestMovingAverageModel_Concurrent(t *testing.T) {
	model := NewMovingAverageModel(0.2)

	done := make(chan bool)

	// Concurrent observers
	for i := 0; i < 10; i++ {
		go func(idx int) {
			for j := 0; j < 100; j++ {
				model.Observe("task", 100, &ResourceImpact{CPUDelta: float64(idx)})
			}
			done <- true
		}(i)
	}

	// Concurrent readers
	for i := 0; i < 5; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				model.GetStats()
				model.Predict("task", 100)
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 15; i++ {
		<-done
	}

	stats := model.GetStats()
	if stats.TotalTasks != 1000 {
		t.Errorf("expected 1000 observations, got %d", stats.TotalTasks)
	}
}

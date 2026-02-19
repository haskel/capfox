package strategy

import (
	"testing"
	"time"

	"github.com/haskel/capfox/internal/decision"
	"github.com/haskel/capfox/internal/monitor"
)

func TestQueueAwareStrategy_Name(t *testing.T) {
	m := newMockModel("test", nil, 0)
	s := NewQueueAwareStrategy(m, 5, nil)
	if s.Name() != "queue_aware" {
		t.Errorf("expected name 'queue_aware', got '%s'", s.Name())
	}
}

func TestQueueAwareStrategy_Decide_NilContext(t *testing.T) {
	m := newMockModel("test", nil, 0)
	s := NewQueueAwareStrategy(m, 5, nil)
	result := s.Decide(nil)

	if !result.Allowed {
		t.Error("expected allowed=true for nil context")
	}
}

func TestQueueAwareStrategy_Decide_FallbackOnNoData(t *testing.T) {
	m := newMockModel("test", nil, 0) // Zero confidence
	s := NewQueueAwareStrategy(m, 5, nil)

	ctx := decision.NewContext("test", 100).
		WithCurrentState(&monitor.SystemState{
			CPU:    monitor.CPUState{UsagePercent: 50.0},
			Memory: monitor.MemoryState{UsagePercent: 40.0},
		}).
		WithThresholds(&decision.ThresholdsConfig{
			CPU:     decision.CPUThreshold{MaxPercent: 80.0},
			Memory:  decision.MemoryThreshold{MaxPercent: 80.0},
			Storage: decision.StorageThreshold{MinFreeGB: 0},
		})

	result := s.Decide(ctx)

	if !result.Allowed {
		t.Error("expected allowed=true (fallback to threshold)")
	}
	if !containsReason(result.Reasons, decision.ReasonInsufficientData) {
		t.Error("expected ReasonInsufficientData in reasons")
	}
}

func TestQueueAwareStrategy_Decide_NoPendingTasks(t *testing.T) {
	prediction := &decision.ResourceImpact{
		CPUDelta:    10.0,
		MemoryDelta: 10.0,
	}
	m := newMockModel("test", prediction, 0.9)
	s := NewQueueAwareStrategy(m, 5, nil)

	ctx := decision.NewContext("test", 100).
		WithCurrentState(&monitor.SystemState{
			CPU:    monitor.CPUState{UsagePercent: 50.0},
			Memory: monitor.MemoryState{UsagePercent: 40.0},
		}).
		WithThresholds(&decision.ThresholdsConfig{
			CPU:     decision.CPUThreshold{MaxPercent: 80.0},
			Memory:  decision.MemoryThreshold{MaxPercent: 80.0},
			Storage: decision.StorageThreshold{MinFreeGB: 0},
		}).
		WithPrediction(prediction).
		WithPendingTasks(nil) // No pending tasks

	result := s.Decide(ctx)

	if !result.Allowed {
		t.Error("expected allowed=true")
	}
	// 50 + 10 = 60% (no pending tasks)
	if result.PredictedState.CPUPercent != 60.0 {
		t.Errorf("expected CPU 60%%, got %f%%", result.PredictedState.CPUPercent)
	}
}

func TestQueueAwareStrategy_Decide_WithPendingTasks(t *testing.T) {
	newTaskPrediction := &decision.ResourceImpact{
		CPUDelta:    10.0,
		MemoryDelta: 10.0,
	}
	m := newMockModel("test", newTaskPrediction, 0.9)
	s := NewQueueAwareStrategy(m, 5, nil)

	pendingTasks := []decision.PendingTask{
		{
			Task:       "pending_task_1",
			Complexity: 100,
			StartedAt:  time.Now(),
			Predicted: &decision.ResourceImpact{
				CPUDelta:    5.0,
				MemoryDelta: 5.0,
			},
		},
		{
			Task:       "pending_task_2",
			Complexity: 200,
			StartedAt:  time.Now(),
			Predicted: &decision.ResourceImpact{
				CPUDelta:    10.0,
				MemoryDelta: 5.0,
			},
		},
	}

	ctx := decision.NewContext("new_task", 100).
		WithCurrentState(&monitor.SystemState{
			CPU:    monitor.CPUState{UsagePercent: 50.0},
			Memory: monitor.MemoryState{UsagePercent: 40.0},
		}).
		WithThresholds(&decision.ThresholdsConfig{
			CPU:     decision.CPUThreshold{MaxPercent: 80.0},
			Memory:  decision.MemoryThreshold{MaxPercent: 80.0},
			Storage: decision.StorageThreshold{MinFreeGB: 0},
		}).
		WithPrediction(newTaskPrediction).
		WithPendingTasks(pendingTasks)

	result := s.Decide(ctx)

	// Current: 50%
	// Pending tasks: 5 + 10 = 15%
	// New task: 10%
	// Total: 50 + 15 + 10 = 75% < 80%
	if !result.Allowed {
		t.Errorf("expected allowed=true, got reasons: %v", result.Reasons)
	}
	if result.PredictedState.CPUPercent != 75.0 {
		t.Errorf("expected CPU 75%%, got %f%%", result.PredictedState.CPUPercent)
	}
}

func TestQueueAwareStrategy_Decide_RejectsWithPendingTasks(t *testing.T) {
	newTaskPrediction := &decision.ResourceImpact{
		CPUDelta:    10.0,
		MemoryDelta: 10.0,
	}
	m := newMockModel("test", newTaskPrediction, 0.9)
	s := NewQueueAwareStrategy(m, 5, nil)

	// Heavy pending tasks
	pendingTasks := []decision.PendingTask{
		{
			Task:       "heavy_task",
			Complexity: 500,
			StartedAt:  time.Now(),
			Predicted: &decision.ResourceImpact{
				CPUDelta:    25.0,
				MemoryDelta: 20.0,
			},
		},
	}

	ctx := decision.NewContext("new_task", 100).
		WithCurrentState(&monitor.SystemState{
			CPU:    monitor.CPUState{UsagePercent: 50.0},
			Memory: monitor.MemoryState{UsagePercent: 40.0},
		}).
		WithThresholds(&decision.ThresholdsConfig{
			CPU:     decision.CPUThreshold{MaxPercent: 80.0},
			Memory:  decision.MemoryThreshold{MaxPercent: 80.0},
			Storage: decision.StorageThreshold{MinFreeGB: 0},
		}).
		WithPrediction(newTaskPrediction).
		WithPendingTasks(pendingTasks)

	result := s.Decide(ctx)

	// Current: 50%
	// Pending tasks: 25%
	// New task: 10%
	// Total: 50 + 25 + 10 = 85% > 80%
	if result.Allowed {
		t.Error("expected allowed=false with pending tasks")
	}
	if !containsReason(result.Reasons, decision.ReasonCPUOverload) {
		t.Error("expected ReasonCPUOverload in reasons")
	}
	if result.PredictedState.CPUPercent != 85.0 {
		t.Errorf("expected CPU 85%%, got %f%%", result.PredictedState.CPUPercent)
	}
}

func TestQueueAwareStrategy_Decide_UsesModelForPendingWithoutPrediction(t *testing.T) {
	// Model returns this prediction for all tasks
	prediction := &decision.ResourceImpact{
		CPUDelta:    10.0,
		MemoryDelta: 10.0,
	}
	m := newMockModel("test", prediction, 0.9)
	s := NewQueueAwareStrategy(m, 5, nil)

	// Pending task without stored prediction
	pendingTasks := []decision.PendingTask{
		{
			Task:       "pending_task",
			Complexity: 100,
			StartedAt:  time.Now(),
			Predicted:  nil, // No stored prediction
		},
	}

	ctx := decision.NewContext("new_task", 100).
		WithCurrentState(&monitor.SystemState{
			CPU:    monitor.CPUState{UsagePercent: 50.0},
			Memory: monitor.MemoryState{UsagePercent: 40.0},
		}).
		WithThresholds(&decision.ThresholdsConfig{
			CPU:     decision.CPUThreshold{MaxPercent: 80.0},
			Memory:  decision.MemoryThreshold{MaxPercent: 80.0},
			Storage: decision.StorageThreshold{MinFreeGB: 0},
		}).
		WithPrediction(prediction).
		WithPendingTasks(pendingTasks)

	result := s.Decide(ctx)

	// Current: 50%
	// Pending tasks (from model): 10%
	// New task: 10%
	// Total: 50 + 10 + 10 = 70% < 80%
	if !result.Allowed {
		t.Errorf("expected allowed=true, got reasons: %v", result.Reasons)
	}
	if result.PredictedState.CPUPercent != 70.0 {
		t.Errorf("expected CPU 70%%, got %f%%", result.PredictedState.CPUPercent)
	}
}

func TestQueueAwareStrategy_Decide_MultiplePendingTasks(t *testing.T) {
	newTaskPrediction := &decision.ResourceImpact{
		CPUDelta:    5.0,
		MemoryDelta: 5.0,
	}
	m := newMockModel("test", newTaskPrediction, 0.9)
	s := NewQueueAwareStrategy(m, 5, nil)

	// Multiple pending tasks
	pendingTasks := []decision.PendingTask{
		{
			Task:       "task_1",
			Complexity: 100,
			Predicted: &decision.ResourceImpact{
				CPUDelta:    5.0,
				MemoryDelta: 5.0,
			},
		},
		{
			Task:       "task_2",
			Complexity: 100,
			Predicted: &decision.ResourceImpact{
				CPUDelta:    5.0,
				MemoryDelta: 5.0,
			},
		},
		{
			Task:       "task_3",
			Complexity: 100,
			Predicted: &decision.ResourceImpact{
				CPUDelta:    5.0,
				MemoryDelta: 5.0,
			},
		},
	}

	ctx := decision.NewContext("new_task", 100).
		WithCurrentState(&monitor.SystemState{
			CPU:    monitor.CPUState{UsagePercent: 50.0},
			Memory: monitor.MemoryState{UsagePercent: 40.0},
		}).
		WithThresholds(&decision.ThresholdsConfig{
			CPU:     decision.CPUThreshold{MaxPercent: 80.0},
			Memory:  decision.MemoryThreshold{MaxPercent: 80.0},
			Storage: decision.StorageThreshold{MinFreeGB: 0},
		}).
		WithPrediction(newTaskPrediction).
		WithPendingTasks(pendingTasks)

	result := s.Decide(ctx)

	// Current: 50%
	// Pending tasks: 5 + 5 + 5 = 15%
	// New task: 5%
	// Total: 50 + 15 + 5 = 70% < 80%
	if !result.Allowed {
		t.Errorf("expected allowed=true, got reasons: %v", result.Reasons)
	}
	if result.PredictedState.CPUPercent != 70.0 {
		t.Errorf("expected CPU 70%%, got %f%%", result.PredictedState.CPUPercent)
	}
}

func TestQueueAwareStrategy_Decide_GPUWithPendingTasks(t *testing.T) {
	newTaskPrediction := &decision.ResourceImpact{
		CPUDelta:    5.0,
		MemoryDelta: 5.0,
		GPUDelta:    15.0,
		VRAMDelta:   10.0,
	}
	m := newMockModel("test", newTaskPrediction, 0.9)
	s := NewQueueAwareStrategy(m, 5, nil)

	pendingTasks := []decision.PendingTask{
		{
			Task:       "gpu_task",
			Complexity: 200,
			Predicted: &decision.ResourceImpact{
				CPUDelta:    5.0,
				MemoryDelta: 5.0,
				GPUDelta:    20.0,
				VRAMDelta:   15.0,
			},
		},
	}

	ctx := decision.NewContext("new_task", 100).
		WithCurrentState(&monitor.SystemState{
			CPU:    monitor.CPUState{UsagePercent: 50.0},
			Memory: monitor.MemoryState{UsagePercent: 40.0},
			GPUs: []monitor.GPUState{
				{UsagePercent: 50.0, VRAMUsedBytes: 5000, VRAMTotalBytes: 10000},
			},
		}).
		WithThresholds(&decision.ThresholdsConfig{
			CPU:     decision.CPUThreshold{MaxPercent: 90.0},
			Memory:  decision.MemoryThreshold{MaxPercent: 90.0},
			GPU:     decision.GPUThreshold{MaxPercent: 80.0},
			VRAM:    decision.VRAMThreshold{MaxPercent: 90.0},
			Storage: decision.StorageThreshold{MinFreeGB: 0},
		}).
		WithPrediction(newTaskPrediction).
		WithPendingTasks(pendingTasks)

	result := s.Decide(ctx)

	// GPU: 50 + 20 + 15 = 85% > 80%
	if result.Allowed {
		t.Error("expected allowed=false for GPU overload")
	}
	if !containsReason(result.Reasons, decision.ReasonGPUOverload) {
		t.Error("expected ReasonGPUOverload in reasons")
	}
	if result.PredictedState.GPUPercent != 85.0 {
		t.Errorf("expected GPU 85%%, got %f%%", result.PredictedState.GPUPercent)
	}
}

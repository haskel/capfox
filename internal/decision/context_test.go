package decision

import (
	"testing"
	"time"

	"github.com/haskel/capfox/internal/monitor"
)

func TestNewContext(t *testing.T) {
	ctx := NewContext("video_encoding", 150)

	if ctx.Task != "video_encoding" {
		t.Errorf("expected task 'video_encoding', got '%s'", ctx.Task)
	}
	if ctx.Complexity != 150 {
		t.Errorf("expected complexity 150, got %d", ctx.Complexity)
	}
}

func TestContextBuilderPattern(t *testing.T) {
	state := &monitor.SystemState{
		CPU: monitor.CPUState{UsagePercent: 50.0},
	}
	prediction := &ResourceImpact{CPUDelta: 10.0}
	thresholds := &ThresholdsConfig{
		CPU: CPUThreshold{MaxPercent: 80.0},
	}
	resources := &ResourceEstimate{CPU: 100}
	pendingTasks := []PendingTask{
		{Task: "task1", Complexity: 50, StartedAt: time.Now()},
	}

	ctx := NewContext("test_task", 100).
		WithCurrentState(state).
		WithPrediction(prediction).
		WithThresholds(thresholds).
		WithSafetyBuffer(0.1).
		WithResources(resources).
		WithPendingTasks(pendingTasks)

	if ctx.CurrentState != state {
		t.Error("CurrentState not set correctly")
	}
	if ctx.Prediction != prediction {
		t.Error("Prediction not set correctly")
	}
	if ctx.Thresholds != thresholds {
		t.Error("Thresholds not set correctly")
	}
	if ctx.SafetyBuffer != 0.1 {
		t.Errorf("expected SafetyBuffer 0.1, got %f", ctx.SafetyBuffer)
	}
	if ctx.Resources != resources {
		t.Error("Resources not set correctly")
	}
	if len(ctx.PendingTasks) != 1 {
		t.Errorf("expected 1 pending task, got %d", len(ctx.PendingTasks))
	}
}

func TestReasonConstants(t *testing.T) {
	reasons := []Reason{
		ReasonCPUOverload,
		ReasonMemoryOverload,
		ReasonGPUOverload,
		ReasonVRAMOverload,
		ReasonStorageLow,
		ReasonInsufficientData,
	}

	expectedStrings := []string{
		"cpu_overload",
		"memory_overload",
		"gpu_overload",
		"vram_overload",
		"storage_low",
		"insufficient_data",
	}

	for i, r := range reasons {
		if string(r) != expectedStrings[i] {
			t.Errorf("expected '%s', got '%s'", expectedStrings[i], string(r))
		}
	}
}

func TestResultStructure(t *testing.T) {
	result := &Result{
		Allowed:    true,
		Reasons:    []Reason{},
		Confidence: 0.95,
		Strategy:   "predictive",
		Model:      "linear",
		PredictedState: &FutureState{
			CPUPercent:    75.0,
			MemoryPercent: 60.0,
		},
	}

	if !result.Allowed {
		t.Error("expected Allowed to be true")
	}
	if result.Confidence != 0.95 {
		t.Errorf("expected Confidence 0.95, got %f", result.Confidence)
	}
	if result.PredictedState.CPUPercent != 75.0 {
		t.Errorf("expected CPUPercent 75.0, got %f", result.PredictedState.CPUPercent)
	}
}

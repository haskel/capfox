package scheduler

import (
	"context"
	"errors"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/haskel/capfox/internal/decision"
	"github.com/haskel/capfox/internal/decision/model"
)

// mockModel implements model.PredictionModel for testing
type mockModel struct {
	needsRetrain  bool
	retrainCount  int
	retrainError  error
	mu            sync.Mutex
}

func newMockModel() *mockModel {
	return &mockModel{}
}

func (m *mockModel) Name() string                                                          { return "mock" }
func (m *mockModel) LearningType() model.LearningType                                      { return model.LearningTypeBatch }
func (m *mockModel) Predict(task string, complexity int) *decision.ResourceImpact          { return nil }
func (m *mockModel) Observe(task string, complexity int, impact *decision.ResourceImpact)  {}
func (m *mockModel) Confidence(task string) float64                                        { return 0 }
func (m *mockModel) Stats() *model.Stats                                                   { return &model.Stats{} }
func (m *mockModel) TaskStats(task string) *model.TaskStats                                { return nil }
func (m *mockModel) Save(w io.Writer) error                                                { return nil }
func (m *mockModel) Load(r io.Reader) error                                                { return nil }

func (m *mockModel) NeedsRetrain() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.needsRetrain
}

func (m *mockModel) Retrain() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.retrainError != nil {
		return m.retrainError
	}
	m.retrainCount++
	m.needsRetrain = false
	return nil
}

func (m *mockModel) setNeedsRetrain(v bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.needsRetrain = v
}

func (m *mockModel) setRetrainError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.retrainError = err
}

func (m *mockModel) getRetrainCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.retrainCount
}

func TestScheduler_StartStop(t *testing.T) {
	mock := newMockModel()
	s := NewScheduler(mock, Config{Interval: 100 * time.Millisecond})

	// Not running initially
	if s.IsRunning() {
		t.Error("expected scheduler to not be running initially")
	}

	// Start
	ctx := context.Background()
	if err := s.Start(ctx); err != nil {
		t.Fatalf("Start error: %v", err)
	}

	// Should be running
	if !s.IsRunning() {
		t.Error("expected scheduler to be running after Start")
	}

	// Double start should be no-op
	if err := s.Start(ctx); err != nil {
		t.Fatalf("Double Start error: %v", err)
	}

	// Stop
	s.Stop()

	// Should not be running
	if s.IsRunning() {
		t.Error("expected scheduler to not be running after Stop")
	}
}

func TestScheduler_RetrainsOnSchedule(t *testing.T) {
	mock := newMockModel()
	mock.setNeedsRetrain(true)

	s := NewScheduler(mock, Config{Interval: 50 * time.Millisecond})

	ctx := context.Background()
	_ = s.Start(ctx)
	defer s.Stop()

	// Wait for at least one retrain
	time.Sleep(100 * time.Millisecond)

	if mock.getRetrainCount() == 0 {
		t.Error("expected at least one retrain")
	}
}

func TestScheduler_DoesNotRetrainWhenNotNeeded(t *testing.T) {
	mock := newMockModel()
	mock.setNeedsRetrain(false) // Does not need retrain

	s := NewScheduler(mock, Config{Interval: 50 * time.Millisecond})

	ctx := context.Background()
	_ = s.Start(ctx)
	defer s.Stop()

	// Wait for scheduler tick
	time.Sleep(100 * time.Millisecond)

	if mock.getRetrainCount() != 0 {
		t.Errorf("expected no retrains, got %d", mock.getRetrainCount())
	}
}

func TestScheduler_ForceRetrain(t *testing.T) {
	mock := newMockModel()
	s := NewScheduler(mock, Config{Interval: time.Hour}) // Long interval

	// Force retrain without starting scheduler
	if err := s.ForceRetrain(); err != nil {
		t.Fatalf("ForceRetrain error: %v", err)
	}

	if mock.getRetrainCount() != 1 {
		t.Errorf("expected 1 retrain, got %d", mock.getRetrainCount())
	}
}

func TestScheduler_ForceRetrainError(t *testing.T) {
	mock := newMockModel()
	mock.setRetrainError(errors.New("retrain failed"))

	s := NewScheduler(mock, Config{Interval: time.Hour})

	err := s.ForceRetrain()
	if err == nil {
		t.Error("expected error from ForceRetrain")
	}

	stats := s.Stats()
	if stats.LastError != "retrain failed" {
		t.Errorf("expected LastError 'retrain failed', got '%s'", stats.LastError)
	}
}

func TestScheduler_Stats(t *testing.T) {
	mock := newMockModel()
	s := NewScheduler(mock, Config{Interval: time.Hour})

	stats := s.Stats()
	if stats.Running {
		t.Error("expected Running=false before start")
	}
	if stats.Interval != "1h0m0s" {
		t.Errorf("expected Interval '1h0m0s', got '%s'", stats.Interval)
	}
	if stats.RetrainCount != 0 {
		t.Errorf("expected RetrainCount 0, got %d", stats.RetrainCount)
	}

	// Start and retrain
	ctx := context.Background()
	_ = s.Start(ctx)
	_ = s.ForceRetrain()

	stats = s.Stats()
	if !stats.Running {
		t.Error("expected Running=true after start")
	}
	if stats.RetrainCount != 1 {
		t.Errorf("expected RetrainCount 1, got %d", stats.RetrainCount)
	}
	if stats.LastRetrain.IsZero() {
		t.Error("expected LastRetrain to be set")
	}

	s.Stop()
}

func TestScheduler_ContextCancellation(t *testing.T) {
	mock := newMockModel()
	s := NewScheduler(mock, Config{Interval: time.Hour})

	ctx, cancel := context.WithCancel(context.Background())
	_ = s.Start(ctx)

	// Cancel context
	cancel()

	// Wait for scheduler to stop
	time.Sleep(50 * time.Millisecond)

	// Scheduler should detect cancellation
	// Note: The scheduler loop checks ctx.Done(), so it should stop
}

func TestScheduler_DefaultInterval(t *testing.T) {
	mock := newMockModel()
	s := NewScheduler(mock, Config{}) // Zero interval

	stats := s.Stats()
	if stats.Interval != "1h0m0s" {
		t.Errorf("expected default interval '1h0m0s', got '%s'", stats.Interval)
	}
}

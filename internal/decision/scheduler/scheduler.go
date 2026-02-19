package scheduler

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/haskel/capfox/internal/decision/model"
)

// Scheduler periodically retrains batch learning models.
type Scheduler struct {
	model    model.PredictionModel
	interval time.Duration
	mu       sync.RWMutex
	running  bool
	stopCh   chan struct{}
	doneCh   chan struct{}
	logger   *log.Logger

	// Stats
	retrainCount int64
	lastRetrain  time.Time
	lastError    error
}

// Config holds scheduler configuration.
type Config struct {
	Interval time.Duration
	Logger   *log.Logger
}

// NewScheduler creates a new model scheduler.
func NewScheduler(m model.PredictionModel, cfg Config) *Scheduler {
	interval := cfg.Interval
	if interval == 0 {
		interval = time.Hour
	}

	return &Scheduler{
		model:    m,
		interval: interval,
		logger:   cfg.Logger,
	}
}

// Start begins the scheduler loop.
func (s *Scheduler) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return nil // Already running
	}
	s.running = true
	s.stopCh = make(chan struct{})
	s.doneCh = make(chan struct{})
	s.mu.Unlock()

	go s.run(ctx)
	return nil
}

// Stop stops the scheduler and waits for it to finish.
func (s *Scheduler) Stop() {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return
	}
	s.running = false
	close(s.stopCh)
	s.mu.Unlock()

	<-s.doneCh
}

// run is the main scheduler loop.
func (s *Scheduler) run(ctx context.Context) {
	defer close(s.doneCh)

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.checkAndRetrain()
		}
	}
}

// checkAndRetrain checks if the model needs retraining and retrains if necessary.
func (s *Scheduler) checkAndRetrain() {
	// Only retrain batch models that need it
	if !s.model.NeedsRetrain() {
		return
	}

	if err := s.model.Retrain(); err != nil {
		s.mu.Lock()
		s.lastError = err
		s.mu.Unlock()

		if s.logger != nil {
			s.logger.Printf("Model retrain error: %v", err)
		}
		return
	}

	s.mu.Lock()
	s.retrainCount++
	s.lastRetrain = time.Now()
	s.lastError = nil
	s.mu.Unlock()

	if s.logger != nil {
		s.logger.Printf("Model retrained successfully (count: %d)", s.retrainCount)
	}
}

// ForceRetrain triggers an immediate retrain regardless of schedule.
func (s *Scheduler) ForceRetrain() error {
	if err := s.model.Retrain(); err != nil {
		s.mu.Lock()
		s.lastError = err
		s.mu.Unlock()
		return err
	}

	s.mu.Lock()
	s.retrainCount++
	s.lastRetrain = time.Now()
	s.lastError = nil
	s.mu.Unlock()

	return nil
}

// Stats returns scheduler statistics.
type Stats struct {
	Running      bool      `json:"running"`
	Interval     string    `json:"interval"`
	RetrainCount int64     `json:"retrain_count"`
	LastRetrain  time.Time `json:"last_retrain,omitempty"`
	LastError    string    `json:"last_error,omitempty"`
}

// Stats returns current scheduler statistics.
func (s *Scheduler) Stats() Stats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := Stats{
		Running:      s.running,
		Interval:     s.interval.String(),
		RetrainCount: s.retrainCount,
		LastRetrain:  s.lastRetrain,
	}

	if s.lastError != nil {
		stats.LastError = s.lastError.Error()
	}

	return stats
}

// IsRunning returns whether the scheduler is running.
func (s *Scheduler) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

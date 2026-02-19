package storage

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Data represents the persisted data structure.
type Data struct {
	Version    int                    `json:"version"`
	UpdatedAt  time.Time              `json:"updated_at"`
	TaskStats  map[string]*TaskData   `json:"task_stats"`
}

// TaskData represents persisted statistics for a task type.
type TaskData struct {
	Task         string  `json:"task"`
	Count        int64   `json:"count"`
	AvgCPUDelta  float64 `json:"avg_cpu_delta"`
	AvgMemDelta  float64 `json:"avg_mem_delta"`
	AvgGPUDelta  float64 `json:"avg_gpu_delta,omitempty"`
	AvgVRAMDelta float64 `json:"avg_vram_delta,omitempty"`
}

const (
	currentVersion = 1
	dataFileName   = "capfox_data.json"
)

// Storage handles persistence of learning data.
type Storage struct {
	dataDir       string
	flushInterval time.Duration
	logger        *slog.Logger

	mu      sync.RWMutex
	data    *Data
	dirty   bool
	cancel  context.CancelFunc
	done    chan struct{}
}

// New creates a new Storage instance.
func New(dataDir string, flushInterval time.Duration, logger *slog.Logger) *Storage {
	return &Storage{
		dataDir:       dataDir,
		flushInterval: flushInterval,
		logger:        logger,
		data:          newEmptyData(),
		done:          make(chan struct{}),
	}
}

func newEmptyData() *Data {
	return &Data{
		Version:   currentVersion,
		UpdatedAt: time.Now(),
		TaskStats: make(map[string]*TaskData),
	}
}

// Load loads data from disk. If file doesn't exist, returns empty data.
func (s *Storage) Load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	filePath := filepath.Join(s.dataDir, dataFileName)

	file, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			s.logger.Info("no existing data file, starting fresh", "path", filePath)
			s.data = newEmptyData()
			return nil
		}
		return err
	}
	defer file.Close()

	var data Data
	if err := json.NewDecoder(file).Decode(&data); err != nil {
		s.logger.Warn("failed to decode data file, starting fresh", "error", err)
		s.data = newEmptyData()
		return nil
	}

	// Validate version
	if data.Version > currentVersion {
		s.logger.Warn("data file version is newer than supported, starting fresh",
			"file_version", data.Version,
			"supported_version", currentVersion,
		)
		s.data = newEmptyData()
		return nil
	}

	// Ensure map is initialized
	if data.TaskStats == nil {
		data.TaskStats = make(map[string]*TaskData)
	}

	s.data = &data
	s.logger.Info("loaded data from disk",
		"path", filePath,
		"tasks", len(data.TaskStats),
	)

	return nil
}

// Save saves data to disk.
func (s *Storage) Save() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.saveLocked()
}

func (s *Storage) saveLocked() error {
	// Ensure data directory exists
	if err := os.MkdirAll(s.dataDir, 0755); err != nil {
		return err
	}

	filePath := filepath.Join(s.dataDir, dataFileName)
	tempPath := filePath + ".tmp"

	s.data.UpdatedAt = time.Now()

	file, err := os.Create(tempPath)
	if err != nil {
		return err
	}

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(s.data); err != nil {
		file.Close()
		os.Remove(tempPath)
		return err
	}

	if err := file.Close(); err != nil {
		os.Remove(tempPath)
		return err
	}

	// Atomic rename
	if err := os.Rename(tempPath, filePath); err != nil {
		os.Remove(tempPath)
		return err
	}

	s.dirty = false
	s.logger.Debug("saved data to disk", "path", filePath)

	return nil
}

// Start starts the periodic flush goroutine.
func (s *Storage) Start(ctx context.Context) {
	ctx, s.cancel = context.WithCancel(ctx)

	go s.flushLoop(ctx)
}

// Stop stops the periodic flush and saves final state.
func (s *Storage) Stop() error {
	if s.cancel != nil {
		s.cancel()
		<-s.done
	}

	// Final save
	return s.Save()
}

func (s *Storage) flushLoop(ctx context.Context) {
	defer close(s.done)

	ticker := time.NewTicker(s.flushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.mu.RLock()
			dirty := s.dirty
			s.mu.RUnlock()

			if dirty {
				if err := s.Save(); err != nil {
					s.logger.Error("failed to save data", "error", err)
				}
			}
		}
	}
}

// GetTaskStats returns statistics for a specific task.
func (s *Storage) GetTaskStats(task string) *TaskData {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.data.TaskStats[task]
}

// GetAllTaskStats returns statistics for all tasks.
func (s *Storage) GetAllTaskStats() map[string]*TaskData {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Return a copy
	result := make(map[string]*TaskData, len(s.data.TaskStats))
	for k, v := range s.data.TaskStats {
		copied := *v
		result[k] = &copied
	}
	return result
}

// UpdateTaskStats updates statistics for a task.
func (s *Storage) UpdateTaskStats(task string, count int64, cpuDelta, memDelta, gpuDelta, vramDelta float64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.data.TaskStats[task] = &TaskData{
		Task:         task,
		Count:        count,
		AvgCPUDelta:  cpuDelta,
		AvgMemDelta:  memDelta,
		AvgGPUDelta:  gpuDelta,
		AvgVRAMDelta: vramDelta,
	}
	s.dirty = true
}

// MarkDirty marks data as needing to be saved.
func (s *Storage) MarkDirty() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.dirty = true
}

// IsDirty returns whether data has unsaved changes.
func (s *Storage) IsDirty() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.dirty
}

// TaskCount returns the number of tracked tasks.
func (s *Storage) TaskCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.data.TaskStats)
}

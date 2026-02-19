package storage

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

const (
	modelFileName = "capfox_model.json"
)

// ModelStorage handles persistence of prediction models.
type ModelStorage struct {
	storage *Storage
}

// NewModelStorage creates a new ModelStorage.
func NewModelStorage(s *Storage) *ModelStorage {
	return &ModelStorage{storage: s}
}

// Saveable is an interface for objects that can be saved.
type Saveable interface {
	Save(w io.Writer) error
}

// Loadable is an interface for objects that can be loaded.
type Loadable interface {
	Load(r io.Reader) error
}

// SaveModel saves a model to disk.
func (ms *ModelStorage) SaveModel(model Saveable) error {
	ms.storage.mu.Lock()
	defer ms.storage.mu.Unlock()

	return ms.saveModelLocked(model)
}

func (ms *ModelStorage) saveModelLocked(model Saveable) error {
	// Ensure data directory exists
	if err := os.MkdirAll(ms.storage.dataDir, 0755); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}

	filePath := filepath.Join(ms.storage.dataDir, modelFileName)
	tempPath := filePath + ".tmp"

	file, err := os.Create(tempPath)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}

	if err := model.Save(file); err != nil {
		file.Close()
		os.Remove(tempPath)
		return fmt.Errorf("failed to save model: %w", err)
	}

	if err := file.Close(); err != nil {
		os.Remove(tempPath)
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tempPath, filePath); err != nil {
		os.Remove(tempPath)
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	ms.storage.logger.Debug("saved model to disk", "path", filePath)
	return nil
}

// LoadModel loads a model from disk.
func (ms *ModelStorage) LoadModel(model Loadable) error {
	ms.storage.mu.Lock()
	defer ms.storage.mu.Unlock()

	filePath := filepath.Join(ms.storage.dataDir, modelFileName)

	file, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			ms.storage.logger.Info("no existing model file, using fresh model", "path", filePath)
			return nil // Not an error - just no saved state
		}
		return fmt.Errorf("failed to open model file: %w", err)
	}
	defer file.Close()

	if err := model.Load(file); err != nil {
		ms.storage.logger.Warn("failed to load model, using fresh model", "error", err)
		return nil // Not fatal - use fresh model
	}

	ms.storage.logger.Info("loaded model from disk", "path", filePath)
	return nil
}

// ModelExists returns whether a saved model exists.
func (ms *ModelStorage) ModelExists() bool {
	filePath := filepath.Join(ms.storage.dataDir, modelFileName)
	_, err := os.Stat(filePath)
	return err == nil
}

// ModelInfo returns information about the saved model.
type ModelInfo struct {
	Exists    bool      `json:"exists"`
	Path      string    `json:"path"`
	Size      int64     `json:"size,omitempty"`
	UpdatedAt time.Time `json:"updated_at,omitempty"`
}

// GetModelInfo returns information about the saved model.
func (ms *ModelStorage) GetModelInfo() ModelInfo {
	filePath := filepath.Join(ms.storage.dataDir, modelFileName)
	info := ModelInfo{
		Path: filePath,
	}

	stat, err := os.Stat(filePath)
	if err != nil {
		info.Exists = false
		return info
	}

	info.Exists = true
	info.Size = stat.Size()
	info.UpdatedAt = stat.ModTime()
	return info
}

// DeleteModel deletes the saved model file.
func (ms *ModelStorage) DeleteModel() error {
	filePath := filepath.Join(ms.storage.dataDir, modelFileName)
	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete model file: %w", err)
	}
	return nil
}

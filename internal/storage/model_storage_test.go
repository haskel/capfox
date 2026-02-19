package storage

import (
	"encoding/json"
	"io"
	"os"
	"testing"
	"time"

	"log/slog"
)

// mockModel for testing
type mockModel struct {
	Data  string `json:"data"`
	Value int    `json:"value"`
}

func (m *mockModel) Save(w io.Writer) error {
	return json.NewEncoder(w).Encode(m)
}

func (m *mockModel) Load(r io.Reader) error {
	return json.NewDecoder(r).Decode(m)
}

func TestModelStorage_SaveLoad(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "capfox_model_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	storage := New(tmpDir, time.Hour, logger)
	ms := NewModelStorage(storage)

	// Test that model doesn't exist initially
	if ms.ModelExists() {
		t.Error("expected model to not exist initially")
	}

	// Save a model
	original := &mockModel{Data: "test data", Value: 42}
	if err := ms.SaveModel(original); err != nil {
		t.Fatalf("SaveModel error: %v", err)
	}

	// Check that model exists
	if !ms.ModelExists() {
		t.Error("expected model to exist after save")
	}

	// Load into new model
	loaded := &mockModel{}
	if err := ms.LoadModel(loaded); err != nil {
		t.Fatalf("LoadModel error: %v", err)
	}

	// Verify data
	if loaded.Data != original.Data {
		t.Errorf("expected Data '%s', got '%s'", original.Data, loaded.Data)
	}
	if loaded.Value != original.Value {
		t.Errorf("expected Value %d, got %d", original.Value, loaded.Value)
	}
}

func TestModelStorage_LoadNonExistent(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "capfox_model_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	storage := New(tmpDir, time.Hour, logger)
	ms := NewModelStorage(storage)

	// Loading non-existent model should not error
	model := &mockModel{}
	if err := ms.LoadModel(model); err != nil {
		t.Errorf("expected no error loading non-existent model, got: %v", err)
	}

	// Model should still have default values
	if model.Data != "" || model.Value != 0 {
		t.Error("expected model to have default values")
	}
}

func TestModelStorage_GetModelInfo(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "capfox_model_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	storage := New(tmpDir, time.Hour, logger)
	ms := NewModelStorage(storage)

	// Initially no model
	info := ms.GetModelInfo()
	if info.Exists {
		t.Error("expected model to not exist")
	}

	// Save model
	model := &mockModel{Data: "test", Value: 123}
	if err := ms.SaveModel(model); err != nil {
		t.Fatalf("failed to save model: %v", err)
	}

	// Check info
	info = ms.GetModelInfo()
	if !info.Exists {
		t.Error("expected model to exist")
	}
	if info.Size == 0 {
		t.Error("expected non-zero size")
	}
	if info.UpdatedAt.IsZero() {
		t.Error("expected non-zero UpdatedAt")
	}
}

func TestModelStorage_DeleteModel(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "capfox_model_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	storage := New(tmpDir, time.Hour, logger)
	ms := NewModelStorage(storage)

	// Save model
	model := &mockModel{Data: "test", Value: 123}
	if err := ms.SaveModel(model); err != nil {
		t.Fatalf("failed to save model: %v", err)
	}

	if !ms.ModelExists() {
		t.Error("expected model to exist after save")
	}

	// Delete model
	if err := ms.DeleteModel(); err != nil {
		t.Fatalf("DeleteModel error: %v", err)
	}

	if ms.ModelExists() {
		t.Error("expected model to not exist after delete")
	}

	// Delete again should not error
	if err := ms.DeleteModel(); err != nil {
		t.Errorf("expected no error deleting non-existent model, got: %v", err)
	}
}

func TestModelStorage_AtomicWrite(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "capfox_model_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	storage := New(tmpDir, time.Hour, logger)
	ms := NewModelStorage(storage)

	// Save multiple times
	for i := 0; i < 5; i++ {
		model := &mockModel{Data: "test", Value: i}
		if err := ms.SaveModel(model); err != nil {
			t.Fatalf("SaveModel iteration %d error: %v", i, err)
		}
	}

	// Load final state
	model := &mockModel{}
	if err := ms.LoadModel(model); err != nil {
		t.Fatalf("LoadModel error: %v", err)
	}

	if model.Value != 4 {
		t.Errorf("expected Value 4, got %d", model.Value)
	}
}

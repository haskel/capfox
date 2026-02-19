package storage

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
}

func TestStorage_NewEmptyData(t *testing.T) {
	data := newEmptyData()

	if data.Version != currentVersion {
		t.Errorf("expected version %d, got %d", currentVersion, data.Version)
	}

	if data.TaskStats == nil {
		t.Error("expected TaskStats to be initialized")
	}
}

func TestStorage_SaveLoad(t *testing.T) {
	tmpDir := t.TempDir()

	s := New(tmpDir, time.Second, testLogger())

	// Add some data
	s.UpdateTaskStats("task1", 10, 5.0, 10.0, 2.0, 3.0)
	s.UpdateTaskStats("task2", 5, 2.5, 5.0, 0, 0)

	// Save
	if err := s.Save(); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Create new storage and load
	s2 := New(tmpDir, time.Second, testLogger())
	if err := s2.Load(); err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Verify data
	stats := s2.GetTaskStats("task1")
	if stats == nil {
		t.Fatal("expected task1 stats")
	}

	if stats.Count != 10 {
		t.Errorf("expected count 10, got %d", stats.Count)
	}

	if stats.AvgCPUDelta != 5.0 {
		t.Errorf("expected CPU delta 5.0, got %f", stats.AvgCPUDelta)
	}

	// Verify task2
	stats2 := s2.GetTaskStats("task2")
	if stats2 == nil {
		t.Fatal("expected task2 stats")
	}

	if stats2.Count != 5 {
		t.Errorf("expected count 5, got %d", stats2.Count)
	}
}

func TestStorage_LoadNonExistent(t *testing.T) {
	tmpDir := t.TempDir()

	s := New(tmpDir, time.Second, testLogger())

	// Load from non-existent file should not error
	if err := s.Load(); err != nil {
		t.Fatalf("Load should not fail for non-existent file: %v", err)
	}

	// Should have empty data
	if s.TaskCount() != 0 {
		t.Errorf("expected 0 tasks, got %d", s.TaskCount())
	}
}

func TestStorage_LoadCorruptedFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Write corrupted data
	filePath := filepath.Join(tmpDir, dataFileName)
	if err := os.WriteFile(filePath, []byte("not json"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	s := New(tmpDir, time.Second, testLogger())

	// Load should not error, just start fresh
	if err := s.Load(); err != nil {
		t.Fatalf("Load should not fail for corrupted file: %v", err)
	}

	// Should have empty data
	if s.TaskCount() != 0 {
		t.Errorf("expected 0 tasks, got %d", s.TaskCount())
	}
}

func TestStorage_GetAllTaskStats(t *testing.T) {
	tmpDir := t.TempDir()
	s := New(tmpDir, time.Second, testLogger())

	s.UpdateTaskStats("task1", 10, 5.0, 10.0, 0, 0)
	s.UpdateTaskStats("task2", 5, 2.5, 5.0, 0, 0)

	all := s.GetAllTaskStats()

	if len(all) != 2 {
		t.Errorf("expected 2 tasks, got %d", len(all))
	}

	if all["task1"] == nil {
		t.Error("expected task1")
	}

	if all["task2"] == nil {
		t.Error("expected task2")
	}
}

func TestStorage_IsDirty(t *testing.T) {
	tmpDir := t.TempDir()
	s := New(tmpDir, time.Second, testLogger())

	if s.IsDirty() {
		t.Error("expected not dirty initially")
	}

	s.UpdateTaskStats("task1", 1, 1.0, 1.0, 0, 0)

	if !s.IsDirty() {
		t.Error("expected dirty after update")
	}

	if err := s.Save(); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	if s.IsDirty() {
		t.Error("expected not dirty after save")
	}
}

func TestStorage_MarkDirty(t *testing.T) {
	tmpDir := t.TempDir()
	s := New(tmpDir, time.Second, testLogger())

	if s.IsDirty() {
		t.Error("expected not dirty")
	}

	s.MarkDirty()

	if !s.IsDirty() {
		t.Error("expected dirty after MarkDirty")
	}
}

func TestStorage_PeriodicFlush(t *testing.T) {
	tmpDir := t.TempDir()
	s := New(tmpDir, 50*time.Millisecond, testLogger())

	ctx := context.Background()
	s.Start(ctx)

	// Add data
	s.UpdateTaskStats("task1", 1, 1.0, 1.0, 0, 0)

	// Wait for flush
	time.Sleep(100 * time.Millisecond)

	// Stop storage
	if err := s.Stop(); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}

	// Verify file was created
	filePath := filepath.Join(tmpDir, dataFileName)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Error("expected data file to be created")
	}

	// Load and verify
	s2 := New(tmpDir, time.Second, testLogger())
	if err := s2.Load(); err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if s2.TaskCount() != 1 {
		t.Errorf("expected 1 task, got %d", s2.TaskCount())
	}
}

func TestStorage_GracefulShutdown(t *testing.T) {
	tmpDir := t.TempDir()
	s := New(tmpDir, time.Hour, testLogger()) // Long interval, won't auto-flush

	ctx := context.Background()
	s.Start(ctx)

	// Add data
	s.UpdateTaskStats("task1", 1, 1.0, 1.0, 0, 0)

	// Stop should save
	if err := s.Stop(); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}

	// Load and verify
	s2 := New(tmpDir, time.Second, testLogger())
	if err := s2.Load(); err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if s2.TaskCount() != 1 {
		t.Errorf("expected 1 task after graceful shutdown, got %d", s2.TaskCount())
	}
}

func TestStorage_AtomicWrite(t *testing.T) {
	tmpDir := t.TempDir()
	s := New(tmpDir, time.Second, testLogger())

	s.UpdateTaskStats("task1", 1, 1.0, 1.0, 0, 0)

	if err := s.Save(); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Check that temp file doesn't exist
	tempPath := filepath.Join(tmpDir, dataFileName+".tmp")
	if _, err := os.Stat(tempPath); !os.IsNotExist(err) {
		t.Error("temp file should not exist after save")
	}

	// Check that main file exists
	filePath := filepath.Join(tmpDir, dataFileName)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Error("data file should exist after save")
	}
}

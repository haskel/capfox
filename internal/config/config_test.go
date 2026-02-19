package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefault(t *testing.T) {
	cfg := Default()

	if cfg.Server.Port != 8080 {
		t.Errorf("expected default port 8080, got %d", cfg.Server.Port)
	}

	if cfg.Server.Host != "0.0.0.0" {
		t.Errorf("expected default host 0.0.0.0, got %s", cfg.Server.Host)
	}

	if cfg.Thresholds.CPU.MaxPercent != 80.0 {
		t.Errorf("expected default CPU threshold 80.0, got %f", cfg.Thresholds.CPU.MaxPercent)
	}

	if cfg.Logging.Level != "info" {
		t.Errorf("expected default log level info, got %s", cfg.Logging.Level)
	}
}

func TestLoad(t *testing.T) {
	content := `
server:
  host: "127.0.0.1"
  port: 9090

thresholds:
  cpu:
    max_percent: 70

logging:
  level: "debug"
  format: "text"
`
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if cfg.Server.Host != "127.0.0.1" {
		t.Errorf("expected host 127.0.0.1, got %s", cfg.Server.Host)
	}

	if cfg.Server.Port != 9090 {
		t.Errorf("expected port 9090, got %d", cfg.Server.Port)
	}

	if cfg.Thresholds.CPU.MaxPercent != 70.0 {
		t.Errorf("expected CPU threshold 70.0, got %f", cfg.Thresholds.CPU.MaxPercent)
	}

	if cfg.Logging.Level != "debug" {
		t.Errorf("expected log level debug, got %s", cfg.Logging.Level)
	}

	// Check that defaults are preserved for unspecified values
	if cfg.Thresholds.Memory.MaxPercent != 85.0 {
		t.Errorf("expected default memory threshold 85.0, got %f", cfg.Thresholds.Memory.MaxPercent)
	}
}

func TestLoadFileNotFound(t *testing.T) {
	_, err := Load("/nonexistent/path/config.yaml")
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}

func TestLoadOrDefault(t *testing.T) {
	// Empty path returns defaults
	cfg := LoadOrDefault("")
	if cfg.Server.Port != 8080 {
		t.Errorf("expected default port 8080, got %d", cfg.Server.Port)
	}

	// Non-existent file returns defaults
	cfg = LoadOrDefault("/nonexistent/path/config.yaml")
	if cfg.Server.Port != 8080 {
		t.Errorf("expected default port 8080, got %d", cfg.Server.Port)
	}
}

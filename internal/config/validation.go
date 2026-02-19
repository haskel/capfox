package config

import (
	"errors"
	"fmt"
)

func (c *Config) Validate() error {
	var errs []error

	if err := c.Server.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("server: %w", err))
	}

	if err := c.Thresholds.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("thresholds: %w", err))
	}

	if err := c.Monitoring.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("monitoring: %w", err))
	}

	if err := c.Persistence.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("persistence: %w", err))
	}

	if err := c.Logging.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("logging: %w", err))
	}

	if err := c.Learning.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("learning: %w", err))
	}

	if err := c.Auth.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("auth: %w", err))
	}

	return errors.Join(errs...)
}

func (s *ServerConfig) Validate() error {
	if s.Port < 1 || s.Port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535, got %d", s.Port)
	}
	return nil
}

func (t *ThresholdsConfig) Validate() error {
	var errs []error

	if t.CPU.MaxPercent < 0 || t.CPU.MaxPercent > 100 {
		errs = append(errs, fmt.Errorf("cpu.max_percent must be between 0 and 100"))
	}

	if t.Memory.MaxPercent < 0 || t.Memory.MaxPercent > 100 {
		errs = append(errs, fmt.Errorf("memory.max_percent must be between 0 and 100"))
	}

	if t.GPU.MaxPercent < 0 || t.GPU.MaxPercent > 100 {
		errs = append(errs, fmt.Errorf("gpu.max_percent must be between 0 and 100"))
	}

	if t.VRAM.MaxPercent < 0 || t.VRAM.MaxPercent > 100 {
		errs = append(errs, fmt.Errorf("vram.max_percent must be between 0 and 100"))
	}

	if t.Storage.MinFreeGB < 0 {
		errs = append(errs, fmt.Errorf("storage.min_free_gb must be non-negative"))
	}

	return errors.Join(errs...)
}

func (m *MonitoringConfig) Validate() error {
	if m.IntervalMS < 100 {
		return fmt.Errorf("interval_ms must be at least 100, got %d", m.IntervalMS)
	}
	return nil
}

func (p *PersistenceConfig) Validate() error {
	if p.DataDir == "" {
		return fmt.Errorf("data_dir cannot be empty")
	}
	if p.FlushIntervalSec < 1 {
		return fmt.Errorf("flush_interval_sec must be at least 1")
	}
	return nil
}

func (l *LoggingConfig) Validate() error {
	validLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}
	if !validLevels[l.Level] {
		return fmt.Errorf("invalid log level: %s (valid: debug, info, warn, error)", l.Level)
	}

	validFormats := map[string]bool{
		"json": true,
		"text": true,
	}
	if !validFormats[l.Format] {
		return fmt.Errorf("invalid log format: %s (valid: json, text)", l.Format)
	}

	return nil
}

func (l *LearningConfig) Validate() error {
	validModels := map[string]bool{
		"moving_average":    true,
		"linear_regression": true,
	}
	if !validModels[l.Model] {
		return fmt.Errorf("invalid learning model: %s (valid: moving_average, linear_regression)", l.Model)
	}
	if l.ObservationDelaySec < 1 {
		return fmt.Errorf("observation_delay_sec must be at least 1")
	}
	return nil
}

func (a *AuthConfig) Validate() error {
	if a.Enabled {
		if a.User == "" {
			return fmt.Errorf("user cannot be empty when auth is enabled")
		}
		if a.Password == "" {
			return fmt.Errorf("password cannot be empty when auth is enabled")
		}
	}
	return nil
}

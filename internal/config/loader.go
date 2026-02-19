package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

func Load(path string) (*Config, error) {
	cfg := Default()

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	data = substituteEnvVars(data)

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return cfg, nil
}

// LoadOrDefault loads config from path, falling back to defaults on error.
// Logs a warning to stderr if config loading fails.
func LoadOrDefault(path string) *Config {
	if path == "" {
		return Default()
	}

	cfg, err := Load(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to load config from %s: %v (using defaults)\n", path, err)
		return Default()
	}

	return cfg
}

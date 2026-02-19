package config

import "time"

type Config struct {
	Server      ServerConfig      `yaml:"server"`
	Auth        AuthConfig        `yaml:"auth"`
	Thresholds  ThresholdsConfig  `yaml:"thresholds"`
	Monitoring  MonitoringConfig  `yaml:"monitoring"`
	Persistence PersistenceConfig `yaml:"persistence"`
	Logging     LoggingConfig     `yaml:"logging"`
	Learning    LearningConfig    `yaml:"learning"`
}

type ServerConfig struct {
	Host    string `yaml:"host"`
	Port    int    `yaml:"port"`
	PIDFile string `yaml:"pid_file"`
}

type AuthConfig struct {
	Enabled  bool   `yaml:"enabled"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
}

type ThresholdsConfig struct {
	CPU     CPUThreshold     `yaml:"cpu"`
	Memory  MemoryThreshold  `yaml:"memory"`
	GPU     GPUThreshold     `yaml:"gpu"`
	VRAM    VRAMThreshold    `yaml:"vram"`
	Storage StorageThreshold `yaml:"storage"`
}

type CPUThreshold struct {
	MaxPercent float64 `yaml:"max_percent"`
}

type MemoryThreshold struct {
	MaxPercent float64 `yaml:"max_percent"`
}

type GPUThreshold struct {
	MaxPercent float64 `yaml:"max_percent"`
}

type VRAMThreshold struct {
	MaxPercent float64 `yaml:"max_percent"`
}

type StorageThreshold struct {
	MinFreeGB float64 `yaml:"min_free_gb"`
}

type MonitoringConfig struct {
	IntervalMS int      `yaml:"interval_ms"`
	Paths      []string `yaml:"paths"`
}

type PersistenceConfig struct {
	DataDir          string `yaml:"data_dir"`
	FlushIntervalSec int    `yaml:"flush_interval_sec"`
}

type LoggingConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
}

type LearningConfig struct {
	Model               string `yaml:"model"`
	ObservationDelaySec int    `yaml:"observation_delay_sec"`
}

func (c *Config) MonitoringInterval() time.Duration {
	return time.Duration(c.Monitoring.IntervalMS) * time.Millisecond
}

func (c *Config) FlushInterval() time.Duration {
	return time.Duration(c.Persistence.FlushIntervalSec) * time.Second
}

func (c *Config) ObservationDelay() time.Duration {
	return time.Duration(c.Learning.ObservationDelaySec) * time.Second
}

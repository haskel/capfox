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
	Decision    DecisionConfig    `yaml:"decision"`
	Debug       DebugConfig       `yaml:"debug"`
}

// DebugConfig holds debug mode configuration.
type DebugConfig struct {
	// Enabled allows debug endpoints like /debug/inject-metrics
	Enabled bool `yaml:"enabled"`
	// Auth holds debug-specific authentication.
	// If set, debug endpoints require this token.
	// If not set but main auth is enabled, main auth is used.
	Auth DebugAuthConfig `yaml:"auth"`
}

// DebugAuthConfig holds debug endpoint authentication.
type DebugAuthConfig struct {
	// Token for Bearer authentication on debug endpoints.
	// If empty, falls back to main auth.
	Token string `yaml:"token"`
}

type ServerConfig struct {
	Host      string          `yaml:"host"`
	Port      int             `yaml:"port"`
	PIDFile   string          `yaml:"pid_file"`
	Profiling ProfilingConfig `yaml:"profiling"`
}

type ProfilingConfig struct {
	Enabled bool `yaml:"enabled"`
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

// DecisionConfig holds decision engine configuration.
type DecisionConfig struct {
	// Strategy type: threshold, predictive, conservative, queue_aware
	Strategy string `yaml:"strategy"`

	// Model type: none, moving_average, linear, polynomial, gradient_boosting
	Model string `yaml:"model"`

	// Fallback strategy when insufficient data
	FallbackStrategy string `yaml:"fallback_strategy"`

	// Minimum observations before using predictions
	MinObservations int `yaml:"min_observations"`

	// Safety buffer for conservative strategy (percentage, e.g., 10 for 10%)
	SafetyBufferPercent float64 `yaml:"safety_buffer_percent"`

	// Model-specific parameters
	ModelParams ModelParamsConfig `yaml:"model_params"`
}

// ModelParamsConfig holds model-specific parameters.
type ModelParamsConfig struct {
	// MovingAverage: smoothing factor (0.1-0.3)
	Alpha float64 `yaml:"alpha"`

	// Polynomial: degree (2-3)
	Degree int `yaml:"degree"`

	// GradientBoosting
	NEstimators       int    `yaml:"n_estimators"`
	MaxDepth          int    `yaml:"max_depth"`
	RetrainInterval   string `yaml:"retrain_interval"`
	MinRetrainSamples int    `yaml:"min_retrain_samples"`
	MaxBufferSize     int    `yaml:"max_buffer_size"`
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

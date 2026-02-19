package config

func Default() *Config {
	return &Config{
		Server: ServerConfig{
			Host:    "0.0.0.0",
			Port:    8080,
			PIDFile: "/var/run/capfox.pid",
			RateLimit: RateLimitConfig{
				Enabled:           false,
				RequestsPerSecond: 100,
				Burst:             200,
			},
		},
		Auth: AuthConfig{
			Enabled:  false,
			User:     "",
			Password: "",
		},
		Thresholds: ThresholdsConfig{
			CPU: CPUThreshold{
				MaxPercent: 80.0,
			},
			Memory: MemoryThreshold{
				MaxPercent: 85.0,
			},
			GPU: GPUThreshold{
				MaxPercent: 90.0,
			},
			VRAM: VRAMThreshold{
				MaxPercent: 85.0,
			},
			Storage: StorageThreshold{
				MinFreeGB: 10.0,
			},
		},
		Monitoring: MonitoringConfig{
			IntervalMS: 1000,
			Paths:      []string{"/"},
		},
		Persistence: PersistenceConfig{
			DataDir:          "/var/lib/capfox",
			FlushIntervalSec: 600,
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "json",
		},
		Learning: LearningConfig{
			Model:               "moving_average",
			ObservationDelaySec: 5,
		},
		Decision: DecisionConfig{
			Strategy:            "predictive",
			Model:               "linear",
			FallbackStrategy:    "threshold",
			MinObservations:     5,
			SafetyBufferPercent: 10.0,
			ModelParams: ModelParamsConfig{
				Alpha:             0.2,
				Degree:            2,
				NEstimators:       100,
				MaxDepth:          5,
				RetrainInterval:   "1h",
				MinRetrainSamples: 50,
				MaxBufferSize:     10000,
			},
		},
	}
}

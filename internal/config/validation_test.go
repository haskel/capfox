package config

import (
	"testing"
)

func TestValidateDefault(t *testing.T) {
	cfg := Default()
	if err := cfg.Validate(); err != nil {
		t.Errorf("default config should be valid: %v", err)
	}
}

func TestValidateServerPort(t *testing.T) {
	tests := []struct {
		port    int
		wantErr bool
	}{
		{0, true},
		{-1, true},
		{65536, true},
		{1, false},
		{8080, false},
		{65535, false},
	}

	for _, tt := range tests {
		cfg := Default()
		cfg.Server.Port = tt.port
		err := cfg.Server.Validate()
		if (err != nil) != tt.wantErr {
			t.Errorf("port %d: wantErr=%v, got %v", tt.port, tt.wantErr, err)
		}
	}
}

func TestValidateThresholds(t *testing.T) {
	tests := []struct {
		name    string
		modify  func(*ThresholdsConfig)
		wantErr bool
	}{
		{
			name:    "valid defaults",
			modify:  func(t *ThresholdsConfig) {},
			wantErr: false,
		},
		{
			name: "cpu over 100",
			modify: func(t *ThresholdsConfig) {
				t.CPU.MaxPercent = 101
			},
			wantErr: true,
		},
		{
			name: "cpu negative",
			modify: func(t *ThresholdsConfig) {
				t.CPU.MaxPercent = -1
			},
			wantErr: true,
		},
		{
			name: "memory over 100",
			modify: func(t *ThresholdsConfig) {
				t.Memory.MaxPercent = 150
			},
			wantErr: true,
		},
		{
			name: "storage negative",
			modify: func(t *ThresholdsConfig) {
				t.Storage.MinFreeGB = -5
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Default()
			tt.modify(&cfg.Thresholds)
			err := cfg.Thresholds.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("wantErr=%v, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestValidateLogging(t *testing.T) {
	tests := []struct {
		level   string
		format  string
		wantErr bool
	}{
		{"debug", "json", false},
		{"info", "json", false},
		{"warn", "json", false},
		{"error", "json", false},
		{"info", "text", false},
		{"invalid", "json", true},
		{"info", "invalid", true},
	}

	for _, tt := range tests {
		cfg := Default()
		cfg.Logging.Level = tt.level
		cfg.Logging.Format = tt.format
		err := cfg.Logging.Validate()
		if (err != nil) != tt.wantErr {
			t.Errorf("level=%s format=%s: wantErr=%v, got %v", tt.level, tt.format, tt.wantErr, err)
		}
	}
}

func TestValidateAuth(t *testing.T) {
	tests := []struct {
		name     string
		enabled  bool
		user     string
		password string
		wantErr  bool
	}{
		{"disabled no creds", false, "", "", false},
		{"enabled with creds", true, "admin", "secret", false},
		{"enabled no user", true, "", "secret", true},
		{"enabled no password", true, "admin", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Default()
			cfg.Auth.Enabled = tt.enabled
			cfg.Auth.User = tt.user
			cfg.Auth.Password = tt.password
			err := cfg.Auth.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("wantErr=%v, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestValidateLearning(t *testing.T) {
	tests := []struct {
		model   string
		delay   int
		wantErr bool
	}{
		{"moving_average", 5, false},
		{"linear_regression", 5, false},
		{"invalid_model", 5, true},
		{"moving_average", 0, true},
	}

	for _, tt := range tests {
		cfg := Default()
		cfg.Learning.Model = tt.model
		cfg.Learning.ObservationDelaySec = tt.delay
		err := cfg.Learning.Validate()
		if (err != nil) != tt.wantErr {
			t.Errorf("model=%s delay=%d: wantErr=%v, got %v", tt.model, tt.delay, tt.wantErr, err)
		}
	}
}

func TestValidateMonitoring(t *testing.T) {
	tests := []struct {
		interval int
		wantErr  bool
	}{
		{1000, false},
		{100, false},
		{99, true},
		{0, true},
	}

	for _, tt := range tests {
		cfg := Default()
		cfg.Monitoring.IntervalMS = tt.interval
		err := cfg.Monitoring.Validate()
		if (err != nil) != tt.wantErr {
			t.Errorf("interval=%d: wantErr=%v, got %v", tt.interval, tt.wantErr, err)
		}
	}
}

func TestValidateDebugSecurity(t *testing.T) {
	tests := []struct {
		name            string
		debugEnabled    bool
		profilingEnabled bool
		authEnabled     bool
		debugToken      string
		wantErr         bool
	}{
		{
			name:            "no debug no profiling",
			debugEnabled:    false,
			profilingEnabled: false,
			authEnabled:     false,
			debugToken:      "",
			wantErr:         false,
		},
		{
			name:            "debug enabled with main auth",
			debugEnabled:    true,
			profilingEnabled: false,
			authEnabled:     true,
			debugToken:      "",
			wantErr:         false,
		},
		{
			name:            "debug enabled with debug token",
			debugEnabled:    true,
			profilingEnabled: false,
			authEnabled:     false,
			debugToken:      "secret-token",
			wantErr:         false,
		},
		{
			name:            "debug enabled no auth",
			debugEnabled:    true,
			profilingEnabled: false,
			authEnabled:     false,
			debugToken:      "",
			wantErr:         true,
		},
		{
			name:            "profiling enabled with main auth",
			debugEnabled:    false,
			profilingEnabled: true,
			authEnabled:     true,
			debugToken:      "",
			wantErr:         false,
		},
		{
			name:            "profiling enabled no auth",
			debugEnabled:    false,
			profilingEnabled: true,
			authEnabled:     false,
			debugToken:      "",
			wantErr:         true,
		},
		{
			name:            "both enabled with token",
			debugEnabled:    true,
			profilingEnabled: true,
			authEnabled:     false,
			debugToken:      "token",
			wantErr:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Default()
			cfg.Debug.Enabled = tt.debugEnabled
			cfg.Server.Profiling.Enabled = tt.profilingEnabled
			cfg.Auth.Enabled = tt.authEnabled
			if tt.authEnabled {
				cfg.Auth.User = "admin"
				cfg.Auth.Password = "secret"
			}
			cfg.Debug.Auth.Token = tt.debugToken

			err := cfg.validateDebugSecurity()
			if (err != nil) != tt.wantErr {
				t.Errorf("wantErr=%v, got %v", tt.wantErr, err)
			}
		})
	}
}

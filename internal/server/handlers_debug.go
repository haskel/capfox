package server

import (
	"encoding/json"
	"net/http"

	"github.com/haskel/capfox/internal/monitor"
)

// InjectMetricsRequest is the request body for POST /debug/inject-metrics.
type InjectMetricsRequest struct {
	CPU       *float64 `json:"cpu,omitempty"`        // CPU usage percent
	Memory    *float64 `json:"memory,omitempty"`     // Memory usage percent
	GPUUsage  *float64 `json:"gpu_usage,omitempty"`  // GPU usage percent
	VRAMUsage *float64 `json:"vram_usage,omitempty"` // VRAM usage percent
	GPUIndex  int      `json:"gpu_index,omitempty"`  // Which GPU (default 0)
}

// handleInjectMetrics handles POST /debug/inject-metrics.
// Only available when debug mode is enabled.
func (s *Server) handleInjectMetrics(w http.ResponseWriter, r *http.Request) {
	var req InjectMetricsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Convert to monitor.InjectedMetrics
	metrics := &monitor.InjectedMetrics{
		CPU:       req.CPU,
		Memory:    req.Memory,
		GPUUsage:  req.GPUUsage,
		VRAMUsage: req.VRAMUsage,
		GPUIndex:  req.GPUIndex,
	}

	s.aggregator.InjectMetrics(metrics)

	s.logger.Info("metrics injected via debug endpoint",
		"cpu", req.CPU,
		"memory", req.Memory,
		"gpu_usage", req.GPUUsage,
		"vram_usage", req.VRAMUsage,
	)

	s.writeJSON(w, http.StatusOK, map[string]bool{"success": true})
}

// handleDebugStatus handles GET /debug/status.
// Returns current debug mode status and injected state.
func (s *Server) handleDebugStatus(w http.ResponseWriter, r *http.Request) {
	state := s.aggregator.GetState()

	resp := map[string]any{
		"debug_enabled": s.config.Debug.Enabled,
		"current_state": map[string]any{
			"cpu":    state.CPU.UsagePercent,
			"memory": state.Memory.UsagePercent,
			"gpus":   state.GPUs,
		},
	}

	s.writeJSON(w, http.StatusOK, resp)
}

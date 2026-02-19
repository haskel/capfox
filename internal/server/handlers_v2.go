package server

import (
	"encoding/json"
	"net/http"

	"github.com/haskel/capfox/internal/decision"
)

// AskRequestV2 is the request body for POST /v2/ask.
type AskRequestV2 struct {
	Task       string                    `json:"task"`
	Complexity int                       `json:"complexity,omitempty"`
	Resources  *decision.ResourceEstimate `json:"resources,omitempty"`
}

// AskResponseV2 is the response for POST /v2/ask.
type AskResponseV2 struct {
	Allowed        bool                  `json:"allowed"`
	Reasons        []string              `json:"reasons,omitempty"`
	PredictedState *decision.FutureState `json:"predicted,omitempty"`
	Confidence     float64               `json:"confidence"`
	Strategy       string                `json:"strategy"`
	Model          string                `json:"model"`
}

// handleAskV2 handles POST /v2/ask using the new decision engine.
func (s *Server) handleAskV2(w http.ResponseWriter, r *http.Request) {
	if s.v2 == nil || s.v2.DecisionManager == nil {
		http.Error(w, "decision engine not enabled", http.StatusServiceUnavailable)
		return
	}

	var req AskRequestV2
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Make decision using new engine
	result := s.v2.DecisionManager.Decide(req.Task, req.Complexity, req.Resources)

	// Convert reasons to strings
	var reasons []string
	if !result.Allowed {
		reasons = make([]string, len(result.Reasons))
		for i, r := range result.Reasons {
			reasons[i] = string(r)
		}
	}

	resp := AskResponseV2{
		Allowed:        result.Allowed,
		Reasons:        reasons,
		PredictedState: result.PredictedState,
		Confidence:     result.Confidence,
		Strategy:       result.Strategy,
		Model:          result.Model,
	}

	if resp.Allowed {
		writeJSON(w, http.StatusOK, resp)
	} else {
		writeJSON(w, http.StatusServiceUnavailable, resp)
	}
}

// ModelStatsResponse is the response for GET /v2/model/stats.
type ModelStatsResponse struct {
	ModelName         string                       `json:"model_name"`
	LearningType      string                       `json:"learning_type"`
	TotalObservations int64                        `json:"total_observations"`
	Tasks             map[string]*TaskStatsV2      `json:"tasks"`
}

// TaskStatsV2 is the stats for a single task in V2.
type TaskStatsV2 struct {
	Task         string       `json:"task"`
	Count        int64        `json:"count"`
	AvgCPUDelta  float64      `json:"avg_cpu_delta"`
	AvgMemDelta  float64      `json:"avg_mem_delta"`
	AvgGPUDelta  float64      `json:"avg_gpu_delta,omitempty"`
	AvgVRAMDelta float64      `json:"avg_vram_delta,omitempty"`
	Coefficients *Coefficients `json:"coefficients,omitempty"`
}

// Coefficients holds regression coefficients.
type Coefficients struct {
	CPUA  float64 `json:"cpu_a,omitempty"`
	CPUB  float64 `json:"cpu_b,omitempty"`
	MemA  float64 `json:"mem_a,omitempty"`
	MemB  float64 `json:"mem_b,omitempty"`
	GPUA  float64 `json:"gpu_a,omitempty"`
	GPUB  float64 `json:"gpu_b,omitempty"`
	VRAMA float64 `json:"vram_a,omitempty"`
	VRAMB float64 `json:"vram_b,omitempty"`
}

// handleModelStats handles GET /v2/model/stats.
func (s *Server) handleModelStats(w http.ResponseWriter, r *http.Request) {
	if s.v2 == nil || s.v2.Model == nil {
		http.Error(w, "decision engine not enabled", http.StatusServiceUnavailable)
		return
	}

	stats := s.v2.Model.Stats()

	// Convert to response format
	tasks := make(map[string]*TaskStatsV2)
	for name, ts := range stats.Tasks {
		task := &TaskStatsV2{
			Task:         ts.Task,
			Count:        ts.Count,
			AvgCPUDelta:  ts.AvgCPUDelta,
			AvgMemDelta:  ts.AvgMemDelta,
			AvgGPUDelta:  ts.AvgGPUDelta,
			AvgVRAMDelta: ts.AvgVRAMDelta,
		}

		if ts.Coefficients != nil {
			task.Coefficients = &Coefficients{
				CPUA:  ts.Coefficients.CPUA,
				CPUB:  ts.Coefficients.CPUB,
				MemA:  ts.Coefficients.MemA,
				MemB:  ts.Coefficients.MemB,
				GPUA:  ts.Coefficients.GPUA,
				GPUB:  ts.Coefficients.GPUB,
				VRAMA: ts.Coefficients.VRAMA,
				VRAMB: ts.Coefficients.VRAMB,
			}
		}

		tasks[name] = task
	}

	resp := ModelStatsResponse{
		ModelName:         stats.ModelName,
		LearningType:      stats.LearningType,
		TotalObservations: stats.TotalObservations,
		Tasks:             tasks,
	}

	writeJSON(w, http.StatusOK, resp)
}

// SchedulerStatsResponse is the response for GET /v2/scheduler/stats.
type SchedulerStatsResponse struct {
	Running      bool   `json:"running"`
	Interval     string `json:"interval"`
	RetrainCount int64  `json:"retrain_count"`
	LastRetrain  string `json:"last_retrain,omitempty"`
	LastError    string `json:"last_error,omitempty"`
}

// handleSchedulerStats handles GET /v2/scheduler/stats.
func (s *Server) handleSchedulerStats(w http.ResponseWriter, r *http.Request) {
	if s.v2 == nil || s.v2.Scheduler == nil {
		http.Error(w, "scheduler not enabled", http.StatusServiceUnavailable)
		return
	}

	stats := s.v2.Scheduler.Stats()

	resp := SchedulerStatsResponse{
		Running:      stats.Running,
		Interval:     stats.Interval,
		RetrainCount: stats.RetrainCount,
		LastError:    stats.LastError,
	}

	if !stats.LastRetrain.IsZero() {
		resp.LastRetrain = stats.LastRetrain.Format("2006-01-02T15:04:05Z")
	}

	writeJSON(w, http.StatusOK, resp)
}

// handleSchedulerRetrain handles POST /v2/scheduler/retrain.
func (s *Server) handleSchedulerRetrain(w http.ResponseWriter, r *http.Request) {
	if s.v2 == nil || s.v2.Scheduler == nil {
		http.Error(w, "scheduler not enabled", http.StatusServiceUnavailable)
		return
	}

	if err := s.v2.Scheduler.ForceRetrain(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"success": true})
}

package server

import (
	"encoding/json"
	"net/http"

	"github.com/haskel/capfox/internal/capacity"
	"github.com/haskel/capfox/internal/learning"
)

type InfoResponse struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type HealthResponse struct {
	Status string `json:"status"`
}

func (s *Server) handleInfo(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	resp := InfoResponse{
		Name:    "capfox",
		Version: s.version,
	}

	s.writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	resp := HealthResponse{
		Status: "ok",
	}

	s.writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	state := s.aggregator.GetState()
	s.writeJSON(w, http.StatusOK, state)
}

func (s *Server) handleAsk(w http.ResponseWriter, r *http.Request) {
	var req capacity.AskRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Check for reason flag in query param or header
	withReasons := r.URL.Query().Get("reason") == "true" || r.Header.Get("X-Reason") == "true"

	resp := s.capacityManager.Ask(req, withReasons)

	if resp.Allowed {
		s.writeJSON(w, http.StatusOK, resp)
	} else {
		s.writeJSON(w, http.StatusServiceUnavailable, resp)
	}
}

func (s *Server) handleTaskStart(w http.ResponseWriter, r *http.Request) {
	var req learning.TaskStartRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Task == "" {
		http.Error(w, "task field is required", http.StatusBadRequest)
		return
	}

	// Notify learning engine about task start
	if s.learningEngine != nil {
		s.learningEngine.NotifyTaskStart(req.Task, req.Complexity)
	}

	resp := learning.TaskStartResponse{
		Received: true,
		Task:     req.Task,
	}

	s.writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	// Check for specific task query
	taskName := r.URL.Query().Get("task")

	if s.learningEngine == nil {
		s.writeJSON(w, http.StatusOK, &learning.AllStats{
			Tasks:      make(map[string]*learning.TaskStats),
			TotalTasks: 0,
		})
		return
	}

	if taskName != "" {
		stats := s.learningEngine.GetTaskStats(taskName)
		if stats == nil {
			http.Error(w, "task not found", http.StatusNotFound)
			return
		}
		s.writeJSON(w, http.StatusOK, stats)
		return
	}

	stats := s.learningEngine.GetStats()
	s.writeJSON(w, http.StatusOK, stats)
}

func (s *Server) writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		s.logger.Error("failed to encode JSON response",
			"error", err,
			"status", status,
		)
	}
}

package server

import (
	"github.com/haskel/capfox/internal/decision"
	"github.com/haskel/capfox/internal/decision/model"
	"github.com/haskel/capfox/internal/decision/scheduler"
)

// V2Components holds the new decision engine components.
type V2Components struct {
	DecisionManager *decision.Manager
	Scheduler       *scheduler.Scheduler
	Model           model.PredictionModel
}

// SetDecisionComponents sets the new decision engine components.
func (s *Server) SetDecisionComponents(v2 *V2Components) {
	s.v2 = v2
}

// DecisionManager returns the decision manager if available.
func (s *Server) DecisionManager() *decision.Manager {
	if s.v2 != nil {
		return s.v2.DecisionManager
	}
	return nil
}

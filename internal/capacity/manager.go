package capacity

import (
	"sync"

	"github.com/haskel/capfox/internal/config"
	"github.com/haskel/capfox/internal/monitor"
)

type Manager struct {
	aggregator *monitor.Aggregator
	checker    *ThresholdChecker
	mu         sync.RWMutex
}

type AskRequest struct {
	Task       string            `json:"task"`
	Complexity int               `json:"complexity,omitempty"`
	Resources  *ResourceEstimate `json:"resources,omitempty"`
}

type ResourceEstimate struct {
	CPU    int `json:"cpu,omitempty"`
	GPU    int `json:"gpu,omitempty"`
	Memory int `json:"memory,omitempty"`
}

type AskResponse struct {
	Allowed bool     `json:"allowed"`
	Reasons []string `json:"reasons,omitempty"`
}

func NewManager(aggregator *monitor.Aggregator, thresholds config.ThresholdsConfig) *Manager {
	return &Manager{
		aggregator: aggregator,
		checker:    NewThresholdChecker(thresholds),
	}
}

func (m *Manager) Ask(req AskRequest, withReasons bool) AskResponse {
	m.mu.RLock()
	defer m.mu.RUnlock()

	state := m.aggregator.GetState()
	reasons := m.checker.Check(state)

	allowed := len(reasons) == 0

	resp := AskResponse{
		Allowed: allowed,
	}

	if withReasons && !allowed {
		resp.Reasons = make([]string, len(reasons))
		for i, r := range reasons {
			resp.Reasons[i] = string(r)
		}
	}

	return resp
}

func (m *Manager) UpdateThresholds(thresholds config.ThresholdsConfig) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.checker.UpdateThresholds(thresholds)
}

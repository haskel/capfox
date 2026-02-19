package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		fetchStatus(m.config),
		fetchStats(m.config),
		tick(m.config.RefreshInterval),
	)
}

// Update handles messages
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyPress(msg)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case statusMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.err = nil
			m.status = msg.data
			m.lastUpdated = time.Now()
		}
		return m, nil

	case statsMsg:
		if msg.err != nil {
			// Don't override status error
			if m.err == nil {
				m.err = msg.err
			}
		} else {
			m.stats = msg.data
		}
		return m, nil

	case tickMsg:
		m.loading = true
		return m, tea.Batch(
			fetchStatus(m.config),
			fetchStats(m.config),
			tick(m.config.RefreshInterval),
		)
	}

	return m, nil
}

func (m Model) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit

	case "r":
		// Manual refresh
		m.loading = true
		return m, tea.Batch(
			fetchStatus(m.config),
			fetchStats(m.config),
		)

	case "up", "k":
		if m.tableOffset > 0 {
			m.tableOffset--
		}
		return m, nil

	case "down", "j":
		if m.stats != nil && m.tableOffset < len(m.stats.Tasks)-1 {
			m.tableOffset++
		}
		return m, nil
	}

	return m, nil
}

package tui

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// Messages for tea.Cmd
type statusMsg struct {
	data *StatusData
	err  error
}

type statsMsg struct {
	data *StatsData
	err  error
}

type tickMsg time.Time

// API client for TUI
type apiClient struct {
	baseURL  string
	client   *http.Client
	user     string
	password string
}

func newAPIClient(cfg Config) *apiClient {
	return &apiClient{
		baseURL: cfg.ServerURL,
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
		user:     cfg.User,
		password: cfg.Password,
	}
}

func (c *apiClient) get(path string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return nil, err
	}

	if c.user != "" && c.password != "" {
		req.SetBasicAuth(c.user, c.password)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

// fetchStatus fetches status from API as tea.Cmd
func fetchStatus(cfg Config) tea.Cmd {
	return func() tea.Msg {
		client := newAPIClient(cfg)
		data, err := client.get("/status")
		if err != nil {
			return statusMsg{err: err}
		}

		var status StatusData
		if err := json.Unmarshal(data, &status); err != nil {
			return statusMsg{err: fmt.Errorf("failed to parse status: %w", err)}
		}

		return statusMsg{data: &status}
	}
}

// fetchStats fetches task statistics from API as tea.Cmd
func fetchStats(cfg Config) tea.Cmd {
	return func() tea.Msg {
		client := newAPIClient(cfg)
		data, err := client.get("/stats")
		if err != nil {
			return statsMsg{err: err}
		}

		var stats StatsData
		if err := json.Unmarshal(data, &stats); err != nil {
			return statsMsg{err: fmt.Errorf("failed to parse stats: %w", err)}
		}

		return statsMsg{data: &stats}
	}
}

// tick creates a periodic tick command
func tick(interval time.Duration) tea.Cmd {
	return tea.Tick(interval, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/haskel/capfox/internal/capacity"
	"github.com/haskel/capfox/internal/config"
	"github.com/haskel/capfox/internal/learning"
	"github.com/haskel/capfox/internal/monitor"
)

func TestServer_Integration(t *testing.T) {
	cfg := config.Default()
	cfg.Server.Port = 0 // Let OS assign port

	monitors := []monitor.Monitor{
		monitor.NewCPUMonitor(),
		monitor.NewMemoryMonitor(),
	}

	agg := monitor.NewAggregator(monitors, 100*time.Millisecond, testLogger())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	agg.Start(ctx)
	defer agg.Stop()

	cm := capacity.NewManager(agg, cfg.Thresholds)
	model := learning.NewMovingAverageModel(0.2)
	le := learning.NewEngine(model, agg, time.Second, testLogger())
	srv := New(cfg, agg, cm, le, testLogger(), "0.1.0")

	// Create test server
	ts := httptest.NewServer(srv.httpServer.Handler)
	defer ts.Close()

	// Test GET /
	t.Run("GET /", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/")
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
		}

		var info InfoResponse
		if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
			t.Fatalf("failed to decode: %v", err)
		}

		if info.Name != "capfox" {
			t.Errorf("expected name 'capfox', got %s", info.Name)
		}
	})

	// Test GET /health
	t.Run("GET /health", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/health")
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
		}

		var health HealthResponse
		if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
			t.Fatalf("failed to decode: %v", err)
		}

		if health.Status != "ok" {
			t.Errorf("expected status 'ok', got %s", health.Status)
		}
	})

	// Test GET /status
	t.Run("GET /status", func(t *testing.T) {
		// Wait for aggregator to collect data
		time.Sleep(150 * time.Millisecond)

		resp, err := http.Get(ts.URL + "/status")
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
		}

		var state monitor.SystemState
		if err := json.NewDecoder(resp.Body).Decode(&state); err != nil {
			t.Fatalf("failed to decode: %v", err)
		}

		// CPU should have valid data
		if state.CPU.UsagePercent < 0 || state.CPU.UsagePercent > 100 {
			t.Errorf("invalid CPU usage: %f", state.CPU.UsagePercent)
		}

		// Memory should have data
		if state.Memory.TotalBytes == 0 {
			t.Error("memory total should not be zero")
		}
	})

	// Test 404
	t.Run("GET /unknown", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/unknown")
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("expected status 404, got %d", resp.StatusCode)
		}
	})
}

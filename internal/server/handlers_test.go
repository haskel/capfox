package server

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/haskel/capfox/internal/capacity"
	"github.com/haskel/capfox/internal/config"
	"github.com/haskel/capfox/internal/learning"
	"github.com/haskel/capfox/internal/monitor"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
}

func testServer(t *testing.T) *Server {
	cfg := config.Default()

	monitors := []monitor.Monitor{
		&mockMonitor{
			name: "cpu",
			data: &monitor.CPUState{UsagePercent: 50.0, Cores: []float64{50.0}},
		},
		&mockMonitor{
			name: "memory",
			data: &monitor.MemoryState{UsedBytes: 1024, TotalBytes: 2048, UsagePercent: 50.0},
		},
		&mockMonitor{
			name: "storage",
			data: monitor.StorageState{
				"/": {UsedBytes: 100 * 1024 * 1024 * 1024, TotalBytes: 500 * 1024 * 1024 * 1024},
			},
		},
	}

	agg := monitor.NewAggregator(monitors, time.Second, testLogger())
	ctx := context.Background()
	agg.Start(ctx)

	cm := capacity.NewManager(agg, cfg.Thresholds)

	model := learning.NewMovingAverageModel(0.2)
	le := learning.NewEngine(model, agg, time.Second, testLogger())

	return New(cfg, agg, cm, le, testLogger(), "0.1.0-test")
}

func testServerOverloaded(t *testing.T) *Server {
	cfg := config.Default()

	monitors := []monitor.Monitor{
		&mockMonitor{
			name: "cpu",
			data: &monitor.CPUState{UsagePercent: 90.0, Cores: []float64{90.0}}, // Overloaded
		},
		&mockMonitor{
			name: "memory",
			data: &monitor.MemoryState{UsedBytes: 1024, TotalBytes: 2048, UsagePercent: 50.0},
		},
	}

	agg := monitor.NewAggregator(monitors, time.Second, testLogger())
	ctx := context.Background()
	agg.Start(ctx)

	cm := capacity.NewManager(agg, cfg.Thresholds)

	model := learning.NewMovingAverageModel(0.2)
	le := learning.NewEngine(model, agg, time.Second, testLogger())

	return New(cfg, agg, cm, le, testLogger(), "0.1.0-test")
}

type mockMonitor struct {
	name string
	data any
}

func (m *mockMonitor) Name() string {
	return m.name
}

func (m *mockMonitor) Collect() (any, error) {
	return m.data, nil
}

func TestHandleInfo(t *testing.T) {
	srv := testServer(t)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	srv.handleInfo(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp InfoResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Name != "capfox" {
		t.Errorf("expected name 'capfox', got %s", resp.Name)
	}

	if resp.Version != "0.1.0-test" {
		t.Errorf("expected version '0.1.0-test', got %s", resp.Version)
	}
}

func TestHandleInfo_NotFound(t *testing.T) {
	srv := testServer(t)

	req := httptest.NewRequest(http.MethodGet, "/other", nil)
	w := httptest.NewRecorder()

	srv.handleInfo(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

func TestHandleHealth(t *testing.T) {
	srv := testServer(t)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	srv.handleHealth(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp HealthResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Status != "ok" {
		t.Errorf("expected status 'ok', got %s", resp.Status)
	}
}

func TestHandleStatus(t *testing.T) {
	srv := testServer(t)

	req := httptest.NewRequest(http.MethodGet, "/status", nil)
	w := httptest.NewRecorder()

	srv.handleStatus(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("expected Content-Type 'application/json', got %s", contentType)
	}

	var state monitor.SystemState
	if err := json.NewDecoder(w.Body).Decode(&state); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if state.CPU.UsagePercent != 50.0 {
		t.Errorf("expected CPU usage 50.0, got %f", state.CPU.UsagePercent)
	}

	if state.Memory.UsagePercent != 50.0 {
		t.Errorf("expected memory usage 50.0, got %f", state.Memory.UsagePercent)
	}
}

func TestHandleAsk_Allowed(t *testing.T) {
	srv := testServer(t)

	body := `{"task": "test_task", "complexity": 100}`
	req := httptest.NewRequest(http.MethodPost, "/ask", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.handleAsk(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp capacity.AskResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if !resp.Allowed {
		t.Error("expected allowed=true")
	}
}

func TestHandleAsk_Denied(t *testing.T) {
	srv := testServerOverloaded(t)

	body := `{"task": "test_task", "complexity": 100}`
	req := httptest.NewRequest(http.MethodPost, "/ask", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.handleAsk(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status 503, got %d", w.Code)
	}

	var resp capacity.AskResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Allowed {
		t.Error("expected allowed=false")
	}
}

func TestHandleAsk_WithReasonQuery(t *testing.T) {
	srv := testServerOverloaded(t)

	body := `{"task": "test_task"}`
	req := httptest.NewRequest(http.MethodPost, "/ask?reason=true", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.handleAsk(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status 503, got %d", w.Code)
	}

	var resp capacity.AskResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Allowed {
		t.Error("expected allowed=false")
	}

	if len(resp.Reasons) == 0 {
		t.Error("expected reasons with ?reason=true")
	}
}

func TestHandleAsk_WithReasonHeader(t *testing.T) {
	srv := testServerOverloaded(t)

	body := `{"task": "test_task"}`
	req := httptest.NewRequest(http.MethodPost, "/ask", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Reason", "true")
	w := httptest.NewRecorder()

	srv.handleAsk(w, req)

	var resp capacity.AskResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(resp.Reasons) == 0 {
		t.Error("expected reasons with X-Reason header")
	}
}

func TestHandleAsk_InvalidBody(t *testing.T) {
	srv := testServer(t)

	body := `invalid json`
	req := httptest.NewRequest(http.MethodPost, "/ask", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.handleAsk(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestHandleTaskStart(t *testing.T) {
	srv := testServer(t)

	body := `{"task": "test_task", "complexity": 100}`
	req := httptest.NewRequest(http.MethodPost, "/task/notify", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.handleTaskStart(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp learning.TaskStartResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if !resp.Received {
		t.Error("expected received=true")
	}

	if resp.Task != "test_task" {
		t.Errorf("expected task 'test_task', got %s", resp.Task)
	}
}

func TestHandleTaskStart_EmptyTask(t *testing.T) {
	srv := testServer(t)

	body := `{"complexity": 100}`
	req := httptest.NewRequest(http.MethodPost, "/task/notify", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.handleTaskStart(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestHandleTaskStart_InvalidBody(t *testing.T) {
	srv := testServer(t)

	body := `invalid json`
	req := httptest.NewRequest(http.MethodPost, "/task/notify", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.handleTaskStart(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestHandleStats(t *testing.T) {
	srv := testServer(t)

	req := httptest.NewRequest(http.MethodGet, "/stats", nil)
	w := httptest.NewRecorder()

	srv.handleStats(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp learning.AllStats
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Tasks == nil {
		t.Error("expected tasks map")
	}
}

func TestHandleStats_WithTask(t *testing.T) {
	srv := testServer(t)

	// First notify about a task
	body := `{"task": "test_task"}`
	req := httptest.NewRequest(http.MethodPost, "/task/notify", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.handleTaskStart(w, req)

	// Wait for observation
	time.Sleep(1100 * time.Millisecond)

	// Now get stats for specific task
	req = httptest.NewRequest(http.MethodGet, "/stats?task=test_task", nil)
	w = httptest.NewRecorder()
	srv.handleStats(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestHandleStats_TaskNotFound(t *testing.T) {
	srv := testServer(t)

	req := httptest.NewRequest(http.MethodGet, "/stats?task=unknown_task", nil)
	w := httptest.NewRecorder()

	srv.handleStats(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

func TestHandleReady_Ready(t *testing.T) {
	srv := testServer(t) // testServer calls agg.Start() which sets ready=true

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	w := httptest.NewRecorder()

	srv.handleReady(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp ReadyResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if !resp.Ready {
		t.Error("expected ready=true")
	}

	if resp.Message != "" {
		t.Errorf("expected empty message, got %s", resp.Message)
	}
}

func TestHandleReady_NotReady(t *testing.T) {
	cfg := config.Default()

	monitors := []monitor.Monitor{
		&mockMonitor{
			name: "cpu",
			data: &monitor.CPUState{UsagePercent: 50.0, Cores: []float64{50.0}},
		},
	}

	// Create aggregator but don't start it - it won't be ready
	agg := monitor.NewAggregator(monitors, time.Second, testLogger())
	// Note: NOT calling agg.Start() - aggregator is not ready

	cm := capacity.NewManager(agg, cfg.Thresholds)

	model := learning.NewMovingAverageModel(0.2)
	le := learning.NewEngine(model, agg, time.Second, testLogger())

	srv := New(cfg, agg, cm, le, testLogger(), "0.1.0-test")

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	w := httptest.NewRecorder()

	srv.handleReady(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status 503, got %d", w.Code)
	}

	var resp ReadyResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Ready {
		t.Error("expected ready=false")
	}

	if resp.Message == "" {
		t.Error("expected non-empty message for not ready state")
	}
}

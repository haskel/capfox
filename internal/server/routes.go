package server

import (
	"net/http"
	"net/http/pprof"

	"github.com/haskel/capfox/internal/server/middleware"
)

func (s *Server) setupRoutes() *http.ServeMux {
	mux := http.NewServeMux()

	// V1 routes (backward compatible)
	mux.HandleFunc("GET /", s.handleInfo)
	mux.HandleFunc("GET /health", s.handleHealth)
	mux.HandleFunc("GET /ready", s.handleReady)
	mux.HandleFunc("GET /status", s.handleStatus)
	mux.HandleFunc("POST /ask", s.handleAsk)
	mux.HandleFunc("POST /task/notify", s.handleTaskStart)
	mux.HandleFunc("GET /stats", s.handleStats)

	// V2 routes (new decision engine)
	mux.HandleFunc("POST /v2/ask", s.handleAskV2)
	mux.HandleFunc("GET /v2/model/stats", s.handleModelStats)
	mux.HandleFunc("GET /v2/scheduler/stats", s.handleSchedulerStats)
	mux.HandleFunc("POST /v2/scheduler/retrain", s.handleSchedulerRetrain)

	// Setup debug routes with separate authentication
	s.setupDebugRoutes(mux)

	return mux
}

// setupDebugRoutes configures debug and profiling endpoints with authentication.
func (s *Server) setupDebugRoutes(mux *http.ServeMux) {
	profilingEnabled := s.config.Server.Profiling.Enabled
	debugEnabled := s.config.Debug.Enabled

	if !profilingEnabled && !debugEnabled {
		return
	}

	// Create debug auth middleware config
	debugAuthConfig := &middleware.DebugAuthConfig{
		Token:              s.config.Debug.Auth.Token,
		FallbackAuthConfig: s.authConfig,
	}
	debugAuth := middleware.DebugAuth(debugAuthConfig)

	// Profiling routes (if enabled)
	if profilingEnabled {
		s.logger.Info("profiling endpoints enabled at /debug/pprof/ (auth required)")
		// Wrap pprof handlers with debug auth
		mux.Handle("GET /debug/pprof/{$}", debugAuth(http.HandlerFunc(pprof.Index)))
		mux.Handle("GET /debug/pprof/cmdline", debugAuth(http.HandlerFunc(pprof.Cmdline)))
		mux.Handle("GET /debug/pprof/profile", debugAuth(http.HandlerFunc(pprof.Profile)))
		mux.Handle("GET /debug/pprof/symbol", debugAuth(http.HandlerFunc(pprof.Symbol)))
		mux.Handle("POST /debug/pprof/symbol", debugAuth(http.HandlerFunc(pprof.Symbol)))
		mux.Handle("GET /debug/pprof/trace", debugAuth(http.HandlerFunc(pprof.Trace)))
		mux.Handle("GET /debug/pprof/heap", debugAuth(pprof.Handler("heap")))
		mux.Handle("GET /debug/pprof/goroutine", debugAuth(pprof.Handler("goroutine")))
		mux.Handle("GET /debug/pprof/allocs", debugAuth(pprof.Handler("allocs")))
		mux.Handle("GET /debug/pprof/block", debugAuth(pprof.Handler("block")))
		mux.Handle("GET /debug/pprof/mutex", debugAuth(pprof.Handler("mutex")))
		mux.Handle("GET /debug/pprof/threadcreate", debugAuth(pprof.Handler("threadcreate")))
		// Catch-all for index
		mux.Handle("GET /debug/pprof/{name...}", debugAuth(http.HandlerFunc(pprof.Index)))
	}

	// Debug routes (if enabled)
	if debugEnabled {
		s.logger.Warn("debug mode enabled - debug endpoints require authentication")
		mux.Handle("POST /debug/inject-metrics", debugAuth(http.HandlerFunc(s.handleInjectMetrics)))
		mux.Handle("GET /debug/status", debugAuth(http.HandlerFunc(s.handleDebugStatus)))
	}
}

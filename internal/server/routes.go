package server

import (
	"net/http"
	"net/http/pprof"
)

func (s *Server) setupRoutes() *http.ServeMux {
	mux := http.NewServeMux()

	// Profiling routes (if enabled)
	if s.config.Server.Profiling.Enabled {
		s.logger.Info("profiling endpoints enabled at /debug/pprof/")
		// Use {$} to match exact paths and {path...} for prefix matching
		mux.HandleFunc("GET /debug/pprof/{$}", pprof.Index)
		mux.HandleFunc("GET /debug/pprof/cmdline", pprof.Cmdline)
		mux.HandleFunc("GET /debug/pprof/profile", pprof.Profile)
		mux.HandleFunc("GET /debug/pprof/symbol", pprof.Symbol)
		mux.HandleFunc("POST /debug/pprof/symbol", pprof.Symbol)
		mux.HandleFunc("GET /debug/pprof/trace", pprof.Trace)
		mux.HandleFunc("GET /debug/pprof/heap", pprof.Handler("heap").ServeHTTP)
		mux.HandleFunc("GET /debug/pprof/goroutine", pprof.Handler("goroutine").ServeHTTP)
		mux.HandleFunc("GET /debug/pprof/allocs", pprof.Handler("allocs").ServeHTTP)
		mux.HandleFunc("GET /debug/pprof/block", pprof.Handler("block").ServeHTTP)
		mux.HandleFunc("GET /debug/pprof/mutex", pprof.Handler("mutex").ServeHTTP)
		mux.HandleFunc("GET /debug/pprof/threadcreate", pprof.Handler("threadcreate").ServeHTTP)
		// Catch-all for index to handle /debug/pprof/ with trailing paths
		mux.HandleFunc("GET /debug/pprof/{name...}", pprof.Index)
	}

	// V1 routes (backward compatible)
	mux.HandleFunc("GET /", s.handleInfo)
	mux.HandleFunc("GET /health", s.handleHealth)
	mux.HandleFunc("GET /status", s.handleStatus)
	mux.HandleFunc("POST /ask", s.handleAsk)
	mux.HandleFunc("POST /task/notify", s.handleTaskStart)
	mux.HandleFunc("GET /stats", s.handleStats)

	// V2 routes (new decision engine)
	mux.HandleFunc("POST /v2/ask", s.handleAskV2)
	mux.HandleFunc("GET /v2/model/stats", s.handleModelStats)
	mux.HandleFunc("GET /v2/scheduler/stats", s.handleSchedulerStats)
	mux.HandleFunc("POST /v2/scheduler/retrain", s.handleSchedulerRetrain)

	// Debug routes (only when debug mode is enabled)
	if s.config.Debug.Enabled {
		s.logger.Warn("debug mode enabled - debug endpoints are accessible")
		mux.HandleFunc("POST /debug/inject-metrics", s.handleInjectMetrics)
		mux.HandleFunc("GET /debug/status", s.handleDebugStatus)
	}

	return mux
}

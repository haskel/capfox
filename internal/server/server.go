package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/haskel/capfox/internal/capacity"
	"github.com/haskel/capfox/internal/config"
	"github.com/haskel/capfox/internal/learning"
	"github.com/haskel/capfox/internal/monitor"
	"github.com/haskel/capfox/internal/server/middleware"
)

type Server struct {
	httpServer      *http.Server
	aggregator      *monitor.Aggregator
	capacityManager *capacity.Manager
	learningEngine  *learning.Engine
	config          *config.Config
	logger          *slog.Logger
	version         string
	authConfig      *middleware.AuthConfig

	// V2 components (new decision engine)
	v2 *V2Components
}

func New(cfg *config.Config, agg *monitor.Aggregator, cm *capacity.Manager, le *learning.Engine, logger *slog.Logger, version string) *Server {
	authConfig := &middleware.AuthConfig{
		Enabled:  cfg.Auth.Enabled,
		User:     cfg.Auth.User,
		Password: cfg.Auth.Password,
	}

	s := &Server{
		aggregator:      agg,
		capacityManager: cm,
		learningEngine:  le,
		config:          cfg,
		logger:          logger,
		version:         version,
		authConfig:      authConfig,
	}

	mux := s.setupRoutes()

	// Build list of paths to exclude from main auth
	authExcludes := []string{"/health", "/ready"}

	// If debug endpoints have their own auth (via token), exclude from main auth
	// to avoid double authentication
	if cfg.Debug.Auth.Token != "" {
		if cfg.Debug.Enabled || cfg.Server.Profiling.Enabled {
			authExcludes = append(authExcludes, "/debug/*")
		}
	}

	// Rate limit config
	rateLimitConfig := &middleware.RateLimitConfig{
		Enabled:           cfg.Server.RateLimit.Enabled,
		RequestsPerSecond: cfg.Server.RateLimit.RequestsPerSecond,
		Burst:             cfg.Server.RateLimit.Burst,
	}

	handler := middleware.Chain(
		mux,
		middleware.Recovery(logger),
		middleware.SecurityHeaders(),
		middleware.RateLimit(rateLimitConfig),
		middleware.MaxBody(0), // 1MB limit
		middleware.Logging(logger),
		middleware.Auth(authConfig, authExcludes...),
	)

	s.httpServer = &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler:      handler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return s
}

// ReloadConfig reloads configuration that can be changed at runtime.
// Note: host/port changes require restart.
func (s *Server) ReloadConfig(cfg *config.Config) {
	s.logger.Info("reloading configuration")

	// Update auth config (thread-safe)
	s.authConfig.Update(cfg.Auth.Enabled, cfg.Auth.User, cfg.Auth.Password)

	// Update thresholds in capacity manager
	s.capacityManager.UpdateThresholds(cfg.Thresholds)

	// Update stored config
	s.config = cfg

	s.logger.Info("configuration reloaded",
		"auth_enabled", cfg.Auth.Enabled,
	)
}

func (s *Server) Start() error {
	s.logger.Info("server starting",
		"addr", s.httpServer.Addr,
	)
	return s.httpServer.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("server shutting down")
	return s.httpServer.Shutdown(ctx)
}

func (s *Server) Addr() string {
	return s.httpServer.Addr
}

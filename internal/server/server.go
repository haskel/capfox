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

	handler := middleware.Chain(
		mux,
		middleware.Recovery(logger),
		middleware.Logging(logger),
		middleware.Auth(authConfig, "/health"), // Exclude /health from auth
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

	// Update auth config (pointer is shared with middleware)
	s.authConfig.Enabled = cfg.Auth.Enabled
	s.authConfig.User = cfg.Auth.User
	s.authConfig.Password = cfg.Auth.Password

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

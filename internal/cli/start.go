package cli

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/haskel/capfox/internal/capacity"
	"github.com/haskel/capfox/internal/config"
	"github.com/haskel/capfox/internal/learning"
	"github.com/haskel/capfox/internal/logger"
	"github.com/haskel/capfox/internal/monitor"
	"github.com/haskel/capfox/internal/server"
	"github.com/haskel/capfox/internal/storage"
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the capfox server",
	Long:  `Start the capfox server in foreground mode.`,
	RunE:  runStart,
}

func init() {
	rootCmd.AddCommand(startCmd)
}

func runStart(cmd *cobra.Command, args []string) error {
	// Load config
	cfg := config.LoadOrDefault(cfgFile)

	// Override port if specified via flag
	if cmd.Flags().Changed("port") {
		cfg.Server.Port = port
	}
	if cmd.Flags().Changed("host") {
		cfg.Server.Host = host
	}

	// Create logger
	log := logger.New(cfg.Logging.Level, cfg.Logging.Format)

	log.Info("capfox starting",
		"version", Version,
		"config", cfgFile,
	)

	// Create monitors
	monitors := []monitor.Monitor{
		monitor.NewCPUMonitor(),
		monitor.NewMemoryMonitor(),
		monitor.NewStorageMonitor(cfg.Monitoring.Paths),
		monitor.NewProcessMonitor(),
		monitor.NewGPUMonitor(),
	}

	// Create aggregator
	agg := monitor.NewAggregator(monitors, cfg.MonitoringInterval(), log)

	// Start aggregator
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := agg.Start(ctx); err != nil {
		return fmt.Errorf("failed to start aggregator: %w", err)
	}

	// Create capacity manager
	cm := capacity.NewManager(agg, cfg.Thresholds)

	// Create storage
	store := storage.New(cfg.Persistence.DataDir, cfg.FlushInterval(), log)

	// Load persisted data
	if err := store.Load(); err != nil {
		log.Warn("failed to load persisted data", "error", err)
	}

	// Create learning engine
	model := learning.NewMovingAverageModel(0.2)

	// Load persisted stats into model
	savedStats := store.GetAllTaskStats()
	if len(savedStats) > 0 {
		allStats := &learning.AllStats{
			Tasks:      make(map[string]*learning.TaskStats),
			TotalTasks: 0,
		}
		for task, data := range savedStats {
			allStats.Tasks[task] = &learning.TaskStats{
				Task:         data.Task,
				Count:        data.Count,
				AvgCPUDelta:  data.AvgCPUDelta,
				AvgMemDelta:  data.AvgMemDelta,
				AvgGPUDelta:  data.AvgGPUDelta,
				AvgVRAMDelta: data.AvgVRAMDelta,
			}
			allStats.TotalTasks += data.Count
		}
		model.LoadStats(allStats)
		log.Info("loaded persisted stats", "tasks", len(savedStats))
	}

	// Set observer to persist stats on update
	model.SetObserver(func(task string, stats *learning.TaskStats) {
		store.UpdateTaskStats(task, stats.Count, stats.AvgCPUDelta, stats.AvgMemDelta, stats.AvgGPUDelta, stats.AvgVRAMDelta)
	})

	le := learning.NewEngine(model, agg, cfg.ObservationDelay(), log)

	// Start storage periodic flush
	store.Start(ctx)

	// Write PID file if configured
	if cfg.Server.PIDFile != "" {
		if err := writePIDFile(cfg.Server.PIDFile); err != nil {
			log.Warn("failed to write PID file", "error", err)
		} else {
			defer os.Remove(cfg.Server.PIDFile)
		}
	}

	// Create and start server
	srv := server.New(cfg, agg, cm, le, log, Version)

	// Signal channels
	sighupCh := make(chan os.Signal, 1)
	sigCh := make(chan os.Signal, 1)
	shutdownDone := make(chan struct{})

	signal.Notify(sighupCh, syscall.SIGHUP)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Handle SIGHUP for hot-reload
	go func() {
		for {
			select {
			case <-sighupCh:
				log.Info("SIGHUP received, reloading configuration")

				newCfg := config.LoadOrDefault(cfgFile)
				if err := newCfg.Validate(); err != nil {
					log.Error("invalid configuration, reload aborted", "error", err)
					continue
				}

				srv.ReloadConfig(newCfg)
			case <-shutdownDone:
				return
			}
		}
	}()

	// Handle shutdown signals
	go func() {
		<-sigCh

		log.Info("shutdown signal received")

		// Stop receiving signals
		signal.Stop(sighupCh)
		signal.Stop(sigCh)
		close(shutdownDone)

		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer shutdownCancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Error("server shutdown error", "error", err)
		}

		// Stop storage (saves final state)
		if err := store.Stop(); err != nil {
			log.Error("storage shutdown error", "error", err)
		}

		// Stop learning engine
		le.Stop()

		agg.Stop()
		cancel()
	}()

	log.Info("capfox ready", "addr", srv.Addr())

	if err := srv.Start(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("server error: %w", err)
	}

	log.Info("capfox stopped")
	return nil
}

func writePIDFile(path string) error {
	pid := os.Getpid()
	return os.WriteFile(path, []byte(fmt.Sprintf("%d", pid)), 0644)
}

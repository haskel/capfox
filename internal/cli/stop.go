package cli

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/haskel/capfox/internal/config"
)

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the running capfox server",
	Long:  `Stop the capfox server by sending SIGTERM to the process specified in the PID file.`,
	RunE:  runStop,
}

var pidFile string

func init() {
	stopCmd.Flags().StringVar(&pidFile, "pid-file", "", "PID file path (overrides config)")
	rootCmd.AddCommand(stopCmd)
}

func runStop(cmd *cobra.Command, args []string) error {
	// Determine PID file path
	pidPath := pidFile
	if pidPath == "" {
		cfg := config.LoadOrDefault(cfgFile)
		pidPath = cfg.Server.PIDFile
	}

	if pidPath == "" {
		return fmt.Errorf("no PID file specified (use --pid-file or configure in config)")
	}

	// Read PID from file
	data, err := os.ReadFile(pidPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("PID file not found: %s (server may not be running)", pidPath)
		}
		return fmt.Errorf("failed to read PID file: %w", err)
	}

	pidStr := strings.TrimSpace(string(data))
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		return fmt.Errorf("invalid PID in file: %s", pidStr)
	}

	// Find the process
	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("process not found: %d", pid)
	}

	// Send SIGTERM
	if err := process.Signal(syscall.SIGTERM); err != nil {
		return fmt.Errorf("failed to send signal: %w", err)
	}

	if !jsonOut {
		fmt.Printf("Sent SIGTERM to process %d\n", pid)
	} else {
		fmt.Printf(`{"status":"stopped","pid":%d}`+"\n", pid)
	}

	return nil
}

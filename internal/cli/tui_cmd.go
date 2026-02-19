package cli

import (
	"time"

	"github.com/haskel/capfox/internal/cli/tui"
	"github.com/spf13/cobra"
)

var (
	refreshInterval time.Duration
)

var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Launch interactive TUI dashboard",
	Long: `Launch an interactive terminal user interface for monitoring
system resources in real-time.

Examples:
  capfox tui                    # Basic launch with default settings
  capfox tui --refresh 500ms    # Faster refresh rate
  capfox tui --host 10.0.0.1    # Connect to remote server`,
	RunE: runTUI,
}

func init() {
	tuiCmd.Flags().DurationVar(&refreshInterval, "refresh", time.Second, "dashboard refresh interval")
	rootCmd.AddCommand(tuiCmd)
}

func runTUI(cmd *cobra.Command, args []string) error {
	config := tui.Config{
		ServerURL:       GetServerURL(),
		RefreshInterval: refreshInterval,
		User:            user,
		Password:        password,
	}

	return tui.Run(config)
}

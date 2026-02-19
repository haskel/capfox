package cli

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/spf13/cobra"
)

var statsCmd = &cobra.Command{
	Use:   "stats [task]",
	Short: "Get task statistics from the learning engine",
	Long: `Query the capfox server for learned task statistics.

Examples:
  capfox stats              # Show all task stats
  capfox stats video_encoding  # Show stats for specific task`,
	Args: cobra.MaximumNArgs(1),
	RunE: runStats,
}

func init() {
	rootCmd.AddCommand(statsCmd)
}

type taskStats struct {
	Task         string  `json:"task"`
	Count        int64   `json:"count"`
	AvgCPUDelta  float64 `json:"avg_cpu_delta"`
	AvgMemDelta  float64 `json:"avg_mem_delta"`
	AvgGPUDelta  float64 `json:"avg_gpu_delta,omitempty"`
	AvgVRAMDelta float64 `json:"avg_vram_delta,omitempty"`
}

type allStats struct {
	Tasks      map[string]*taskStats `json:"tasks"`
	TotalTasks int64                 `json:"total_tasks"`
}

func runStats(cmd *cobra.Command, args []string) error {
	client := NewClient()

	path := "/stats"
	if len(args) > 0 {
		path += "?task=" + args[0]
	}

	data, status, err := client.Get(path)
	if err != nil {
		return fmt.Errorf("failed to get stats: %w", err)
	}

	if status == http.StatusNotFound {
		return fmt.Errorf("task not found: %s", args[0])
	}

	if status != http.StatusOK {
		return fmt.Errorf("server returned status %d: %s", status, string(data))
	}

	if jsonOut {
		fmt.Println(string(data))
		return nil
	}

	// Pretty print
	if len(args) > 0 {
		// Single task
		var stats taskStats
		if err := json.Unmarshal(data, &stats); err != nil {
			return err
		}
		printTaskStats(&stats)
	} else {
		// All tasks
		var all allStats
		if err := json.Unmarshal(data, &all); err != nil {
			return err
		}

		fmt.Printf("=== Task Statistics ===\n")
		fmt.Printf("Total observations: %d\n\n", all.TotalTasks)

		if len(all.Tasks) == 0 {
			fmt.Println("No tasks recorded yet.")
			return nil
		}

		for _, stats := range all.Tasks {
			printTaskStats(stats)
			fmt.Println()
		}
	}

	return nil
}

func printTaskStats(stats *taskStats) {
	fmt.Printf("Task: %s\n", stats.Task)
	fmt.Printf("  Observations: %d\n", stats.Count)
	fmt.Printf("  Avg CPU delta:  %+.2f%%\n", stats.AvgCPUDelta)
	fmt.Printf("  Avg Memory delta: %+.2f%%\n", stats.AvgMemDelta)
	if stats.AvgGPUDelta != 0 {
		fmt.Printf("  Avg GPU delta:  %+.2f%%\n", stats.AvgGPUDelta)
	}
	if stats.AvgVRAMDelta != 0 {
		fmt.Printf("  Avg VRAM delta: %+.2f%%\n", stats.AvgVRAMDelta)
	}
}

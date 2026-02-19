package cli

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Get current server status and resource metrics",
	Long:  `Query the running capfox server for current resource utilization.`,
	RunE:  runStatus,
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

func runStatus(cmd *cobra.Command, args []string) error {
	client := NewClient()

	data, status, err := client.Get("/status")
	if err != nil {
		return fmt.Errorf("failed to get status: %w", err)
	}

	if status != http.StatusOK {
		return fmt.Errorf("server returned status %d: %s", status, string(data))
	}

	if jsonOut {
		fmt.Println(string(data))
		return nil
	}

	// Pretty print
	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		return err
	}

	fmt.Println("=== System Status ===")

	if cpu, ok := result["cpu"].(map[string]any); ok {
		fmt.Printf("\nCPU:\n")
		fmt.Printf("  Usage: %.1f%%\n", cpu["usage_percent"])
	}

	if mem, ok := result["memory"].(map[string]any); ok {
		fmt.Printf("\nMemory:\n")
		fmt.Printf("  Usage: %.1f%%\n", mem["usage_percent"])
		if total, ok := mem["total_bytes"].(float64); ok {
			fmt.Printf("  Total: %.1f GB\n", total/1024/1024/1024)
		}
		if used, ok := mem["used_bytes"].(float64); ok {
			fmt.Printf("  Used:  %.1f GB\n", used/1024/1024/1024)
		}
	}

	if storage, ok := result["storage"].(map[string]any); ok {
		fmt.Printf("\nStorage:\n")
		for path, info := range storage {
			if diskInfo, ok := info.(map[string]any); ok {
				total := diskInfo["total_bytes"].(float64) / 1024 / 1024 / 1024
				free := diskInfo["free_bytes"].(float64) / 1024 / 1024 / 1024
				fmt.Printf("  %s: %.1f GB free / %.1f GB total\n", path, free, total)
			}
		}
	}

	if gpus, ok := result["gpus"].([]any); ok && len(gpus) > 0 {
		fmt.Printf("\nGPU:\n")
		for i, gpu := range gpus {
			if g, ok := gpu.(map[string]any); ok {
				fmt.Printf("  GPU %d: %.1f%% usage", i, g["usage_percent"])
				if vramTotal, ok := g["vram_total_bytes"].(float64); ok && vramTotal > 0 {
					vramUsed := g["vram_used_bytes"].(float64)
					fmt.Printf(", VRAM: %.1f / %.1f GB",
						vramUsed/1024/1024/1024,
						vramTotal/1024/1024/1024)
				}
				fmt.Println()
			}
		}
	}

	if proc, ok := result["process"].(map[string]any); ok {
		fmt.Printf("\nProcesses:\n")
		fmt.Printf("  Total: %.0f\n", proc["total_processes"])
		fmt.Printf("  Threads: %.0f\n", proc["total_threads"])
	}

	return nil
}

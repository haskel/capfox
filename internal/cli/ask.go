package cli

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/spf13/cobra"
)

var askCmd = &cobra.Command{
	Use:   "ask <task>",
	Short: "Ask if a task can be run",
	Long: `Ask the capfox server if a task can be run based on current resource availability.

Examples:
  capfox ask video_encoding
  capfox ask video_encoding --complexity 100
  capfox ask ml_training --complexity 500 --reason`,
	Args: cobra.ExactArgs(1),
	RunE: runAsk,
}

var (
	complexity int
	showReason bool
	cpuEst     float64
	memEst     float64
	gpuEst     float64
	vramEst    float64
)

func init() {
	askCmd.Flags().IntVar(&complexity, "complexity", 0, "task complexity in parrots")
	askCmd.Flags().BoolVar(&showReason, "reason", false, "show denial reasons")
	askCmd.Flags().Float64Var(&cpuEst, "cpu", 0, "estimated CPU usage percent")
	askCmd.Flags().Float64Var(&memEst, "mem", 0, "estimated memory usage percent")
	askCmd.Flags().Float64Var(&gpuEst, "gpu", 0, "estimated GPU usage percent")
	askCmd.Flags().Float64Var(&vramEst, "vram", 0, "estimated VRAM usage percent")
	rootCmd.AddCommand(askCmd)
}

type askRequest struct {
	Task       string            `json:"task"`
	Complexity int               `json:"complexity,omitempty"`
	Resources  *resourceEstimate `json:"resources,omitempty"`
}

type resourceEstimate struct {
	CPU    float64 `json:"cpu,omitempty"`
	Memory float64 `json:"memory,omitempty"`
	GPU    float64 `json:"gpu,omitempty"`
	VRAM   float64 `json:"vram,omitempty"`
}

type askResponse struct {
	Allowed bool     `json:"allowed"`
	Reasons []string `json:"reasons,omitempty"`
}

func runAsk(cmd *cobra.Command, args []string) error {
	task := args[0]

	req := askRequest{
		Task:       task,
		Complexity: complexity,
	}

	// Add resource estimates if provided
	if cpuEst > 0 || memEst > 0 || gpuEst > 0 || vramEst > 0 {
		req.Resources = &resourceEstimate{
			CPU:    cpuEst,
			Memory: memEst,
			GPU:    gpuEst,
			VRAM:   vramEst,
		}
	}

	client := NewClient()

	path := "/ask"
	if showReason {
		path += "?reason=true"
	}

	data, status, err := client.Post(path, req)
	if err != nil {
		return fmt.Errorf("failed to ask: %w", err)
	}

	var resp askResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if jsonOut {
		fmt.Println(string(data))
	} else {
		if resp.Allowed {
			fmt.Printf("✓ Task '%s' is ALLOWED\n", task)
		} else {
			fmt.Printf("✗ Task '%s' is DENIED\n", task)
			if len(resp.Reasons) > 0 {
				fmt.Println("Reasons:")
				for _, reason := range resp.Reasons {
					fmt.Printf("  - %s\n", reason)
				}
			}
		}
	}

	// Exit with 75 (EX_TEMPFAIL) if denied - consistent with 'run' command
	if status == http.StatusServiceUnavailable {
		os.Exit(75)
	}

	return nil
}

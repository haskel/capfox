package cli

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/spf13/cobra"
)

var notifyCmd = &cobra.Command{
	Use:   "notify <task>",
	Short: "Notify server about task start",
	Long: `Notify the capfox server that a task has started.
This allows the learning engine to observe the task's impact on resources.

Examples:
  capfox notify video_encoding
  capfox notify ml_training --complexity 500`,
	Args: cobra.ExactArgs(1),
	RunE: runNotify,
}

func init() {
	notifyCmd.Flags().IntVar(&complexity, "complexity", 0, "task complexity in parrots")
	rootCmd.AddCommand(notifyCmd)
}

type notifyRequest struct {
	Task       string `json:"task"`
	Complexity int    `json:"complexity,omitempty"`
}

type notifyResponse struct {
	Received bool   `json:"received"`
	Task     string `json:"task"`
}

func runNotify(cmd *cobra.Command, args []string) error {
	task := args[0]

	req := notifyRequest{
		Task:       task,
		Complexity: complexity,
	}

	client := NewClient()

	data, status, err := client.Post("/task/notify", req)
	if err != nil {
		return fmt.Errorf("failed to notify: %w", err)
	}

	if status != http.StatusOK {
		return fmt.Errorf("server returned status %d: %s", status, string(data))
	}

	var resp notifyResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if jsonOut {
		fmt.Println(string(data))
	} else {
		if resp.Received {
			fmt.Printf("✓ Task '%s' notification received\n", task)
		} else {
			fmt.Printf("✗ Task '%s' notification failed\n", task)
		}
	}

	return nil
}

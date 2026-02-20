package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:   "run [flags] <command> [args...]",
	Short: "Run command if capacity available",
	Long: `Run a command only if server has available capacity.
Works like 'time' or 'nice' - wrap any command with capfox run.

Exit codes:
  0-125  Command's exit code
  75     No capacity available (command not started)
  126    Command not executable
  127    Command not found`,
	Example: `  capfox run ./script.sh
  capfox run --task ml python train.py
  capfox run --complexity 100 make build
  capfox run --cpu 50 --mem 30 ./heavy.sh`,
	Args: cobra.MinimumNArgs(1),
	RunE: runRun,
}

var (
	runTask       string
	runComplexity int
	runCPU        float64
	runMem        float64
	runGPU        float64
	runVRAM       float64
	runReason     bool
	runQuiet      bool
)

func init() {
	runCmd.Flags().StringVar(&runTask, "task", "", "task name for /ask (default: command name)")
	runCmd.Flags().IntVar(&runComplexity, "complexity", 0, "task complexity in parrots")
	runCmd.Flags().Float64Var(&runCPU, "cpu", 0, "estimated CPU usage percent")
	runCmd.Flags().Float64Var(&runMem, "mem", 0, "estimated memory usage percent")
	runCmd.Flags().Float64Var(&runGPU, "gpu", 0, "estimated GPU usage percent")
	runCmd.Flags().Float64Var(&runVRAM, "vram", 0, "estimated VRAM usage percent")
	runCmd.Flags().BoolVar(&runReason, "reason", false, "show denial reasons")
	runCmd.Flags().BoolVar(&runQuiet, "quiet", false, "suppress capfox output")
	rootCmd.AddCommand(runCmd)
}

const (
	exitNoCapacity     = 75  // EX_TEMPFAIL from sysexits.h
	exitNotExecutable  = 126
	exitCommandNotFound = 127
)

func runRun(cmd *cobra.Command, args []string) error {
	// 1. Determine task name
	taskName := runTask
	if taskName == "" {
		taskName = filepath.Base(args[0])
	}

	// 2. Build ask request
	req := askRequest{
		Task:       taskName,
		Complexity: runComplexity,
	}

	// Add resource estimates if provided
	if runCPU > 0 || runMem > 0 || runGPU > 0 || runVRAM > 0 {
		req.Resources = &resourceEstimate{
			CPU:    runCPU,
			Memory: runMem,
			GPU:    runGPU,
			VRAM:   runVRAM,
		}
	}

	// 3. Call /ask endpoint
	client := NewClient()

	path := "/ask?reason=true"
	data, _, err := client.Post(path, req)
	if err != nil {
		if !runQuiet {
			fmt.Fprintf(os.Stderr, "capfox: failed to check capacity: %v\n", err)
		}
		// If we can't reach the server, still try to run the command
		// This is a design decision - fail open
		return executeCommand(args)
	}

	var resp askResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		if !runQuiet {
			fmt.Fprintf(os.Stderr, "capfox: failed to parse response: %v\n", err)
		}
		return executeCommand(args)
	}

	// 4. If denied, exit with code 75
	if !resp.Allowed {
		if !runQuiet {
			fmt.Fprintf(os.Stderr, "capfox: denied\n")
			if runReason && len(resp.Reasons) > 0 {
				for _, r := range resp.Reasons {
					fmt.Fprintf(os.Stderr, "  - %s\n", r)
				}
			}
		}
		os.Exit(exitNoCapacity)
	}

	// 5. Notify server about task start (only if complexity is specified)
	if runComplexity > 0 {
		notifyReq := notifyRequest{
			Task:       taskName,
			Complexity: runComplexity,
		}
		// Fire-and-forget: ignore errors
		_, _, _ = client.Post("/task/notify", notifyReq)
	}

	// 6. If allowed, execute command
	if !runQuiet {
		fmt.Fprintf(os.Stderr, "capfox: allowed\n")
	}

	return executeCommand(args)
}

func executeCommand(args []string) error {
	execCmd := exec.Command(args[0], args[1:]...)
	execCmd.Stdin = os.Stdin
	execCmd.Stdout = os.Stdout
	execCmd.Stderr = os.Stderr

	err := execCmd.Run()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		// Check if it's a "not found" error
		if execErr, ok := err.(*exec.Error); ok {
			if execErr.Err == exec.ErrNotFound {
				os.Exit(exitCommandNotFound)
			}
		}
		// For permission denied or not executable
		os.Exit(exitNotExecutable)
	}

	return nil
}

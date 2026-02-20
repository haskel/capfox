package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	// Global flags
	cfgFile  string
	host     string
	port     int
	jsonOut  bool
	verbose  bool
	user     string
	password string

	// Version info (set from main)
	Version = "0.2.0"
)

// rootCmd represents the base command
var rootCmd = &cobra.Command{
	Use:   "capfox",
	Short: "Server resource monitoring and capacity management",
	Long: `Capfox is a server-side monitoring utility that tracks system resources
(CPU, GPU, RAM, VRAM, storage, processes) and provides an API for other
services to check if they can run tasks based on current resource availability.`,
}

// Execute runs the root command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file path")
	rootCmd.PersistentFlags().StringVar(&host, "host", "localhost", "server host")
	rootCmd.PersistentFlags().IntVarP(&port, "port", "p", 8080, "server port")
	rootCmd.PersistentFlags().BoolVar(&jsonOut, "json", false, "output in JSON format")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
	rootCmd.PersistentFlags().StringVar(&user, "user", "", "auth username")
	rootCmd.PersistentFlags().StringVar(&password, "password", "", "auth password")
}

// SetVersion sets the version for the CLI
func SetVersion(v string) {
	Version = v
	rootCmd.Version = v
}

// GetServerURL returns the server URL based on flags
func GetServerURL() string {
	return fmt.Sprintf("http://%s:%d", host, port)
}

// GetConfigFile returns the config file path
func GetConfigFile() string {
	return cfgFile
}

// IsJSON returns whether JSON output is enabled
func IsJSON() bool {
	return jsonOut
}

// IsVerbose returns whether verbose output is enabled
func IsVerbose() bool {
	return verbose
}

// GetAuth returns auth credentials
func GetAuth() (string, string) {
	return user, password
}

package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/haskel/capfox/internal/config"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Show current configuration",
	Long:  `Display the current configuration (loaded from file or defaults).`,
	RunE:  runConfig,
}

var validateOnly bool

func init() {
	configCmd.Flags().BoolVar(&validateOnly, "validate", false, "only validate config, don't print")
	rootCmd.AddCommand(configCmd)
}

func runConfig(cmd *cobra.Command, args []string) error {
	cfg := config.LoadOrDefault(cfgFile)

	// Validate
	if err := cfg.Validate(); err != nil {
		if jsonOut {
			fmt.Printf(`{"valid":false,"error":%q}`+"\n", err.Error())
		} else {
			fmt.Printf("Configuration invalid: %v\n", err)
		}
		return err
	}

	if validateOnly {
		if jsonOut {
			fmt.Println(`{"valid":true}`)
		} else {
			fmt.Println("Configuration is valid")
		}
		return nil
	}

	// Print config
	if jsonOut {
		data, err := json.MarshalIndent(cfg, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(data))
	} else {
		data, err := yaml.Marshal(cfg)
		if err != nil {
			return err
		}
		fmt.Println(string(data))
	}

	return nil
}

package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/flotio-dev/cli/internal/config"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var configureCmd = &cobra.Command{
	Use:   "configure",
	Short: "Configure Flotio CLI defaults",
	Long: `Set default values stored in ~/.flotio/config.yaml.
The --host flag sets the default API host (accepts scheme://host).`,
	Example: `  flotio configure --host api.flotio.ovh
  flotio configure --host http://localhost:8080`,
	RunE: func(cmd *cobra.Command, args []string) error {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("home dir: %w", err)
		}
		dir := filepath.Join(home, ".flotio")
		path := filepath.Join(dir, "config.yaml")

		// Load existing config
		var current map[string]interface{}
		data, err := os.ReadFile(path)
		if err == nil {
			yaml.Unmarshal(data, &current)
		}
		if current == nil {
			current = map[string]interface{}{}
		}

		host, _ := cmd.Flags().GetString("host")
		if host != "" {
			current["host"] = host
			fmt.Printf("✓ Default host set to: %s\n", host)
		}

		if host == "" {
			// Show current config
			if cfg, err := config.Load(); err == nil {
				fmt.Printf("Current configuration (%s):\n", path)
				fmt.Printf("  host: %s\n", cfg.ResolveHost())
			}
			return nil
		}

		os.MkdirAll(dir, 0700)
		out, _ := yaml.Marshal(current)
		if err := os.WriteFile(path, out, 0644); err != nil {
			return fmt.Errorf("writing config: %w", err)
		}
		fmt.Printf("  Config saved to %s\n", path)
		return nil
	},
}

func init() {
	configureCmd.Flags().String("host", "", "Default API host (e.g. api.flotio.ovh or http://localhost:8080)")
	rootCmd.AddCommand(configureCmd)
}

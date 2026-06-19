package cmd

import (
	"fmt"

	"github.com/flotio-dev/cli/internal/config"
	"github.com/flotio-dev/cli/pkg/client"
	"github.com/flotio-dev/cli/pkg/display"
	apiclient "github.com/flotio-dev/cli/pkg/api/client"
	"github.com/spf13/cobra"
)

var (
	cfg *config.Config
	api *apiclient.FlotioAPI
)

// rootCmd is the base command.
var rootCmd = &cobra.Command{
	Use:   "flotio",
	Short: "Flotio CLI - cloud infrastructure management",
	Long: `Flotio CLI is a command-line tool for managing Flotio cloud infrastructure,
deploying projects, and interacting with the Flotio platform.

Getting started:

  flotio login --email you@example.com --password pass
  flotio project list
  flotio build start <project-id> --platform android --mode release

Use "flotio <command> --help" for details on any command.`,
	Example: `  # Log in to Flotio
  flotio login --email me@example.com --password s3cret

  # List your projects
  flotio project list

  # Create a project and trigger a build
  flotio project create "My App" --repo https://github.com/user/repo
  flotio build start 1 --branch main --platform android --mode release

  # Manage signing keys
  flotio keystore create "release" --file keystore.jks --alias mykey

  # Manage Google Play credentials
  flotio play create "play-store" --file service-account.json

  # Manage environment variables
  flotio env create DATABASE_URL "postgres://..." --type env`,
	SilenceErrors: true,
	SilenceUsage:  false,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		var err error
		cfg, err = config.Load()
		if err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}
		// Propagate the --output flag to the display layer.
		out, _ := cmd.Flags().GetString("output")
		display.SetOutputFormat(out)
		// Build the API client (auth is injected from stored tokens).
		api = client.New(cfg.ResolveHost())
		return nil
	},
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVar(&config.FlagConfigPath, "config", "", "path to config file (default: $HOME/.flotio/config.yaml)")
	rootCmd.PersistentFlags().StringVar(&config.FlagHost, "host", "", "API host (default: api.flotio.ovh, accepts scheme://host e.g. http://localhost:8080)")
	rootCmd.PersistentFlags().StringP("output", "o", "table", "Output format: table or json")
}

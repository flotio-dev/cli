package cmd

import (
	"fmt"

	"github.com/flotio-dev/cli/pkg/api/client/flutter"
	"github.com/spf13/cobra"
)

var flutterCmd = &cobra.Command{
	Use:     "flutter",
	Short:   "Flutter version info",
	Long:    `List available Flutter versions on the Flotio platform.`,
	Example: `  flotio flutter versions`,
}

var flutterVersionsCmd = &cobra.Command{
	Use:   "versions",
	Short: "List available Flutter versions",
	RunE: func(cmd *cobra.Command, args []string) error {
		params := flutter.NewGetFlutterVersionsParams()
		resp, err := api.Flutter.GetFlutterVersions(params)
		if err != nil {
			return fmt.Errorf("getting Flutter versions: %w", err)
		}
		list := resp.GetPayload()
		for _, v := range list.Versions {
			fmt.Printf("  %-8s %s\n", v.Channel, v.Version)
		}
		return nil
	},
}

func init() {
	flutterCmd.AddCommand(flutterVersionsCmd)
	rootCmd.AddCommand(flutterCmd)
}

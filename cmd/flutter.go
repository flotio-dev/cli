package cmd

import (
	"fmt"

	"github.com/flotio-dev/cli/pkg/api/client/flutter"
	"github.com/flotio-dev/cli/pkg/display"
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
		if list == nil || len(list.Versions) == 0 {
			display.NoResults("Flutter versions")
			return nil
		}
		table := &display.Table{
			Columns: []display.Column{
				{Header: "Channel", Width: 8},
				{Header: "Version", Width: 12},
			},
		}
		for _, v := range list.Versions {
			table.AddRow(v.Channel, v.Version)
		}
		table.Render()
		return nil
	},
}

func init() {
	flutterCmd.AddCommand(flutterVersionsCmd)
	rootCmd.AddCommand(flutterCmd)
}

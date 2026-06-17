package cmd

import (
	"fmt"

	"github.com/flotio-dev/cli/pkg/build"
	"github.com/spf13/cobra"
)

// versionCmd represents the version command.
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version information",
	Long:  `Print the version, commit hash, and build date of the Flotio CLI.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(build.Summary())
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

package cmd

import (
	"encoding/base64"
	"fmt"
	"os"

	"github.com/flotio-dev/cli/pkg/client"
	"github.com/spf13/cobra"
)

var playCmd = &cobra.Command{
	Use:   "play",
	Short: "Manage Google Play credentials",
	Long: `Upload, list, and delete Google Play service account credentials
used for Android app publishing.`,
	Example: `  flotio play list
  flotio play create "release" --file service-account.json
  flotio play delete 1`,
}

var playListCmd = &cobra.Command{
	Use:     "list",
	Short:   "List Google Play credentials",
	Example: `  flotio play list`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if !client.IsLoggedIn() {
			return fmt.Errorf("not logged in")
		}
		var raw map[string]interface{}
		if err := client.GetJSON(cfg.ResolveHost(), "/google-play-credentials", &raw); err != nil {
			return fmt.Errorf("listing credentials: %w", err)
		}
		items, _ := client.ExtractList(raw)
		if len(items) == 0 {
			fmt.Println("No credentials found.")
			return nil
		}
		for _, raw := range items {
			c, ok := raw.(map[string]interface{})
			if !ok {
				continue
			}
			fmt.Printf("  [%v] %v\n", c["id"], c["name"])
		}
		return nil
	},
}

var playCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Upload Google Play service account credentials",
	Long: `Upload a Google Play service account JSON key file.
The file is stored securely and used during Android builds.`,
	Example: `  flotio play create "my-play-creds" --file ~/service-account.json`,
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if !client.IsLoggedIn() {
			return fmt.Errorf("not logged in")
		}
		name := args[0]
		file, _ := cmd.Flags().GetString("file")
		if file == "" {
			return fmt.Errorf("--file is required (path to Google Play service account JSON)")
		}

		data, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("reading credentials file: %w", err)
		}
		encoded := base64.StdEncoding.EncodeToString(data)

		body := map[string]interface{}{
			"name":        name,
			"credentials": encoded,
		}
		var result map[string]interface{}
		if err := client.PostJSON(cfg.ResolveHost(), "/google-play-credentials", body, &result); err != nil {
			return fmt.Errorf("creating credentials: %w", err)
		}
		fmt.Printf("✓ Play credentials created: [%v] %v\n", result["id"], result["name"])
		return nil
	},
}

var playDeleteCmd = &cobra.Command{
	Use:     "delete <id>",
	Short:   "Delete Google Play credentials",
	Example: `  flotio play delete 1`,
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if !client.IsLoggedIn() {
			return fmt.Errorf("not logged in")
		}
		if err := client.DeleteJSON(cfg.ResolveHost(), "/google-play-credentials/"+args[0]); err != nil {
			return fmt.Errorf("deleting credentials: %w", err)
		}
		fmt.Printf("✓ Play credentials %s deleted\n", args[0])
		return nil
	},
}

func init() {
	playCreateCmd.Flags().String("file", "", "Path to Google Play service account JSON")

	playCmd.AddCommand(playListCmd)
	playCmd.AddCommand(playCreateCmd)
	playCmd.AddCommand(playDeleteCmd)
	rootCmd.AddCommand(playCmd)
}

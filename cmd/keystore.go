package cmd

import (
	"encoding/base64"
	"fmt"
	"os"

	"github.com/flotio-dev/cli/pkg/client"
	"github.com/flotio-dev/cli/pkg/display"
	"github.com/spf13/cobra"
)

var keystoreCmd = &cobra.Command{
	Use:   "keystore",
	Short: "Manage signing keystores",
	Long: `Upload, list, and delete Android signing keystores.
Keystores are used to sign APK/AAB builds for Google Play distribution.`,
	Example: `  flotio keystore list
  flotio keystore create "release" --file keystore.jks --alias mykey
  flotio keystore delete 1`,
}

var keystoreListCmd = &cobra.Command{
	Use:     "list",
	Short:   "List keystores",
	Example: `  flotio keystore list`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if !client.IsLoggedIn() {
			return fmt.Errorf("not logged in")
		}
		var raw map[string]interface{}
		if err := client.GetJSON(cfg.ResolveHost(), "/keystore", &raw); err != nil {
			return fmt.Errorf("listing keystores: %w", err)
		}
		items, _ := client.ExtractList(raw)
		if display.JSONOutput() {
			if len(items) == 0 {
				fmt.Println("[]")
			} else {
				display.PrintJSON(items)
			}
			return nil
		}
		if len(items) == 0 {
			display.NoResults("keystores")
			return nil
		}
		table := &display.Table{
			Columns: []display.Column{
				{Header: "ID", Width: 5, Align: 1},
				{Header: "Name", Max: 30},
				{Header: "Alias", Max: 20},
			},
		}
		for _, raw := range items {
			k, ok := raw.(map[string]interface{})
			if !ok {
				continue
			}
			table.AddRow(
				fmt.Sprintf("%v", k["ID"]),
				fmt.Sprintf("%v", k["name"]),
				fmt.Sprintf("%v", k["key_alias"]),
			)
		}
		table.Render()
		return nil
	},
}

var keystoreCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Upload a new signing keystore",
	Long: `Upload a Java keystore (.jks or .keystore) for Android app signing.
Requires the file path, key alias, and optionally store/key passwords.`,
	Example: `  flotio keystore create "release" --file keystore.jks --alias mykey --store-password storepass --key-password keypass`,
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if !client.IsLoggedIn() {
			return fmt.Errorf("not logged in")
		}
		name := args[0]
		file, _ := cmd.Flags().GetString("file")
		alias, _ := cmd.Flags().GetString("alias")
		storePass, _ := cmd.Flags().GetString("store-password")
		keyPass, _ := cmd.Flags().GetString("key-password")

		if file == "" || alias == "" {
			return fmt.Errorf("--file and --alias are required")
		}

		data, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("reading keystore file: %w", err)
		}

		body := map[string]interface{}{
			"name":           name,
			"keystore_file":  base64.StdEncoding.EncodeToString(data),
			"key_alias":      alias,
			"store_password": storePass,
			"key_password":   keyPass,
		}

		var wrapper map[string]interface{}
		if err := client.PostJSON(cfg.ResolveHost(), "/keystore", body, &wrapper); err != nil {
			return fmt.Errorf("creating keystore: %w", err)
		}
		ks, ok := wrapper["keystore"].(map[string]interface{})
		if !ok {
			return fmt.Errorf("unexpected response format")
		}
		display.SuccessPrint("Keystore created: [%v] %v", ks["ID"], ks["name"])
		return nil
	},
}

var keystoreDeleteCmd = &cobra.Command{
	Use:     "delete <id>",
	Short:   "Delete a keystore",
	Example: `  flotio keystore delete 1`,
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if !client.IsLoggedIn() {
			return fmt.Errorf("not logged in")
		}
		if err := client.DeleteJSON(cfg.ResolveHost(), "/keystore/"+args[0]); err != nil {
			return fmt.Errorf("deleting keystore: %w", err)
		}
		display.SuccessPrint("Keystore %s deleted", args[0])
		return nil
	},
}

func init() {
	keystoreCreateCmd.Flags().String("file", "", "Path to keystore file (.jks, .keystore)")
	keystoreCreateCmd.Flags().String("alias", "", "Key alias")
	keystoreCreateCmd.Flags().String("store-password", "", "Store password")
	keystoreCreateCmd.Flags().String("key-password", "", "Key password")

	keystoreCmd.AddCommand(keystoreListCmd)
	keystoreCmd.AddCommand(keystoreCreateCmd)
	keystoreCmd.AddCommand(keystoreDeleteCmd)
	rootCmd.AddCommand(keystoreCmd)
}

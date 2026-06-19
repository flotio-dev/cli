package cmd

import (
	"encoding/base64"
	"fmt"
	"os"
	"strings"

	"github.com/flotio-dev/cli/internal/config"
	"github.com/flotio-dev/cli/pkg/client"
	"github.com/flotio-dev/cli/pkg/display"
	"github.com/spf13/cobra"
)

// resolveFileValue handles the @file syntax and base64 encoding for file uploads.
func resolveFileValue(value, assetType string) (string, bool, error) {
	if strings.HasPrefix(value, "@") {
		data, err := os.ReadFile(value[1:])
		if err != nil {
			return "", false, fmt.Errorf("reading file %s: %w", value[1:], err)
		}
		return base64.StdEncoding.EncodeToString(data), true, nil
	}
	if assetType == "file" {
		return base64.StdEncoding.EncodeToString([]byte(value)), true, nil
	}
	return value, false, nil
}

var envCmd = &cobra.Command{
	Use:   "env",
	Short: "Manage environment variables and files",
	Long:  `Create, list, update, and delete environment assets (variables and files).`,
	Example: `  flotio env list
  flotio env list --project 1
  flotio env create DATABASE_URL "postgres://..."
  flotio env create .env.production @./.env.prod --type file --path .env
  flotio env update 1 --value "new-value"
  flotio env update 1 --value @./new-key.jks --type file
  flotio env delete 1`,
}

var envListCmd = &cobra.Command{
	Use:     "list",
	Short:   "List environment assets",
	Example: `  flotio env list`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if !client.IsLoggedIn() {
			return fmt.Errorf("not logged in")
		}
		url := "/env"
		pid, _ := cmd.Flags().GetInt64("project")
		if pid == 0 {
			if id, err := config.ResolveProjectID(0); err == nil {
				pid = id
			}
		}
		if pid > 0 {
			url = fmt.Sprintf("/env?project_id=%d", pid)
		}

		var raw map[string]interface{}
		if err := client.GetJSON(cfg.ResolveHost(), url, &raw); err != nil {
			return fmt.Errorf("listing env: %w", err)
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
			display.NoResults("environment assets")
			return nil
		}

		table := &display.Table{
			Columns: []display.Column{
				{Header: "ID", Width: 5, Align: 1},
				{Header: "Type", Width: 6},
				{Header: "Key", Max: 30},
				{Header: "Value", Max: 40},
				{Header: "Project", Width: 9},
			},
		}
		for _, raw := range items {
			item, ok := raw.(map[string]interface{})
			if !ok {
				continue
			}
			projectStr := "-"
			if pid, ok := item["project_id"]; ok && pid != nil {
				projectStr = fmt.Sprintf("%v", pid)
			}
			val := display.Truncate(fmt.Sprintf("%v", item["value"]), 40)
			table.AddRow(
				fmt.Sprintf("%v", item["ID"]),
				fmt.Sprintf("%v", item["type"]),
				fmt.Sprintf("%v", item["key"]),
				val,
				projectStr,
			)
		}
		table.Render()
		return nil
	},
}

var envGetCmd = &cobra.Command{
	Use:     "get <id>",
	Short:   "Get an environment asset by ID",
	Args:    cobra.ExactArgs(1),
	Example: `  flotio env get 1`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if !client.IsLoggedIn() {
			return fmt.Errorf("not logged in")
		}
		var wrapper map[string]interface{}
		if err := client.GetJSON(cfg.ResolveHost(), "/env/"+args[0], &wrapper); err != nil {
			return fmt.Errorf("getting env: %w", err)
		}
		e, ok := wrapper["env"].(map[string]interface{})
		if !ok {
			return fmt.Errorf("unexpected response format")
		}
		if display.JSONOutput() {
			display.PrintJSON(e)
			return nil
		}
		display.HeadingPrint("Environment Asset %v", e["ID"])
		display.KeyValue("Key", "%v", e["key"])
		display.KeyValue("Type", "%v", e["type"])
		display.KeyValue("Value", "%v", e["value"])
		if pid := e["project_id"]; pid != nil {
			display.KeyValue("Project", "%v", pid)
		} else {
			display.KeyValue("Project", "-")
		}
		if p, ok := e["path"]; ok && p != nil && p != "" {
			display.KeyValue("Path", "%v", p)
		}
		if b, ok := e["is_base64"]; ok && b != nil && b != false {
			display.KeyValue("Encoding", "base64")
		}
		return nil
	},
}

var envCreateCmd = &cobra.Command{
	Use:   "create <key> <value>",
	Short: "Create an environment variable or file",
	Args:  cobra.ExactArgs(2),
	Example: `  flotio env create DATABASE_URL "postgres://..."`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if !client.IsLoggedIn() {
			return fmt.Errorf("not logged in")
		}
		key, rawValue := args[0], args[1]
		assetType, _ := cmd.Flags().GetString("type")
		projectID, _ := cmd.Flags().GetInt64("project")
		filePath, _ := cmd.Flags().GetString("path")

		if assetType == "" {
			assetType = "env"
		}
		if projectID == 0 {
			if id, err := config.ResolveProjectID(0); err == nil {
				projectID = id
			}
		}

		value, isBase64, err := resolveFileValue(rawValue, assetType)
		if err != nil {
			return err
		}

		body := map[string]interface{}{
			"key":       key,
			"value":     value,
			"type":      assetType,
			"is_base64": isBase64,
		}
		if projectID > 0 {
			body["project_id"] = projectID
		}
		if filePath != "" {
			body["path"] = filePath
		}

		var wrapper map[string]interface{}
		if err := client.PostJSON(cfg.ResolveHost(), "/env", body, &wrapper); err != nil {
			return fmt.Errorf("creating env: %w", err)
		}
		env, ok := wrapper["env"].(map[string]interface{})
		if !ok {
			return fmt.Errorf("unexpected response format")
		}
		display.SuccessPrint("Env created: [%v] %v", env["ID"], env["key"])
		return nil
	},
}

var envUpdateCmd = &cobra.Command{
	Use:     "update <id>",
	Short:   "Update an environment asset",
	Args:    cobra.ExactArgs(1),
	Example: `  flotio env update 1 --value "new-value"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if !client.IsLoggedIn() {
			return fmt.Errorf("not logged in")
		}
		newValue, _ := cmd.Flags().GetString("value")
		newKey, _ := cmd.Flags().GetString("key")
		newType, _ := cmd.Flags().GetString("type")
		newPath, _ := cmd.Flags().GetString("path")
		forceBase64 := cmd.Flags().Changed("base64")
		isBase64Flag, _ := cmd.Flags().GetBool("base64")

		if newValue == "" && newKey == "" && newType == "" && newPath == "" && !forceBase64 {
			return fmt.Errorf("at least one of --value, --key, --type, --path, --base64 is required")
		}

		var wrapper map[string]interface{}
		if err := client.GetJSON(cfg.ResolveHost(), "/env/"+args[0], &wrapper); err != nil {
			return fmt.Errorf("fetching current env: %w", err)
		}
		current, ok := wrapper["env"].(map[string]interface{})
		if !ok {
			return fmt.Errorf("unexpected response format when fetching env")
		}

		effectiveType := fmt.Sprint(current["type"])
		if newType != "" {
			effectiveType = newType
		}

		body := map[string]interface{}{
			"key":       current["key"],
			"value":     current["value"],
			"type":      current["type"],
			"path":      current["path"],
			"is_base64": current["is_base64"],
		}
		if pid := current["project_id"]; pid != nil {
			body["project_id"] = pid
		}
		if newKey != "" {
			body["key"] = newKey
		}
		if newValue != "" {
			resolved, isB64, err := resolveFileValue(newValue, effectiveType)
			if err != nil {
				return err
			}
			body["value"] = resolved
			body["is_base64"] = isB64
		}
		if newType != "" {
			body["type"] = newType
			if newType == "file" && newValue == "" {
				if v, ok := current["value"].(string); ok {
					body["value"] = base64.StdEncoding.EncodeToString([]byte(v))
					body["is_base64"] = true
				}
			}
		}
		if newPath != "" {
			body["path"] = newPath
		}
		if forceBase64 {
			body["is_base64"] = isBase64Flag
		}

		if err := client.PutJSON(cfg.ResolveHost(), "/env/"+args[0], body, nil); err != nil {
			return fmt.Errorf("updating env: %w", err)
		}
		display.SuccessPrint("Env %s updated", args[0])
		return nil
	},
}

var envDeleteCmd = &cobra.Command{
	Use:     "delete <id>",
	Short:   "Delete an environment asset",
	Args:    cobra.ExactArgs(1),
	Example: `  flotio env delete 1`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if !client.IsLoggedIn() {
			return fmt.Errorf("not logged in")
		}
		if err := client.DeleteJSON(cfg.ResolveHost(), "/env/"+args[0]); err != nil {
			return fmt.Errorf("deleting env: %w", err)
		}
		display.SuccessPrint("Env %s deleted", args[0])
		return nil
	},
}

func init() {
	envCreateCmd.Flags().String("type", "env", "Type: env or file")
	envCreateCmd.Flags().Int64("project", 0, "Project ID to associate with")
	envCreateCmd.Flags().String("path", "", "Target path (for file type)")
	envUpdateCmd.Flags().String("value", "", "New value")
	envUpdateCmd.Flags().String("key", "", "New key name")
	envUpdateCmd.Flags().String("type", "", "New type (env or file)")
	envUpdateCmd.Flags().String("path", "", "New target path (for file type)")
	envUpdateCmd.Flags().Bool("base64", false, "Mark value as base64 encoded")
	envListCmd.Flags().Int64("project", 0, "Filter by project ID")

	envCmd.AddCommand(envListCmd)
	envCmd.AddCommand(envGetCmd)
	envCmd.AddCommand(envCreateCmd)
	envCmd.AddCommand(envUpdateCmd)
	envCmd.AddCommand(envDeleteCmd)
	rootCmd.AddCommand(envCmd)
}

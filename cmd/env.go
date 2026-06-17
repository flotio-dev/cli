package cmd

import (
	"encoding/base64"
	"fmt"
	"os"
	"strings"

	"github.com/flotio-dev/cli/internal/config"
	"github.com/flotio-dev/cli/pkg/client"
	"github.com/spf13/cobra"
)

// resolveFileValue handles the @file syntax and base64 encoding for file uploads.
// If value starts with "@", the referenced file is read and base64-encoded.
// If assetType is "file", the value is always base64-encoded.
// Returns the final value and whether it's base64-encoded.
func resolveFileValue(value, assetType string) (string, bool, error) {
	isFileRef := strings.HasPrefix(value, "@")
	isFileType := assetType == "file"

	if isFileRef {
		filePath := value[1:]
		data, err := os.ReadFile(filePath)
		if err != nil {
			return "", false, fmt.Errorf("reading file %s: %w", filePath, err)
		}
		return base64.StdEncoding.EncodeToString(data), true, nil
	}

	if isFileType {
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

		// Build URL with optional project_id filter
		url := "/env"
		pid, _ := cmd.Flags().GetInt64("project")
		if pid == 0 {
			// Try .flotio.yaml
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
		if len(items) == 0 {
			fmt.Println("No environment assets found.")
			return nil
		}
		for _, raw := range items {
			item, ok := raw.(map[string]interface{})
			if !ok {
				continue
			}
			pid := item["project_id"]
			if pid == nil {
				pid = "-"
			}
			fmt.Printf("  [%v] %-6v %-25v = %v  (project: %v)\n",
				item["id"], item["type"], item["key"], item["value"], pid)
		}
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
		// API wraps response in {"env": {...}}
		e, ok := wrapper["env"].(map[string]interface{})
		if !ok {
			return fmt.Errorf("unexpected response format")
		}
		fmt.Printf("Key:       %v\n", e["key"])
		fmt.Printf("Type:      %v\n", e["type"])
		fmt.Printf("Value:     %v\n", e["value"])
		fmt.Printf("Project:   %v\n", e["project_id"])
		if p, ok := e["path"]; ok && p != nil && p != "" {
			fmt.Printf("Path:      %v\n", p)
		}
		if b, ok := e["is_base64"]; ok && b != nil && b != false {
			fmt.Println("Encoding:  base64")
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
		// Auto-resolve project from .flotio.yaml if not explicitly passed
		if projectID == 0 {
			if id, err := config.ResolveProjectID(0); err == nil {
				projectID = id
			}
		}

		// Resolve @file syntax and base64-encode files
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
		// API wraps response in {"env": {...}}
		env, ok := wrapper["env"].(map[string]interface{})
		if !ok {
			return fmt.Errorf("unexpected response format")
		}
		fmt.Printf("✓ Env created: [%v] %v\n", env["id"], env["key"])
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

		// Fetch current env to preserve all fields (API overwrites everything)
		var wrapper map[string]interface{}
		if err := client.GetJSON(cfg.ResolveHost(), "/env/"+args[0], &wrapper); err != nil {
			return fmt.Errorf("fetching current env: %w", err)
		}
		current, ok := wrapper["env"].(map[string]interface{})
		if !ok {
			return fmt.Errorf("unexpected response format when fetching env")
		}

		// Determine effective type for base64 decision
		effectiveType := fmt.Sprint(current["type"])
		if newType != "" {
			effectiveType = newType
		}

		// Build full update body, preserving existing values
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
			// Resolve @file syntax and base64-encode files
			resolved, isB64, err := resolveFileValue(newValue, effectiveType)
			if err != nil {
				return err
			}
			body["value"] = resolved
			body["is_base64"] = isB64
		}
		if newType != "" {
			body["type"] = newType
			// If switching to file type, re-encode existing value as base64
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
		fmt.Printf("✓ Env %s updated\n", args[0])
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
		fmt.Printf("✓ Env %s deleted\n", args[0])
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

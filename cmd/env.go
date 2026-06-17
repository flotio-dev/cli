package cmd

import (
	"fmt"

	"github.com/flotio-dev/cli/internal/config"
	"github.com/flotio-dev/cli/pkg/client"
	"github.com/spf13/cobra"
)

var envCmd = &cobra.Command{
	Use:   "env",
	Short: "Manage environment variables and files",
	Long:  `Create, list, update, and delete environment assets (variables and files).`,
	Example: `  flotio env list
  flotio env list --project 1
  flotio env create DATABASE_URL "postgres://..." 
  flotio env create .env.production "$(cat .env.prod)" --type file --path .env
  flotio env update 1 --value "new-value"
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
		var e map[string]interface{}
		if err := client.GetJSON(cfg.ResolveHost(), "/env/"+args[0], &e); err != nil {
			return fmt.Errorf("getting env: %w", err)
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
	Use:     "create <key> <value>",
	Short:   "Create an environment variable or file",
	Args:    cobra.ExactArgs(2),
	Example: `  flotio env create DATABASE_URL "postgres://..."`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if !client.IsLoggedIn() {
			return fmt.Errorf("not logged in")
		}
		key, value := args[0], args[1]
		assetType, _ := cmd.Flags().GetString("type")
		projectID, _ := cmd.Flags().GetInt64("project")
		path, _ := cmd.Flags().GetString("path")

		if assetType == "" {
			assetType = "env"
		}
		body := map[string]interface{}{
			"key":   key,
			"value": value,
			"type":  assetType,
		}
		if projectID > 0 {
			body["project_id"] = projectID
		}
		if path != "" {
			body["path"] = path
		}

		var result map[string]interface{}
		if err := client.PostJSON(cfg.ResolveHost(), "/env", body, &result); err != nil {
			return fmt.Errorf("creating env: %w", err)
		}
		fmt.Printf("✓ Env created: [%v] %v\n", result["id"], result["key"])
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
		value, _ := cmd.Flags().GetString("value")
		if value == "" {
			return fmt.Errorf("--value is required for update")
		}

		body := map[string]interface{}{"value": value}
		if err := client.PostJSON(cfg.ResolveHost(), "/env/"+args[0], body, nil); err != nil {
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
	envListCmd.Flags().Int64("project", 0, "Filter by project ID")

	envCmd.AddCommand(envListCmd)
	envCmd.AddCommand(envGetCmd)
	envCmd.AddCommand(envCreateCmd)
	envCmd.AddCommand(envUpdateCmd)
	envCmd.AddCommand(envDeleteCmd)
	rootCmd.AddCommand(envCmd)
}

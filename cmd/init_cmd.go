package cmd

import (
	"fmt"
	"os"
	"strconv"

	"github.com/flotio-dev/cli/internal/config"
	"github.com/flotio-dev/cli/pkg/client"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init [project-id]",
	Short: "Initialize a Flotio project in the current directory",
	Long: `Mark the current directory as a Flotio project by creating a .flotio.yaml file.

If a project ID is given, it is used directly.
If no ID is given and --create is set, a new project is created via the API.
Otherwise, the command is interactive and asks for the project ID.`,
	Example: `  flotio init 42
  flotio init --create "My App" --repo https://github.com/user/repo
  flotio init`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("getting current directory: %w", err)
		}

		// Check if already initialized
		if pc, dir := config.FindProject(cwd); pc != nil {
			return fmt.Errorf("already a Flotio project (ID %d in %s/.flotio.yaml)", pc.ProjectID, dir)
		}

		var projectID int64

		if len(args) > 0 {
			// Explicit project ID
			id, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid project ID: %s", args[0])
			}
			projectID = id
		} else if name, _ := cmd.Flags().GetString("create"); name != "" {
			// Create project via API
			if !client.IsLoggedIn() {
				return fmt.Errorf("not logged in — run 'flotio login' first")
			}
			repo, _ := cmd.Flags().GetString("repo")
			body := map[string]interface{}{"name": name}
			if repo != "" {
				body["config"] = map[string]interface{}{"git_repo": repo}
			}
			var result map[string]interface{}
			if err := client.PostJSON(cfg.ResolveHost(), "/project", body, &result); err != nil {
				return fmt.Errorf("creating project: %w", err)
			}
			pid, ok := result["id"].(float64)
			if !ok {
				return fmt.Errorf("unexpected response from project creation")
			}
			projectID = int64(pid)
			fmt.Printf("✓ Project created: [%d] %s\n", projectID, name)
		} else {
			// Interactive — ask for project ID
			fmt.Print("Enter project ID: ")
			var input string
			fmt.Scanln(&input)
			id, err := strconv.ParseInt(input, 10, 64)
			if err != nil || id <= 0 {
				return fmt.Errorf("invalid project ID: %s", input)
			}
			projectID = id
		}

		pc := &config.ProjectConfig{ProjectID: projectID}
		if err := config.SaveProject(cwd, pc); err != nil {
			return err
		}
		fmt.Printf("✓ Initialized Flotio project [%d] in %s/\n", projectID, cwd)
		fmt.Println("  Created .flotio.yaml")
		return nil
	},
}

func init() {
	initCmd.Flags().String("create", "", "Create a new project with this name instead of using an existing ID")
	initCmd.Flags().String("repo", "", "Git repository URL (used with --create)")
	rootCmd.AddCommand(initCmd)
}

// parseProjectID resolves the project ID from a flag value or .flotio.yaml.
func parseProjectID(flagValue string) (int64, error) {
	if flagValue != "" {
		return strconv.ParseInt(flagValue, 10, 64)
	}
	return config.ResolveProjectID(0)
}

// projectIDArg extracts the first arg or resolves from .flotio.yaml.
func projectIDArg(args []string) (string, error) {
	if len(args) > 0 {
		return args[0], nil
	}
	id, err := config.ResolveProjectID(0)
	if err != nil {
		return "", err
	}
	return strconv.FormatInt(id, 10), nil
}

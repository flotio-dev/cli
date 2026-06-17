package cmd

import (
	"fmt"

	"github.com/flotio-dev/cli/pkg/client"
	"github.com/spf13/cobra"
)

var projectConfigCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage project configuration",
	Long:  `View and update project-level configuration (Flutter version, platforms, build mode, etc.).`,
	Example: `  flotio config get
  flotio config get 1
  flotio config update --flutter 3.27.4 --mode release --platform android
  flotio config delete`,
}

var configGetCmd = &cobra.Command{
	Use:     "get [project-id]",
	Short:   "Get project configuration",
	Args:    cobra.MaximumNArgs(1),
	Example: `  flotio config get`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if !client.IsLoggedIn() {
			return fmt.Errorf("not logged in")
		}
		id, err := projectIDArg(args)
		if err != nil {
			return err
		}
		var result map[string]interface{}
		if err := client.GetJSON(cfg.ResolveHost(), "/project/"+id+"/config", &result); err != nil {
			return fmt.Errorf("getting config: %w", err)
		}
		fmt.Printf("Git Repo:       %v\n", result["git_repo"])
		fmt.Printf("Flutter:        %v\n", result["flutter_version"])
		fmt.Printf("Platforms:      %v\n", result["platforms"])
		fmt.Printf("Build Mode:     %v\n", result["build_mode"])
		fmt.Printf("Build Trigger:  %v\n", result["build_trigger"])
		fmt.Printf("Caching:        %v\n", result["dependency_caching"])
		fmt.Printf("Tests:          %v\n", result["test"])
		return nil
	},
}

var configUpdateCmd = &cobra.Command{
	Use:     "update [project-id]",
	Short:   "Update project configuration",
	Args:    cobra.MaximumNArgs(1),
	Example: `  flotio config update --flutter 3.27.4 --mode release`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if !client.IsLoggedIn() {
			return fmt.Errorf("not logged in")
		}
		id, err := projectIDArg(args)
		if err != nil {
			return err
		}
		body := map[string]interface{}{}
		if f := flagStr(cmd, "flutter"); f != "" {
			body["flutter_version"] = f
		}
		if f := flagStr(cmd, "mode"); f != "" {
			body["build_mode"] = f
		}
		if f := flagStr(cmd, "repo"); f != "" {
			body["git_repo"] = f
		}
		platforms, _ := cmd.Flags().GetStringSlice("platform")
		if len(platforms) > 0 {
			body["platforms"] = platforms
		}

		if err := client.PostJSON(cfg.ResolveHost(), "/project/"+id+"/config", body, nil); err != nil {
			return fmt.Errorf("updating config: %w", err)
		}
		fmt.Printf("✓ Config for project %s updated\n", id)
		return nil
	},
}

var configDeleteCmd = &cobra.Command{
	Use:     "delete [project-id]",
	Short:   "Delete project configuration",
	Args:    cobra.MaximumNArgs(1),
	Example: `  flotio config delete`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if !client.IsLoggedIn() {
			return fmt.Errorf("not logged in")
		}
		id, err := projectIDArg(args)
		if err != nil {
			return err
		}
		if err := client.DeleteJSON(cfg.ResolveHost(), "/project/"+id+"/config"); err != nil {
			return fmt.Errorf("deleting config: %w", err)
		}
		fmt.Printf("✓ Config for project %s deleted\n", id)
		return nil
	},
}

func flagStr(cmd *cobra.Command, name string) string {
	v, _ := cmd.Flags().GetString(name)
	return v
}

func init() {
	configUpdateCmd.Flags().String("flutter", "", "Flutter version")
	configUpdateCmd.Flags().String("mode", "", "Build mode (debug, release, profile)")
	configUpdateCmd.Flags().String("repo", "", "Git repository URL")
	configUpdateCmd.Flags().StringSlice("platform", nil, "Platforms (android, ios, web)")

	projectConfigCmd.AddCommand(configGetCmd)
	projectConfigCmd.AddCommand(configUpdateCmd)
	projectConfigCmd.AddCommand(configDeleteCmd)
	rootCmd.AddCommand(projectConfigCmd)
}

package cmd

import (
	"fmt"
	"strings"

	"github.com/flotio-dev/cli/pkg/api/client/projects"
	"github.com/flotio-dev/cli/pkg/client"
	"github.com/spf13/cobra"
)

var projectCmd = &cobra.Command{
	Use:   "project",
	Short: "Manage projects",
	Long:  `Create, list, update, and delete Flotio projects.`,
	Example: `  flotio project list
  flotio project get
  flotio project create "My App" --repo https://github.com/user/repo --platform android
  flotio project update --name "New Name"
  flotio project delete`,
}

var projectListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all projects",
	RunE: func(cmd *cobra.Command, args []string) error {
		if !client.IsLoggedIn() {
			return fmt.Errorf("not logged in — run 'flotio login' first")
		}
		params := projects.NewGetProjectParams()
		resp, err := api.Projects.GetProject(params)
		if err != nil {
			return fmt.Errorf("listing projects: %w", err)
		}
		list := resp.GetPayload()
		if list == nil || list.Projects == nil {
			fmt.Println("No projects found.")
			return nil
		}
		for _, p := range list.Projects {
			fmt.Printf("  [%d] %s\n", p.ID, p.Name)
		}
		return nil
	},
}

var projectGetCmd = &cobra.Command{
	Use:     "get [id]",
	Short:   "Get a project by ID",
	Args:    cobra.MaximumNArgs(1),
	Example: `  flotio project get`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if !client.IsLoggedIn() {
			return fmt.Errorf("not logged in")
		}
		arg := ""
		if len(args) > 0 {
			arg = args[0]
		}
		id, err := parseProjectID(arg)
		if err != nil {
			return err
		}
		params := projects.NewGetProjectIDParams().WithID(id)
		resp, err := api.Projects.GetProjectID(params)
		if err != nil {
			return fmt.Errorf("getting project: %w", err)
		}
		p := resp.GetPayload().Project
		fmt.Printf("ID:        %d\n", p.ID)
		fmt.Printf("Name:      %s\n", p.Name)
		fmt.Printf("User ID:   %d\n", p.UserID)
		fmt.Printf("Created:   %s\n", p.CreatedAt)
		fmt.Printf("Updated:   %s\n", p.UpdatedAt)
		return nil
	},
}

var projectCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new project",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if !client.IsLoggedIn() {
			return fmt.Errorf("not logged in")
		}
		name := args[0]
		gitRepo, _ := cmd.Flags().GetString("repo")
		platforms, _ := cmd.Flags().GetStringSlice("platform")

		body := &ProjectCreateReq{Name: name}
		if gitRepo != "" || len(platforms) > 0 {
			cfg := &ProjectConfig{GitRepo: gitRepo}
			if len(platforms) > 0 {
				cfg.Platforms = flattenPlatforms(platforms)
			}
			body.Config = cfg
		}

		params := projects.NewPostProjectParams().WithProject(body)
		resp, err := api.Projects.PostProject(params)
		if err != nil {
			return fmt.Errorf("creating project: %w", err)
		}
		p := resp.GetPayload().Project
		fmt.Printf("✓ Project created: [%d] %s\n", p.ID, p.Name)
		return nil
	},
}

var projectUpdateCmd = &cobra.Command{
	Use:     "update [id]",
	Short:   "Update a project",
	Args:    cobra.MaximumNArgs(1),
	Example: `  flotio project update --name "New Name"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if !client.IsLoggedIn() {
			return fmt.Errorf("not logged in")
		}
		arg := ""
		if len(args) > 0 {
			arg = args[0]
		}
		id, err := parseProjectID(arg)
		if err != nil {
			return err
		}
		name, _ := cmd.Flags().GetString("name")
		if name == "" {
			return fmt.Errorf("--name is required for update")
		}
		gitRepo, _ := cmd.Flags().GetString("repo")

		body := &ProjectUpdateReq{Name: name}
		if gitRepo != "" {
			body.Config = &ProjectConfig{GitRepo: gitRepo}
		}

		params := projects.NewPutProjectIDParams().WithID(id).WithProject(body)
		_, err = api.Projects.PutProjectID(params)
		if err != nil {
			return fmt.Errorf("updating project: %w", err)
		}
		fmt.Printf("✓ Project %d updated\n", id)
		return nil
	},
}

var projectDeleteCmd = &cobra.Command{
	Use:     "delete [id]",
	Short:   "Delete a project",
	Args:    cobra.MaximumNArgs(1),
	Example: `  flotio project delete`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if !client.IsLoggedIn() {
			return fmt.Errorf("not logged in")
		}
		arg := ""
		if len(args) > 0 {
			arg = args[0]
		}
		id, err := parseProjectID(arg)
		if err != nil {
			return err
		}
		params := projects.NewDeleteProjectIDParams().WithID(id)
		_, err = api.Projects.DeleteProjectID(params)
		if err != nil {
			return fmt.Errorf("deleting project: %w", err)
		}
		fmt.Printf("✓ Project %d deleted\n", id)
		return nil
	},
}

func flattenPlatforms(raw []string) []string {
	var out []string
	for _, p := range raw {
		for _, s := range strings.Split(p, ",") {
			s = strings.TrimSpace(s)
			if s != "" {
				out = append(out, s)
			}
		}
	}
	return out
}

func init() {
	projectCreateCmd.Flags().String("repo", "", "Git repository URL")
	projectCreateCmd.Flags().StringSlice("platform", nil, "Platforms (android, ios, web)")
	projectUpdateCmd.Flags().String("name", "", "New project name")
	projectUpdateCmd.Flags().String("repo", "", "Git repository URL")

	projectCmd.AddCommand(projectListCmd)
	projectCmd.AddCommand(projectGetCmd)
	projectCmd.AddCommand(projectCreateCmd)
	projectCmd.AddCommand(projectUpdateCmd)
	projectCmd.AddCommand(projectDeleteCmd)
	rootCmd.AddCommand(projectCmd)
}

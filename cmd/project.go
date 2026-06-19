package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/flotio-dev/cli/pkg/api/client/projects"
	"github.com/flotio-dev/cli/pkg/client"
	"github.com/flotio-dev/cli/pkg/display"
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
		if list == nil || list.Projects == nil || len(list.Projects) == 0 {
			display.NoResults("projects")
			return nil
		}
		table := &display.Table{
			Columns: []display.Column{
				{Header: "ID", Width: 5, Align: 1},
				{Header: "Name", Max: 35},
				{Header: "Platforms", Max: 25},
			},
		}
		for _, p := range list.Projects {
			platforms := "-"
			if p.Config != nil && len(p.Config.Platforms) > 0 {
				platforms = strings.Join(p.Config.Platforms, ", ")
			}
			table.AddRow(
				fmt.Sprintf("%d", p.ID),
				p.Name,
				platforms,
			)
		}
		table.Render()
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
		display.HeadingPrint("Project %d", p.ID)
		display.KeyValue("Name", "%s", p.Name)
		display.KeyValue("User ID", "%d", p.UserID)
		display.KeyValue("Created", "%s", p.CreatedAt)
		display.KeyValue("Updated", "%s", p.UpdatedAt)
		if p.Config != nil {
			if p.Config.GitRepo != "" {
				display.KeyValue("Git Repo", "%s", p.Config.GitRepo)
			}
			if len(p.Config.Platforms) > 0 {
				display.KeyValue("Platforms", "%s", strings.Join(p.Config.Platforms, ", "))
			}
		}
		return nil
	},
}

var projectCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new project",
	Args:  cobra.ExactArgs(1),
	Example: `  flotio project create "My App" --repo https://github.com/user/repo --platform android
  flotio project create "My App" --repo <url> --flutter-version 3.22.0 --build-mode release \
    --android-format aab --play-track production`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if !client.IsLoggedIn() {
			return fmt.Errorf("not logged in")
		}
		body := &ProjectCreateReq{Name: args[0]}
		cfg, err := configFromFlags(cmd)
		if err != nil {
			return err
		}
		if cfg != nil {
			body.Config = cfg
		}

		params := projects.NewPostProjectParams().WithProject(body)
		resp, err := api.Projects.PostProject(params)
		if err != nil {
			return fmt.Errorf("creating project: %w", err)
		}
		p := resp.GetPayload().Project
		display.SuccessPrint("Project created: [%d] %s", p.ID, p.Name)
		return nil
	},
}

var projectUpdateCmd = &cobra.Command{
	Use:     "update [id]",
	Short:   "Update a project (name and/or config flags)",
	Args:    cobra.MaximumNArgs(1),
	Example: `  flotio project update 3 --name "New Name"
  flotio project update 3 --flutter-version 3.22.0 --build-mode release
  flotio project update --repo https://github.com/user/repo   # uses project-id from config`,
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
		cfg, err := configFromFlags(cmd)
		if err != nil {
			return err
		}
		if name == "" && cfg == nil {
			return fmt.Errorf("nothing to update — pass --name or at least one config flag (--repo, --flutter-version, ...)")
		}

		body := &ProjectUpdateReq{Name: name}
		if cfg != nil {
			body.Config = cfg
		}

		params := projects.NewPutProjectIDParams().WithID(id).WithProject(body)
		_, err = api.Projects.PutProjectID(params)
		if err != nil {
			return fmt.Errorf("updating project: %w", err)
		}
		display.SuccessPrint("Project %d updated", id)
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
		display.SuccessPrint("Project %d deleted", id)
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

// addProjectConfigFlags registers the shared project-config flags on a command.
// Used by both `project create` and `project update` so they expose the same
// surface and stay in sync.
func addProjectConfigFlags(cmd *cobra.Command) {
	cmd.Flags().String("repo", "", "Git repository URL")
	cmd.Flags().String("git-username", "", "Git username for private repos")
	cmd.Flags().String("git-token", "", "Git token/password for private repos (use @file to read from a file)")
	cmd.Flags().StringSlice("platform", nil, "Target platforms (android, ios, web)")
	cmd.Flags().String("flutter-version", "", "Flutter SDK version (e.g. 3.22.0)")
	cmd.Flags().String("build-mode", "", "Build mode: debug, release, or profile")
	cmd.Flags().String("project-path", "", "Path to the Flutter project inside the repo")
	cmd.Flags().String("android-format", "", "Android build format: apk or aab")
	cmd.Flags().String("play-track", "", "Google Play track: internal, alpha, beta, or production")
}

// resolveSecret handles the @file syntax for secret flags (e.g. --git-token @./pat).
// A leading "@" reads the file content (trimmed of surrounding whitespace).
// Otherwise the value is returned as-is.
func resolveSecret(v string) (string, error) {
	if !strings.HasPrefix(v, "@") {
		return v, nil
	}
	data, err := os.ReadFile(strings.TrimPrefix(v, "@"))
	if err != nil {
		return "", fmt.Errorf("reading secret file: %w", err)
	}
	return strings.TrimSpace(string(data)), nil
}

// configFromFlags builds a *ProjectConfig from the shared config flags.
// Returns nil when no config flag was set, so callers can omit the config block.
func configFromFlags(cmd *cobra.Command) (*ProjectConfig, error) {
	var cfg ProjectConfig
	any := false

	if v, _ := cmd.Flags().GetString("repo"); v != "" {
		cfg.GitRepo = v
		any = true
	}
	if v, _ := cmd.Flags().GetString("git-username"); v != "" {
		cfg.GitUsername = v
		any = true
	}
	if v, _ := cmd.Flags().GetString("git-token"); v != "" {
		secret, err := resolveSecret(v)
		if err != nil {
			return nil, err
		}
		cfg.GitToken = secret
		any = true
	}
	if v, _ := cmd.Flags().GetStringSlice("platform"); len(v) > 0 {
		cfg.Platforms = flattenPlatforms(v)
		any = true
	}
	if v, _ := cmd.Flags().GetString("flutter-version"); v != "" {
		cfg.FlutterVersion = v
		any = true
	}
	if v, _ := cmd.Flags().GetString("build-mode"); v != "" {
		cfg.BuildMode = v
		any = true
	}
	if v, _ := cmd.Flags().GetString("project-path"); v != "" {
		cfg.ProjectPath = v
		any = true
	}
	if v, _ := cmd.Flags().GetString("android-format"); v != "" {
		cfg.AndroidBuildFormat = v
		any = true
	}
	if v, _ := cmd.Flags().GetString("play-track"); v != "" {
		cfg.GooglePlayTrack = v
		any = true
	}

	if !any {
		return nil, nil
	}
	return &cfg, nil
}

func init() {
	addProjectConfigFlags(projectCreateCmd)
	projectUpdateCmd.Flags().String("name", "", "New project name")
	addProjectConfigFlags(projectUpdateCmd)

	projectCmd.AddCommand(projectListCmd)
	projectCmd.AddCommand(projectGetCmd)
	projectCmd.AddCommand(projectCreateCmd)
	projectCmd.AddCommand(projectUpdateCmd)
	projectCmd.AddCommand(projectDeleteCmd)
	rootCmd.AddCommand(projectCmd)
}

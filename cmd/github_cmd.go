package cmd

import (
	"fmt"

	"github.com/flotio-dev/cli/pkg/api/client/github"
	"github.com/flotio-dev/cli/pkg/client"
	"github.com/spf13/cobra"
)

var githubCmd = &cobra.Command{
	Use:   "github",
	Short: "Manage GitHub integration",
	Long:  `Connect your GitHub account, browse repositories, and manage installations.`,
	Example: `  flotio github status
  flotio github repos
  flotio github repo owner/repo
  flotio github connect 12345678
  flotio github disconnect`,
}

var githubReposCmd = &cobra.Command{
	Use:   "repos",
	Short: "List accessible GitHub repositories",
	RunE: func(cmd *cobra.Command, args []string) error {
		if !client.IsLoggedIn() {
			return fmt.Errorf("not logged in")
		}
		params := github.NewGetGithubReposParams()
		resp, err := api.Github.GetGithubRepos(params, nil)
		if err != nil {
			return fmt.Errorf("listing repos: %w", err)
		}
		list := resp.GetPayload()
		for _, r := range list.Repositories {
			visibility := "public"
			if r.Private {
				visibility = "private"
			}
			fmt.Printf("  %s (%s)\n", r.FullName, visibility)
		}
		return nil
	},
}

var githubRepoCmd = &cobra.Command{
	Use:   "repo <owner/repo>",
	Short: "Show a repository tree",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if !client.IsLoggedIn() {
			return fmt.Errorf("not logged in")
		}
		params := github.NewGetGithubRepoParams().WithRepo(args[0])
		resp, err := api.Github.GetGithubRepo(params, nil)
		if err != nil {
			return fmt.Errorf("getting repo: %w", err)
		}
		tree := resp.GetPayload()
		for _, item := range tree.Tree {
			icon := "📁"
			if item.Type != "dir" {
				icon = "📄"
			}
			fmt.Printf("  %s %s (%s)\n", icon, item.Name, item.Path)
		}
		return nil
	},
}

var githubConnectCmd = &cobra.Command{
	Use:   "connect <installation-id>",
	Short: "Connect a GitHub App installation",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if !client.IsLoggedIn() {
			return fmt.Errorf("not logged in")
		}
		var instID int64
		fmt.Sscanf(args[0], "%d", &instID)

		body := &GHPostInstallReq{InstallationID: instID}
		params := github.NewPostGithubPostInstallationParams().WithPayload(body)
		_, err := api.Github.PostGithubPostInstallation(params, nil)
		if err != nil {
			return fmt.Errorf("connecting GitHub: %w", err)
		}
		fmt.Printf("✓ GitHub installation %d connected\n", instID)
		return nil
	},
}

var githubDisconnectCmd = &cobra.Command{
	Use:     "disconnect",
	Short:   "Disconnect GitHub integration",
	Example: `  flotio github disconnect`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if !client.IsLoggedIn() {
			return fmt.Errorf("not logged in")
		}
		if err := client.DeleteJSON(cfg.ResolveHost(), "/github/disconnect"); err != nil {
			return fmt.Errorf("disconnecting GitHub: %w", err)
		}
		fmt.Println("✓ GitHub disconnected")
		return nil
	},
}

var githubStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check GitHub installation status",
	RunE: func(cmd *cobra.Command, args []string) error {
		if !client.IsLoggedIn() {
			return fmt.Errorf("not logged in")
		}
		params := github.NewGetGithubInstallationsParams()
		resp, err := api.Github.GetGithubInstallations(params, nil)
		if err != nil {
			return fmt.Errorf("checking GitHub status: %w", err)
		}
		inst := resp.GetPayload()
		if inst.ID == 0 {
			fmt.Println("No GitHub installation connected.")
			return nil
		}
		fmt.Printf("Installation ID: %d\n", inst.ID)
		fmt.Printf("Account:         %s (%s)\n", inst.AccountLogin, inst.AccountType)
		return nil
	},
}

func init() {
	githubCmd.AddCommand(githubReposCmd)
	githubCmd.AddCommand(githubRepoCmd)
	githubCmd.AddCommand(githubConnectCmd)
	githubCmd.AddCommand(githubDisconnectCmd)
	githubCmd.AddCommand(githubStatusCmd)
	rootCmd.AddCommand(githubCmd)
}

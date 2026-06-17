package cmd

import (
	"fmt"

	"github.com/flotio-dev/cli/pkg/api/client/github"
	"github.com/flotio-dev/cli/pkg/client"
	"github.com/flotio-dev/cli/pkg/display"
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
		if list == nil || len(list.Repositories) == 0 {
			display.NoResults("repositories")
			return nil
		}
		table := &display.Table{
			Columns: []display.Column{
				{Header: "Repository", Max: 45},
				{Header: "Visibility", Width: 12},
			},
		}
		for _, r := range list.Repositories {
			vis := "public"
			if r.Private {
				vis = display.Yellow + "private" + display.Reset
			}
			table.AddRow(r.FullName, vis)
		}
		table.Render()
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
		if tree == nil || len(tree.Tree) == 0 {
			display.NoResults("files")
			return nil
		}
		table := &display.Table{
			Columns: []display.Column{
				{Header: "Name", Max: 30},
				{Header: "Path", Max: 40},
				{Header: "Type", Width: 6},
			},
		}
		for _, item := range tree.Tree {
			icon := "📄"
			if item.Type == "dir" {
				icon = "📁"
			}
			table.AddRow(icon+" "+item.Name, item.Path, item.Type)
		}
		table.Render()
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
		display.SuccessPrint("GitHub installation %d connected", instID)
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
		display.SuccessPrint("GitHub disconnected")
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
			display.NoResults("GitHub installation")
			return nil
		}
		display.HeadingPrint("GitHub Installation")
		display.KeyValue("Installation ID", "%d", inst.ID)
		display.KeyValue("Account", "%s (%s)", inst.AccountLogin, inst.AccountType)
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

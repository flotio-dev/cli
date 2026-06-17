package cmd

import (
	"fmt"
	"strconv"

	"github.com/flotio-dev/cli/pkg/api/client/builds"
	"github.com/flotio-dev/cli/pkg/client"
	"github.com/spf13/cobra"
)

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Manage builds",
	Long:  `Trigger, monitor, and manage Flotio builds for your projects.`,
	Example: `  flotio build start 1 --branch main --platform android --mode release
  flotio build list 1
  flotio build logs 1 42
  flotio build download 1 42
  flotio build cancel 1 42
  flotio build delete 1 42`,
}

var buildStartCmd = &cobra.Command{
	Use:   "start <project-id>",
	Short: "Trigger a new build",
	Example: `  flotio build start 1 --branch main --platform android --mode release --target aab`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if !client.IsLoggedIn() {
			return fmt.Errorf("not logged in")
		}
		projectID, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid project ID: %s", args[0])
		}
		branch, _ := cmd.Flags().GetString("branch")
		platform, _ := cmd.Flags().GetString("platform")
		mode, _ := cmd.Flags().GetString("mode")
		target, _ := cmd.Flags().GetString("target")

		body := &BuildRequest{
			GitBranch:    branch,
			Platform:     platform,
			BuildMode:    mode,
			BuildTarget:  target,
		}

		params := builds.NewPostProjectIDBuildParams().WithID(projectID).WithBuild(body)
		resp, err := api.Builds.PostProjectIDBuild(params)
		if err != nil {
			return fmt.Errorf("starting build: %w", err)
		}
		b := resp.GetPayload().Build
		fmt.Printf("✓ Build started: [%d] %s\n", b.ID, b.Status)
		return nil
	},
}

var buildListCmd = &cobra.Command{
	Use:   "list <project-id>",
	Short: "List builds for a project",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if !client.IsLoggedIn() {
			return fmt.Errorf("not logged in")
		}
		projectID, _ := strconv.ParseInt(args[0], 10, 64)

		params := builds.NewGetProjectIDBuildsParams().WithID(projectID)
		resp, err := api.Builds.GetProjectIDBuilds(params)
		if err != nil {
			return fmt.Errorf("listing builds: %w", err)
		}
		list := resp.GetPayload()
		if list == nil || list.Builds == nil {
			fmt.Println("No builds found.")
			return nil
		}
		for _, b := range list.Builds {
			fmt.Printf("  [%d] %s | %s | %s\n", b.ID, b.Status, b.Platform, b.GitBranch)
		}
		return nil
	},
}

var buildLogsCmd = &cobra.Command{
	Use:     "logs <project-id> <build-id>",
	Short:   "Get build logs",
	Example: `  flotio build logs 1 42`,
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		if !client.IsLoggedIn() {
			return fmt.Errorf("not logged in")
		}
		projectID, _ := strconv.ParseInt(args[0], 10, 64)
		buildID, _ := strconv.ParseInt(args[1], 10, 64)

		params := builds.NewGetProjectIDBuildBuildIDLogsSyncParams().
			WithID(projectID).WithBuildID(buildID)
		resp, err := api.Builds.GetProjectIDBuildBuildIDLogsSync(params)
		if err != nil {
			return fmt.Errorf("getting logs: %w", err)
		}
		payload := resp.GetPayload()
		for _, line := range payload.Logs {
			fmt.Println(line)
		}
		if payload.HasMore {
			fmt.Println("(more logs available)")
		}
		return nil
	},
}

var buildCancelCmd = &cobra.Command{
	Use:   "cancel <project-id> <build-id>",
	Short: "Cancel a running build",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		if !client.IsLoggedIn() {
			return fmt.Errorf("not logged in")
		}
		projectID, _ := strconv.ParseInt(args[0], 10, 64)
		buildID, _ := strconv.ParseInt(args[1], 10, 64)

		params := builds.NewPutProjectIDBuildBuildIDCancelParams().
			WithID(projectID).WithBuildID(buildID)
		_, err := api.Builds.PutProjectIDBuildBuildIDCancel(params)
		if err != nil {
			return fmt.Errorf("cancelling build: %w", err)
		}
		fmt.Printf("✓ Build %d cancelled\n", buildID)
		return nil
	},
}

var buildDownloadCmd = &cobra.Command{
	Use:   "download <project-id> <build-id>",
	Short: "Get download URL for a build artifact",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		if !client.IsLoggedIn() {
			return fmt.Errorf("not logged in")
		}
		projectID, _ := strconv.ParseInt(args[0], 10, 64)
		buildID, _ := strconv.ParseInt(args[1], 10, 64)

		params := builds.NewGetProjectIDBuildBuildIDDownloadParams().
			WithID(projectID).WithBuildID(buildID)
		resp, err := api.Builds.GetProjectIDBuildBuildIDDownload(params)
		if err != nil {
			return fmt.Errorf("getting download URL: %w", err)
		}
		payload := resp.GetPayload()
		fmt.Printf("Download URL: %s\n", payload.DownloadURL)
		fmt.Printf("Expires in:  %d seconds\n", payload.ExpiresIn)
		return nil
	},
}

var buildDeleteCmd = &cobra.Command{
	Use:   "delete <project-id> <build-id>",
	Short: "Delete a build and its artifacts",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		if !client.IsLoggedIn() {
			return fmt.Errorf("not logged in")
		}
		projectID, _ := strconv.ParseInt(args[0], 10, 64)
		buildID, _ := strconv.ParseInt(args[1], 10, 64)

		params := builds.NewDeleteProjectIDBuildBuildIDParams().
			WithID(projectID).WithBuildID(buildID)
		_, err := api.Builds.DeleteProjectIDBuildBuildID(params)
		if err != nil {
			return fmt.Errorf("deleting build: %w", err)
		}
		fmt.Printf("✓ Build %d deleted\n", buildID)
		return nil
	},
}

func init() {
	buildStartCmd.Flags().String("branch", "", "Git branch")
	buildStartCmd.Flags().String("platform", "", "Platform (android, ios, web)")
	buildStartCmd.Flags().String("mode", "", "Build mode (debug, release, profile)")
	buildStartCmd.Flags().String("target", "", "Build target (apk, aab)")

	buildCmd.AddCommand(buildStartCmd)
	buildCmd.AddCommand(buildListCmd)
	buildCmd.AddCommand(buildLogsCmd)
	buildCmd.AddCommand(buildCancelCmd)
	buildCmd.AddCommand(buildDownloadCmd)
	buildCmd.AddCommand(buildDeleteCmd)
	rootCmd.AddCommand(buildCmd)
}

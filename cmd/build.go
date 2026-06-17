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
	Example: `  flotio build start --branch main --platform android --mode release
  flotio build start 1 --branch main --platform android --mode release
  flotio build list
  flotio build logs 42
  flotio build download 42
  flotio build cancel 42
  flotio build delete 42`,
}

var buildStartCmd = &cobra.Command{
	Use:     "start [project-id]",
	Short:   "Trigger a new build",
	Example: `  flotio build start --branch main --platform android --mode release --target aab`,
	Args:    cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if !client.IsLoggedIn() {
			return fmt.Errorf("not logged in")
		}
		arg := ""
		if len(args) > 0 {
			arg = args[0]
		}
		projectID, err := parseProjectID(arg)
		if err != nil {
			return err
		}
		branch, _ := cmd.Flags().GetString("branch")
		platform, _ := cmd.Flags().GetString("platform")
		mode, _ := cmd.Flags().GetString("mode")
		target, _ := cmd.Flags().GetString("target")

		body := &BuildRequest{
			GitBranch:   branch,
			Platform:    platform,
			BuildMode:   mode,
			BuildTarget: target,
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
	Use:     "list [project-id]",
	Short:   "List builds for a project",
	Args:    cobra.MaximumNArgs(1),
	Example: `  flotio build list`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if !client.IsLoggedIn() {
			return fmt.Errorf("not logged in")
		}
		arg := ""
		if len(args) > 0 {
			arg = args[0]
		}
		projectID, err := parseProjectID(arg)
		if err != nil {
			return err
		}

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
	Use:     "logs [project-id] <build-id>",
	Short:   "Get build logs",
	Example: `  flotio build logs 42`,
	Args:    cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		if !client.IsLoggedIn() {
			return fmt.Errorf("not logged in")
		}
		projectArg := ""
		buildArg := args[0]
		if len(args) == 2 {
			projectArg = args[0]
			buildArg = args[1]
		}
		projectID, err := parseProjectID(projectArg)
		if err != nil {
			return err
		}
		buildID, err := strconv.ParseInt(buildArg, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid build ID: %s", buildArg)
		}

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
	Use:     "cancel [project-id] <build-id>",
	Short:   "Cancel a running build",
	Args:    cobra.RangeArgs(1, 2),
	Example: `  flotio build cancel 42`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if !client.IsLoggedIn() {
			return fmt.Errorf("not logged in")
		}
		projectArg := ""
		buildArg := args[0]
		if len(args) == 2 {
			projectArg = args[0]
			buildArg = args[1]
		}
		projectID, err := parseProjectID(projectArg)
		if err != nil {
			return err
		}
		buildID, err := strconv.ParseInt(buildArg, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid build ID: %s", buildArg)
		}

		params := builds.NewPutProjectIDBuildBuildIDCancelParams().
			WithID(projectID).WithBuildID(buildID)
		_, err = api.Builds.PutProjectIDBuildBuildIDCancel(params)
		if err != nil {
			return fmt.Errorf("cancelling build: %w", err)
		}
		fmt.Printf("✓ Build %d cancelled\n", buildID)
		return nil
	},
}

var buildDownloadCmd = &cobra.Command{
	Use:     "download [project-id] <build-id>",
	Short:   "Get download URL for a build artifact",
	Args:    cobra.RangeArgs(1, 2),
	Example: `  flotio build download 42`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if !client.IsLoggedIn() {
			return fmt.Errorf("not logged in")
		}
		projectArg := ""
		buildArg := args[0]
		if len(args) == 2 {
			projectArg = args[0]
			buildArg = args[1]
		}
		projectID, err := parseProjectID(projectArg)
		if err != nil {
			return err
		}
		buildID, err := strconv.ParseInt(buildArg, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid build ID: %s", buildArg)
		}

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
	Use:     "delete [project-id] <build-id>",
	Short:   "Delete a build and its artifacts",
	Args:    cobra.RangeArgs(1, 2),
	Example: `  flotio build delete 42`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if !client.IsLoggedIn() {
			return fmt.Errorf("not logged in")
		}
		projectArg := ""
		buildArg := args[0]
		if len(args) == 2 {
			projectArg = args[0]
			buildArg = args[1]
		}
		projectID, err := parseProjectID(projectArg)
		if err != nil {
			return err
		}
		buildID, err := strconv.ParseInt(buildArg, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid build ID: %s", buildArg)
		}

		params := builds.NewDeleteProjectIDBuildBuildIDParams().
			WithID(projectID).WithBuildID(buildID)
		_, err = api.Builds.DeleteProjectIDBuildBuildID(params)
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

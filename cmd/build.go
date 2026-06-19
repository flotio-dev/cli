package cmd

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/flotio-dev/cli/pkg/api/client/builds"
	"github.com/flotio-dev/cli/pkg/client"
	"github.com/flotio-dev/cli/pkg/display"
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
		display.SuccessPrint("Build started: [%d] %s | %s | %s", b.ID, display.StatusBadge(b.Status), b.Platform, b.GitBranch)
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
		if display.JSONOutput() {
			if list != nil {
				display.PrintJSON(list.Builds)
			} else {
				fmt.Println("[]")
			}
			return nil
		}
		if list == nil || list.Builds == nil || len(list.Builds) == 0 {
			display.NoResults("builds")
			return nil
		}
		table := &display.Table{
			Columns: []display.Column{
				{Header: "ID", Width: 5, Align: 1},
				{Header: "Status", Width: 12},
				{Header: "Platform", Width: 10},
				{Header: "Branch", Max: 20},
				{Header: "Duration", Width: 10},
			},
		}
		for _, b := range list.Builds {
			dur := "-"
			if b.Duration > 0 {
				dur = fmt.Sprintf("%ds", b.Duration)
			}
			table.AddRow(
				fmt.Sprintf("%d", b.ID),
				display.StatusBadge(b.Status),
				b.Platform,
				b.GitBranch,
				dur,
			)
		}
		table.Render()
		return nil
	},
}

var buildLogsCmd = &cobra.Command{
	Use:     "logs [project-id] <build-id>",
	Short:   "Get build logs",
	Example: `  flotio build logs 42
  flotio build logs --watch 42`,
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

		watch, _ := cmd.Flags().GetBool("watch")
		if watch {
			return runBuildLogsWatch(projectID, buildID)
		}

		// One-shot: use the non-sync logs endpoint.
		logParams := builds.NewGetProjectIDBuildBuildIDLogsParams().
			WithID(projectID).WithBuildID(buildID)
		logResp, err := api.Builds.GetProjectIDBuildBuildIDLogs(logParams)
		if err != nil {
			return fmt.Errorf("getting logs: %w", err)
		}
		payload := logResp.GetPayload()
		for _, line := range payload.Logs {
			fmt.Println(line)
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
		display.SuccessPrint("Build %d cancelled", buildID)
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
		display.HeadingPrint("Build %d Artifact", buildID)
		display.KeyValue("Download URL", "%s", payload.DownloadURL)
		display.KeyValue("Expires in", "%ds", payload.ExpiresIn)
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
		display.SuccessPrint("Build %d deleted", buildID)
		return nil
	},
}

func init() {
	buildStartCmd.Flags().String("branch", "", "Git branch")
	buildStartCmd.Flags().String("platform", "", "Platform (android, ios, web)")
	buildStartCmd.Flags().String("mode", "", "Build mode (debug, release, profile)")
	buildStartCmd.Flags().String("target", "", "Build target (apk, aab)")

	buildLogsCmd.Flags().BoolP("watch", "w", false, "Watch logs in real-time (HTTP polling)")

	buildCmd.AddCommand(buildStartCmd)
	buildCmd.AddCommand(buildListCmd)
	buildCmd.AddCommand(buildLogsCmd)
	buildCmd.AddCommand(buildCancelCmd)
	buildCmd.AddCommand(buildDownloadCmd)
	buildCmd.AddCommand(buildDeleteCmd)
	rootCmd.AddCommand(buildCmd)
}

// runBuildLogsWatch polls the logs/sync endpoint continuously,
// printing new lines as they arrive until the build finishes.
func runBuildLogsWatch(projectID, buildID int64) error {
	connectionID := fmt.Sprintf("flotio-cli-%d", time.Now().UnixNano())
	lastLine := int64(0)
	pollInterval := 2 * time.Second
	pollTimeout := 15 * time.Second

	fmt.Printf("%sWatching logs for build %d (project %d)...%s\n", display.Bold, buildID, projectID, display.Reset)
	fmt.Printf("  %sPress Ctrl+C to stop.%s\n\n", display.Muted, display.Reset)

	for {
		ll := lastLine
		params := builds.NewGetProjectIDBuildBuildIDLogsSyncParams().
			WithID(projectID).
			WithBuildID(buildID).
			WithConnectionID(connectionID).
			WithLastLine(&ll).
			WithTimeout(pollTimeout)

		resp, err := api.Builds.GetProjectIDBuildBuildIDLogsSync(params)
		if err != nil {
			errStr := err.Error()
			// 404 typically means the build no longer exists (finished/cleaned up).
			if strings.Contains(errStr, "404") || strings.Contains(errStr, "not found") {
				fmt.Printf("\n%sBuild %d finished (no longer available).%s\n", display.Muted, buildID, display.Reset)
				return nil
			}
			if strings.Contains(errStr, "400") {
				// 400 may mean the build isn't in a running state.
				fmt.Printf("\n%sBuild %d is not running.%s\n", display.Muted, buildID, display.Reset)
				return nil
			}
			return fmt.Errorf("watching logs: %w", err)
		}

		payload := resp.GetPayload()

		// Print new log lines.
		for _, line := range payload.Logs {
			fmt.Println(line)
		}

		// Update cursor for next poll.
		if payload.LastLine > lastLine {
			lastLine = payload.LastLine
		}

		// Check if the build has reached a terminal state.
		status := strings.ToLower(payload.Status)
		if !payload.HasMore && isTerminalStatus(status) {
			fmt.Printf("\n%s── Build %s ──%s\n", display.Bold, display.StatusBadge(status), display.Reset)
			if payload.ElapsedTime > 0 {
				fmt.Printf("  %sDuration: %ds%s\n", display.Muted, payload.ElapsedTime, display.Reset)
			}
			return nil
		}

		// Wait before next poll (API already did long-poll, but add a small
		// backoff to avoid hammering when no new logs are available).
		time.Sleep(pollInterval)
	}
}

func isTerminalStatus(status string) bool {
	switch status {
	case "succeeded", "success", "failed", "error", "cancelled", "canceled", "timeout":
		return true
	}
	return false
}

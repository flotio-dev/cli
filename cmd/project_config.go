package cmd

import (
	"fmt"

	"github.com/flotio-dev/cli/pkg/client"
	"github.com/flotio-dev/cli/pkg/display"
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
		var wrapper map[string]interface{}
		if err := client.GetJSON(cfg.ResolveHost(), "/project/"+id+"/config", &wrapper); err != nil {
			return fmt.Errorf("getting config: %w", err)
		}

		raw, ok := wrapper["config"]
		if !ok || raw == nil {
			display.NoResults("configuration")
			return nil
		}
		c, ok := raw.(map[string]interface{})
		if !ok || len(c) == 0 {
			display.NoResults("configuration")
			return nil
		}

		display.HeadingPrint("Project %s Configuration", id)

		// Git
		gitFields := mapFields(c, "git_repo", "git_username", "project_path")
		if hasAny(gitFields) {
			display.HeadingPrint("  Git")
			printField(c, "Repo", "git_repo")
			printField(c, "Username", "git_username")
			printField(c, "Token", "git_token")
			printField(c, "Project Path", "project_path")
			printField(c, "Watched Branches", "watched_branch_patterns")
			printField(c, "Watched Tags", "watched_tag_patterns")
		}

		// Build
		buildFields := mapFields(c, "flutter_version", "build_mode", "platforms", "android_build_format")
		if hasAny(buildFields) {
			display.HeadingPrint("  Build")
			printField(c, "Flutter Version", "flutter_version")
			printField(c, "Build Mode", "build_mode")
			printField(c, "Platforms", "platforms")
			printField(c, "Android Format", "android_build_format")
			printField(c, "Android Args", "android_build_args")
			printField(c, "iOS Args", "ios_build_args")
			printField(c, "Web Args", "web_build_args")
			printField(c, "Xcode Version", "xcode_version")
			printField(c, "CocoaPods", "cocoapods_version")
			printField(c, "Build Trigger", "build_trigger")
		}

		// Testing
		testFields := mapFields(c, "test", "enable_flutter_test", "enable_flutter_analyze")
		if hasAny(testFields) {
			display.HeadingPrint("  Testing")
			printField(c, "Enable Tests", "test")
			printField(c, "Flutter Test", "enable_flutter_test")
			printField(c, "Test Args", "flutter_test_args")
			printField(c, "Analyze", "enable_flutter_analyze")
			printField(c, "Analyze Args", "flutter_analyze_args")
			printField(c, "Driver", "enable_flutter_driver")
			printField(c, "Driver Args", "flutter_driver_args")
			printField(c, "Driver Targets", "flutter_driver_targets")
		}

		// Signing & Distribution
		signingFields := mapFields(c, "keystore_id", "enable_android_code_signing", "google_play_credentials_id", "enable_google_play_publishing")
		if hasAny(signingFields) {
			display.HeadingPrint("  Signing & Distribution")
			printField(c, "Keystore ID", "keystore_id")
			printField(c, "Code Signing", "enable_android_code_signing")
			printField(c, "Play Creds ID", "google_play_credentials_id")
			printField(c, "Play Track", "google_play_track")
			printField(c, "Play Publishing", "enable_google_play_publishing")
			printField(c, "Rollout Fraction", "rollout_fraction")
			printField(c, "Submit as Draft", "submit_as_draft")
			printField(c, "Skip Review", "do_not_send_for_review")
			printField(c, "Publish on Fail", "publish_even_if_tests_fail")
			printField(c, "Update Priority", "update_priority")
		}

		// Caching
		cacheFields := mapFields(c, "dependency_caching", "dependency_dirs")
		if hasAny(cacheFields) {
			display.HeadingPrint("  Caching")
			printField(c, "Dependency Caching", "dependency_caching")
			printField(c, "Dependency Dirs", "dependency_dirs")
		}

		// Scripts
		scriptFields := mapFields(c, "post_clone_script", "pre_build_script", "post_build_script", "pre_test_script", "post_test_script", "pre_publish_script")
		if hasAny(scriptFields) {
			display.HeadingPrint("  Scripts")
			printField(c, "Post Clone", "post_clone_script")
			printField(c, "Pre Build", "pre_build_script")
			printField(c, "Post Build", "post_build_script")
			printField(c, "Pre Test", "pre_test_script")
			printField(c, "Post Test", "post_test_script")
			printField(c, "Pre Publish", "pre_publish_script")
		}

		// Notifications
		notifFields := mapFields(c, "enable_email_notifications", "email_recipients", "webhook_urls")
		if hasAny(notifFields) {
			display.HeadingPrint("  Notifications")
			printField(c, "Email Notifs", "enable_email_notifications")
			printField(c, "Recipients", "email_recipients")
			printField(c, "Webhook URLs", "webhook_urls")
		}

		return nil
	},
}

func hasAny(fields map[string]interface{}) bool {
	for _, v := range fields {
		if v != nil && v != "" && v != false && v != float64(0) {
			return true
		}
	}
	return false
}

func mapFields(c map[string]interface{}, keys ...string) map[string]interface{} {
	out := make(map[string]interface{}, len(keys))
	for _, k := range keys {
		out[k] = c[k]
	}
	return out
}

func printField(c map[string]interface{}, label, key string) {
	v, ok := c[key]
	if !ok || v == nil || v == "" {
		return
	}
	if b, ok := v.(bool); ok && !b {
		return
	}
	if n, ok := v.(float64); ok && n == 0 {
		return
	}
	display.KeyValue("    "+label, "%v", v)
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
		display.SuccessPrint("Config for project %s updated", id)
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
		display.SuccessPrint("Config for project %s deleted", id)
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

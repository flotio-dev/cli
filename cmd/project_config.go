package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

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
	Use:   "update [project-id]",
	Short: "Update project configuration",
	Args:  cobra.MaximumNArgs(1),
	Example: `  flotio config update --flutter 3.27.4 --mode release
  flotio config update --tty`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if !client.IsLoggedIn() {
			return fmt.Errorf("not logged in")
		}
		id, err := projectIDArg(args)
		if err != nil {
			return err
		}

		tty, _ := cmd.Flags().GetBool("tty")
		if tty {
			return runConfigUpdateTTY(id)
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

// --- TTY interactive config update ---

type field struct {
	key    string
	label  string
	typ    string // "string", "bool", "int", "float", "slice"
	sensitive bool
}

func runConfigUpdateTTY(id string) error {
	host := cfg.ResolveHost()

	// Fetch current config
	var wrapper map[string]interface{}
	if err := client.GetJSON(host, "/project/"+id+"/config", &wrapper); err != nil {
		return fmt.Errorf("fetching current config: %w", err)
	}
	current := map[string]interface{}{}
	if raw, ok := wrapper["config"]; ok && raw != nil {
		if c, ok := raw.(map[string]interface{}); ok {
			current = c
		}
	}

	fmt.Printf("\n%sInteractive config update for project %s%s\n", display.Bold, id, display.Reset)
	fmt.Println("Press Enter to keep the current value, type '-' to clear a field.")
	fmt.Printf("Type %sq%s to quit at any section prompt.\n\n", display.Bold, display.Reset)

	r := bufio.NewReader(os.Stdin)
	body := map[string]interface{}{}

	sections := []struct {
		title  string
		fields []field
	}{
		{
			title: "Git",
			fields: []field{
				{"git_repo", "Repository URL", "string", false},
				{"git_username", "Username", "string", false},
				{"git_token", "Token", "string", true},
				{"project_path", "Project Path", "string", false},
			},
		},
		{
			title: "Build",
			fields: []field{
				{"flutter_version", "Flutter Version", "string", false},
				{"build_mode", "Build Mode", "string", false},
				{"platforms", "Platforms (comma-separated)", "slice", false},
				{"android_build_format", "Android Format (apk/aab)", "string", false},
				{"android_build_args", "Android Build Args", "string", false},
				{"ios_build_args", "iOS Build Args", "string", false},
				{"web_build_args", "Web Build Args", "string", false},
				{"xcode_version", "Xcode Version", "string", false},
				{"cocoapods_version", "CocoaPods Version", "string", false},
				{"build_trigger", "Build Trigger", "string", false},
				{"package_name", "Package Name", "string", false},
			},
		},
		{
			title: "Testing",
			fields: []field{
				{"test", "Enable Tests", "bool", false},
				{"enable_flutter_test", "Flutter Test", "bool", false},
				{"flutter_test_args", "Flutter Test Args", "string", false},
				{"enable_flutter_analyze", "Flutter Analyze", "bool", false},
				{"flutter_analyze_args", "Analyze Args", "string", false},
				{"enable_flutter_driver", "Flutter Driver", "bool", false},
				{"flutter_driver_args", "Driver Args", "string", false},
			},
		},
		{
			title: "Signing & Distribution",
			fields: []field{
				{"keystore_id", "Keystore ID", "int", false},
				{"enable_android_code_signing", "Android Code Signing", "bool", false},
				{"google_play_credentials_id", "Play Credentials ID", "int", false},
				{"google_play_track", "Play Track", "string", false},
				{"enable_google_play_publishing", "Play Publishing", "bool", false},
				{"rollout_fraction", "Rollout Fraction", "float", false},
				{"submit_as_draft", "Submit as Draft", "bool", false},
				{"do_not_send_for_review", "Skip Review", "bool", false},
				{"publish_even_if_tests_fail", "Publish on Fail", "bool", false},
				{"update_priority", "Update Priority", "int", false},
			},
		},
		{
			title: "Caching",
			fields: []field{
				{"dependency_caching", "Dependency Caching", "bool", false},
				{"dependency_dirs", "Dependency Dirs (comma-separated)", "slice", false},
			},
		},
		{
			title: "Scripts",
			fields: []field{
				{"post_clone_script", "Post Clone", "string", false},
				{"pre_build_script", "Pre Build", "string", false},
				{"post_build_script", "Post Build", "string", false},
				{"pre_test_script", "Pre Test", "string", false},
				{"post_test_script", "Post Test", "string", false},
				{"pre_publish_script", "Pre Publish", "string", false},
			},
		},
		{
			title: "Notifications",
			fields: []field{
				{"enable_email_notifications", "Email Notifications", "bool", false},
				{"email_recipients", "Recipients (comma-separated)", "slice", false},
				{"webhook_urls", "Webhook URLs (comma-separated)", "slice", false},
			},
		},
	}

	for i, sec := range sections {
		fmt.Printf("\n%s── %s ──%s\n", display.Bold, sec.title, display.Reset)
		sectionChanged := false

		for _, f := range sec.fields {
			curVal := current[f.key]
			prompt := formatPrompt(f.label, curVal, f.typ, f.sensitive)
			fmt.Print(prompt)
			input, err := r.ReadString('\n')
			if err != nil {
				return fmt.Errorf("reading input: %w", err)
			}
			input = strings.TrimSpace(input)

			if input == "" {
				continue // keep current
			}
			if input == "-" {
				// Clear: send null-like value
				body[f.key] = nil
				sectionChanged = true
				continue
			}

			val, err := parseFieldValue(input, f.typ)
			if err != nil {
				fmt.Printf("  %s✗ %s%s\n", display.Error, err, display.Reset)
				continue
			}
			body[f.key] = val
			sectionChanged = true
		}

		if !sectionChanged {
			fmt.Printf("  %s(unchanged)%s\n", display.Muted, display.Reset)
		}

		// Ask to continue (skip on last section)
		if i < len(sections)-1 {
			fmt.Printf("\nContinue to %s%s%s? [Y/n/q] ", display.Bold, sections[i+1].title, display.Reset)
			choice, _ := r.ReadString('\n')
			choice = strings.TrimSpace(strings.ToLower(choice))
			if choice == "q" || choice == "quit" {
				fmt.Println("Cancelled.")
				return nil
			}
			if choice == "n" || choice == "no" {
				break
			}
		}
	}

	if len(body) == 0 {
		fmt.Printf("\n%sNothing to update.%s\n", display.Muted, display.Reset)
		return nil
	}

	// Confirm
	fmt.Println()
	fmt.Printf("%sFields to update:%s\n", display.Bold, display.Reset)
	for k, v := range body {
		if v == nil {
			fmt.Printf("  %s (clear)\n", k)
		} else {
			fmt.Printf("  %s = %v\n", k, v)
		}
	}
	fmt.Print("\nApply these changes? [Y/n] ")
	confirm, _ := r.ReadString('\n')
	confirm = strings.TrimSpace(strings.ToLower(confirm))
	if confirm != "" && confirm != "y" && confirm != "yes" {
		fmt.Println("Cancelled.")
		return nil
	}

	if err := client.PostJSON(host, "/project/"+id+"/config", body, nil); err != nil {
		return fmt.Errorf("updating config: %w", err)
	}
	display.SuccessPrint("Config for project %s updated", id)
	return nil
}

func formatPrompt(label string, cur interface{}, typ string, sensitive bool) string {
	current := formatCurrent(cur, typ, sensitive)
	if sensitive && cur != nil && cur != "" {
		current = "[set]"
	}
	return fmt.Sprintf("  %s %s: ", label, current)
}

func formatCurrent(v interface{}, typ string, sensitive bool) string {
	if v == nil || v == "" {
		return "[not set]"
	}
	if sensitive {
		return "[set]"
	}
	switch typ {
	case "bool":
		if b, ok := v.(bool); ok && b {
			return "[yes]"
		}
		return "[no]"
	case "slice":
		if arr, ok := v.([]interface{}); ok {
			parts := make([]string, len(arr))
			for i, e := range arr {
				parts[i] = fmt.Sprintf("%v", e)
			}
			return "[" + strings.Join(parts, ", ") + "]"
		}
		return fmt.Sprintf("[%v]", v)
	default:
		return fmt.Sprintf("[%v]", v)
	}
}

func parseFieldValue(input string, typ string) (interface{}, error) {
	switch typ {
	case "bool":
		switch strings.ToLower(input) {
		case "y", "yes", "true", "1":
			return true, nil
		case "n", "no", "false", "0":
			return false, nil
		default:
			return nil, fmt.Errorf("invalid boolean: %s (use y/n)", input)
		}
	case "int":
		n, err := strconv.ParseInt(input, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid integer: %s", input)
		}
		return n, nil
	case "float":
		f, err := strconv.ParseFloat(input, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid float: %s", input)
		}
		return f, nil
	case "slice":
		if input == "" {
			return []string{}, nil
		}
		parts := strings.Split(input, ",")
		out := make([]string, 0, len(parts))
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p != "" {
				out = append(out, p)
			}
		}
		return out, nil
	default:
		return input, nil
	}
}

func init() {
	configUpdateCmd.Flags().String("flutter", "", "Flutter version")
	configUpdateCmd.Flags().String("mode", "", "Build mode (debug, release, profile)")
	configUpdateCmd.Flags().String("repo", "", "Git repository URL")
	configUpdateCmd.Flags().StringSlice("platform", nil, "Platforms (android, ios, web)")
	configUpdateCmd.Flags().Bool("tty", false, "Interactive TTY mode for line-by-line config editing")

	projectConfigCmd.AddCommand(configGetCmd)
	projectConfigCmd.AddCommand(configUpdateCmd)
	projectConfigCmd.AddCommand(configDeleteCmd)
	rootCmd.AddCommand(projectConfigCmd)
}

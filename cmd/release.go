package cmd

import (
	"fmt"
	"strconv"

	"github.com/flotio-dev/cli/pkg/client"
	"github.com/flotio-dev/cli/pkg/display"
	"github.com/spf13/cobra"
)

var releaseCmd = &cobra.Command{
	Use:   "release",
	Short: "Manage Google Play releases",
	Long:  `Publish builds to Google Play and track release status.`,
	Example: `  flotio release publish 42
  flotio release list
  flotio release get 7
  flotio release access
  flotio release audit`,
}

var releasePublishCmd = &cobra.Command{
	Use:     "publish [project-id] <build-id>",
	Short:   "Publish a successful build to Google Play",
	Args:    cobra.RangeArgs(1, 2),
	Example: `  flotio release publish 42
  flotio release publish 3 42 --track production --notes "Bug fixes"`,
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

		body := map[string]interface{}{}
		track, _ := cmd.Flags().GetString("track")
		notes, _ := cmd.Flags().GetString("notes")
		notesLang, _ := cmd.Flags().GetString("notes-lang")
		draft, _ := cmd.Flags().GetBool("draft")
		if track != "" {
			body["track"] = track
		}
		if notes != "" {
			body["release_notes"] = notes
		}
		if notesLang != "" {
			body["release_notes_lang"] = notesLang
		}
		if cmd.Flags().Changed("draft") {
			body["draft"] = draft
		}

		var wrapper map[string]interface{}
		path := fmt.Sprintf("/project/%d/build/%d/publish", projectID, buildID)
		if err := client.PostJSON(cfg.ResolveHost(), path, body, &wrapper); err != nil {
			return fmt.Errorf("publishing build: %w", err)
		}

		if display.JSONOutput() {
			display.PrintJSON(wrapper)
			return nil
		}
		rel, _ := wrapper["release"].(map[string]interface{})
		display.SuccessPrint("Release triggered: build %d → track %s", buildID, relStr(rel, "track"))
		display.KeyValue("Release ID", "%v", rel["id"])
		display.KeyValue("Status", "%v", rel["status"])
		display.KeyValue("Version", "%v", rel["version_name"])
		return nil
	},
}

var releaseGetCmd = &cobra.Command{
	Use:     "get [project-id] <release-id>",
	Short:   "Get a release and its status",
	Args:    cobra.RangeArgs(1, 2),
	Example: `  flotio release get 7`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if !client.IsLoggedIn() {
			return fmt.Errorf("not logged in")
		}
		projectArg := ""
		relArg := args[0]
		if len(args) == 2 {
			projectArg = args[0]
			relArg = args[1]
		}
		projectID, err := parseProjectID(projectArg)
		if err != nil {
			return err
		}
		releaseID, err := strconv.ParseInt(relArg, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid release ID: %s", relArg)
		}

		var wrapper map[string]interface{}
		path := fmt.Sprintf("/project/%d/release/%d", projectID, releaseID)
		if err := client.GetJSON(cfg.ResolveHost(), path, &wrapper); err != nil {
			return fmt.Errorf("getting release: %w", err)
		}
		rel, ok := wrapper["release"].(map[string]interface{})
		if !ok {
			return fmt.Errorf("unexpected response format")
		}
		if display.JSONOutput() {
			display.PrintJSON(rel)
			return nil
		}
		display.HeadingPrint("Release %v", rel["id"])
		display.KeyValue("Status", "%v", rel["status"])
		display.KeyValue("Track", "%v", rel["track"])
		display.KeyValue("Build ID", "%v", rel["build_id"])
		display.KeyValue("Version", "%v", rel["version_name"])
		display.KeyValue("Version Code", "%v", rel["version_code"])
		display.KeyValue("Rollout", "%.0f%%", rel["rollout_fraction"])
		if rn := rel["release_notes"]; rn != nil && rn != "" {
			display.KeyValue("Notes", "%v", rn)
		}
		return nil
	},
}

var releaseListCmd = &cobra.Command{
	Use:     "list [project-id]",
	Short:   "List releases for a project",
	Args:    cobra.MaximumNArgs(1),
	Example: `  flotio release list
  flotio release list 3`,
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

		var wrapper map[string]interface{}
		path := fmt.Sprintf("/project/%d/releases", projectID)
		if err := client.GetJSON(cfg.ResolveHost(), path, &wrapper); err != nil {
			return fmt.Errorf("listing releases: %w", err)
		}
		items, _ := client.ExtractList(wrapper)
		if display.JSONOutput() {
			if len(items) == 0 {
				fmt.Println("[]")
			} else {
				display.PrintJSON(items)
			}
			return nil
		}
		if len(items) == 0 {
			display.NoResults("releases")
			return nil
		}
		table := &display.Table{
			Columns: []display.Column{
				{Header: "ID", Width: 5, Align: 1},
				{Header: "Status", Width: 12},
				{Header: "Track", Width: 10},
				{Header: "Build", Width: 7, Align: 1},
				{Header: "Version", Max: 15},
			},
		}
		for _, raw := range items {
			r, ok := raw.(map[string]interface{})
			if !ok {
				continue
			}
			table.AddRow(
				fmt.Sprintf("%v", r["id"]),
				display.StatusBadge(fmt.Sprintf("%v", r["status"])),
				fmt.Sprintf("%v", r["track"]),
				fmt.Sprintf("%v", r["build_id"]),
				fmt.Sprintf("%v", r["version_name"]),
			)
		}
		table.Render()
		return nil
	},
}

var releaseAccessCmd = &cobra.Command{
	Use:     "access [project-id]",
	Short:   "Check Google Play access for a project",
	Args:    cobra.MaximumNArgs(1),
	Example: `  flotio release access`,
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

		var res map[string]interface{}
		path := fmt.Sprintf("/project/%d/google-play/access", projectID)
		if err := client.GetJSON(cfg.ResolveHost(), path, &res); err != nil {
			return fmt.Errorf("checking access: %w", err)
		}
		if display.JSONOutput() {
			display.PrintJSON(res)
			return nil
		}
		accessible, _ := res["accessible"].(bool)
		if accessible {
			display.SuccessPrint("Google Play access verified")
		} else {
			display.ErrorPrint("Google Play access denied")
			if r, ok := res["reason"].(string); ok && r != "" {
				display.KeyValue("Reason", "%s", r)
			}
			if m, ok := res["message"].(string); ok && m != "" {
				display.KeyValue("Detail", "%s", m)
			}
		}
		return nil
	},
}

var releaseAuditCmd = &cobra.Command{
	Use:     "audit [project-id]",
	Short:   "List the publication audit log",
	Args:    cobra.MaximumNArgs(1),
	Example: `  flotio release audit`,
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

		var wrapper map[string]interface{}
		path := fmt.Sprintf("/project/%d/audit", projectID)
		if err := client.GetJSON(cfg.ResolveHost(), path, &wrapper); err != nil {
			return fmt.Errorf("listing audit log: %w", err)
		}
		items, _ := client.ExtractList(wrapper)
		if display.JSONOutput() {
			if len(items) == 0 {
				fmt.Println("[]")
			} else {
				display.PrintJSON(items)
			}
			return nil
		}
		if len(items) == 0 {
			display.NoResults("audit entries")
			return nil
		}
		table := &display.Table{
			Columns: []display.Column{
				{Header: "ID", Width: 5, Align: 1},
				{Header: "Action", Width: 12},
				{Header: "Track", Width: 10},
				{Header: "Pkg", Max: 25},
				{Header: "VC", Width: 8, Align: 1},
				{Header: "Detail", Max: 30},
			},
		}
		for _, raw := range items {
			e, ok := raw.(map[string]interface{})
			if !ok {
				continue
			}
			table.AddRow(
				fmt.Sprintf("%v", e["id"]),
				fmt.Sprintf("%v", e["action"]),
				fmt.Sprintf("%v", e["track"]),
				fmt.Sprintf("%v", e["package_name"]),
				fmt.Sprintf("%v", e["version_code"]),
				display.Truncate(fmt.Sprintf("%v", e["detail"]), 30),
			)
		}
		table.Render()
		return nil
	},
}

// relStr safely extracts a string field from a release map.
func relStr(m map[string]interface{}, key string) string {
	if m == nil {
		return "-"
	}
	v, ok := m[key]
	if !ok || v == nil {
		return "-"
	}
	return fmt.Sprintf("%v", v)
}

func init() {
	releasePublishCmd.Flags().String("track", "", "Override the Google Play track (internal, alpha, beta, production)")
	releasePublishCmd.Flags().String("notes", "", "Release notes")
	releasePublishCmd.Flags().String("notes-lang", "", "Language code for release notes (e.g. en-US)")
	releasePublishCmd.Flags().Bool("draft", false, "Submit as draft instead of publishing")

	releaseCmd.AddCommand(releasePublishCmd)
	releaseCmd.AddCommand(releaseGetCmd)
	releaseCmd.AddCommand(releaseListCmd)
	releaseCmd.AddCommand(releaseAccessCmd)
	releaseCmd.AddCommand(releaseAuditCmd)
	rootCmd.AddCommand(releaseCmd)
}

package cmd

import (
	"fmt"
	"strings"

	"github.com/flotio-dev/cli/pkg/client"
	"github.com/flotio-dev/cli/pkg/display"
	"github.com/spf13/cobra"
)

var cacheCmd = &cobra.Command{
	Use:   "cache",
	Short: "Manage build cache",
	Long:  `Inspect and purge the dependency build cache for a project.`,
	Example: `  flotio cache metrics
  flotio cache entries --branch main
  flotio cache purge --branch main`,
}

var cachePurgeCmd = &cobra.Command{
	Use:     "purge [project-id]",
	Short:   "Purge the build cache for a project branch",
	Args:    cobra.MaximumNArgs(1),
	Example: `  flotio cache purge --branch main`,
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

		path := fmt.Sprintf("/project/%d/cache", projectID)
		params := []string{}
		if branch != "" {
			params = append(params, "branch="+branch)
		}
		if len(params) > 0 {
			path += "?" + strings.Join(params, "&")
		}
		if err := client.DeleteJSON(cfg.ResolveHost(), path); err != nil {
			return fmt.Errorf("purging cache: %w", err)
		}
		scope := "all branches"
		if branch != "" {
			scope = "branch " + branch
		}
		display.SuccessPrint("Cache purged for project %d (%s)", projectID, scope)
		return nil
	},
}

var cacheMetricsCmd = &cobra.Command{
	Use:     "metrics [project-id]",
	Short:   "Show build cache metrics",
	Args:    cobra.MaximumNArgs(1),
	Example: `  flotio cache metrics --branch main`,
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

		path := fmt.Sprintf("/project/%d/cache/metrics", projectID)
		if branch != "" {
			path += "?branch=" + branch
		}
		var m map[string]interface{}
		if err := client.GetJSON(cfg.ResolveHost(), path, &m); err != nil {
			return fmt.Errorf("fetching cache metrics: %w", err)
		}
		if display.JSONOutput() {
			display.PrintJSON(m)
			return nil
		}
		display.HeadingPrint("Cache Metrics (project %d)", projectID)
		display.KeyValue("Namespace", "%v", m["namespace"])
		display.KeyValue("Objects", "%v", m["object_count"])
		display.KeyValue("Size", "%s", humanBytes(m["total_size_bytes"]))
		if la := m["last_modified_at"]; la != nil {
			display.KeyValue("Last Modified", "%v", la)
		}
		if pr := m["purge_requests"]; pr != nil {
			display.KeyValue("Purge Requests", "%v", pr)
		}
		if ttl := m["retention_ttl_hours"]; ttl != nil {
			display.KeyValue("Retention TTL", "%vh", ttl)
		}
		return nil
	},
}

var cacheEntriesCmd = &cobra.Command{
	Use:     "entries [project-id]",
	Short:   "List cache entries (fingerprints) for a branch",
	Args:    cobra.MaximumNArgs(1),
	Example: `  flotio cache entries --branch main`,
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
		if branch == "" {
			return fmt.Errorf("--branch is required for cache entries")
		}

		path := fmt.Sprintf("/project/%d/cache/entries?branch=%s", projectID, branch)
		var wrapper map[string]interface{}
		if err := client.GetJSON(cfg.ResolveHost(), path, &wrapper); err != nil {
			return fmt.Errorf("listing cache entries: %w", err)
		}
		entries, ok := wrapper["entries"].([]interface{})
		if !ok || len(entries) == 0 {
			display.NoResults("cache entries")
			return nil
		}
		if display.JSONOutput() {
			display.PrintJSON(entries)
			return nil
		}
		table := &display.Table{
			Columns: []display.Column{
				{Header: "Fingerprint", Max: 40},
				{Header: "Objects", Width: 8, Align: 1},
				{Header: "Size", Width: 10},
			},
		}
		for _, raw := range entries {
			e, ok := raw.(map[string]interface{})
			if !ok {
				continue
			}
			table.AddRow(
				fmt.Sprintf("%v", e["fingerprint"]),
				fmt.Sprintf("%v", e["object_count"]),
				humanBytes(e["total_size_bytes"]),
			)
		}
		table.Render()
		return nil
	},
}

// humanBytes converts a numeric byte count (as interface{}) to a human-readable string.
func humanBytes(v interface{}) string {
	if v == nil {
		return "-"
	}
	f, ok := toFloat(v)
	if !ok {
		return fmt.Sprintf("%v", v)
	}
	const unit = 1024
	if f < unit {
		return fmt.Sprintf("%.0f B", f)
	}
	div, exp := float64(unit), 0
	for n := f / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", f/div, "KMGTPE"[exp])
}

// toFloat coerces numeric interface{} values (from JSON) to float64.
func toFloat(v interface{}) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	}
	return 0, false
}

func init() {
	cachePurgeCmd.Flags().String("branch", "", "Limit purge to a specific branch")
	cacheMetricsCmd.Flags().String("branch", "", "Scope metrics to a specific branch")
	cacheEntriesCmd.Flags().String("branch", "", "Branch name (required)")

	cacheCmd.AddCommand(cachePurgeCmd)
	cacheCmd.AddCommand(cacheMetricsCmd)
	cacheCmd.AddCommand(cacheEntriesCmd)
	rootCmd.AddCommand(cacheCmd)
}

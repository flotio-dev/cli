package cmd

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/flotio-dev/cli/internal/config"
	"github.com/flotio-dev/cli/pkg/client"
	"github.com/flotio-dev/cli/pkg/display"
	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:     "doctor",
	Short:   "Diagnose CLI configuration and connectivity",
	Example: `  flotio doctor`,
	RunE: func(cmd *cobra.Command, args []string) error {
		host := cfg.ResolveHost()
		failures := 0

		// 1. Config file
		fmt.Println(display.Bold + "Configuration" + display.Reset)
		fmt.Printf("  API host:        %s\n", host)
		if path, err := tokenDir(); err == nil {
			fmt.Printf("  Config dir:      %s\n", path)
		}
		fmt.Println()

		// 2. Auth state
		fmt.Println(display.Bold + "Authentication" + display.Reset)
		tokens, _ := client.LoadTokens()
		if tokens == nil || tokens.AccessToken == "" {
			display.ErrorPrint("Not logged in")
			fmt.Println("  → Run: flotio login -e <email> -p <password>")
			failures++
		} else {
			display.SuccessPrint("Logged in")
			if tokens.Email != "" {
				fmt.Printf("  Account:         %s\n", tokens.Email)
			}
			hasRefresh := tokens.RefreshToken != ""
			fmt.Printf("  Refresh token:   %s\n", display.Bool(hasRefresh))
		}
		fmt.Println()

		// 3. API reachability
		fmt.Println(display.Bold + "Connectivity" + display.Reset)
		client2 := &http.Client{Timeout: 10 * time.Second}
		resp, err := client2.Get(host + "/healthz")
		if err != nil {
			display.ErrorPrint("API unreachable: %v", err)
			failures++
		} else {
			resp.Body.Close()
			if resp.StatusCode == 200 {
				display.SuccessPrint("API reachable (%s)", host)
			} else {
				display.ErrorPrint("API returned %d on /healthz", resp.StatusCode)
				failures++
			}
		}
		fmt.Println()

		// 4. Project context
		fmt.Println(display.Bold + "Project context" + display.Reset)
		if cwd, err := os.Getwd(); err == nil {
			if pc, dir := config.FindProject(cwd); pc != nil {
				display.SuccessPrint("Project linked (ID %d in %s/.flotio.yaml)", pc.ProjectID, dir)
			} else {
				fmt.Println("  No .flotio.yaml in this directory (optional)")
			}
		}
		fmt.Println()

		// Verdict
		if failures > 0 {
			display.ErrorPrint("%d issue(s) found", failures)
			return fmt.Errorf("diagnosis incomplete — %d failure(s)", failures)
		}
		display.SuccessPrint("All checks passed")
		return nil
	},
}

// tokenDir returns the ~/.flotio directory path.
func tokenDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return home + "/.flotio", nil
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}

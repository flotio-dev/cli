package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
)

// Set at build time with -ldflags:
//
//	go build -ldflags "-X github.com/flotio-dev/cli/cmd.version=v1.2.3" .
var version = "dev"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Printf("flotio %s  %s/%s\n", version, runtime.GOOS, runtime.GOARCH)

		// Check for updates
		latest, err := fetchLatestVersion()
		if err != nil {
			return nil // silent — no network or GitHub down
		}
		if latest != "" && latest != version && version != "dev" {
			fmt.Printf("\nUpdate available: %s → %s\n", version, latest)
			fmt.Printf("Run 'flotio update' to upgrade.\n")
		} else if version == "dev" {
			fmt.Println("(development build — run 'flotio update' to download latest release)")
		}
		return nil
	},
}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update flotio to the latest version",
	Long: `Download the latest flotio binary from GitHub Releases and replace the current executable.

Requires write access to the current binary location.`,
	Example: `  flotio update`,
	RunE: func(cmd *cobra.Command, args []string) error {
		exe, err := os.Executable()
		if err != nil {
			return fmt.Errorf("cannot find current executable: %w", err)
		}

		// Build download URL
		goos := runtime.GOOS
		goarch := runtime.GOARCH
		ext := ""
		if goos == "windows" {
			ext = ".exe"
		}
		url := fmt.Sprintf("https://github.com/flotio-dev/cli/releases/latest/download/flotio-%s-%s%s", goos, goarch, ext)

		fmt.Printf("Downloading flotio-%s-%s...\n", goos, goarch)

		// Download
		resp, err := http.Get(url)
		if err != nil {
			return fmt.Errorf("download failed: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			return fmt.Errorf("download returned %d — no release for %s/%s yet?", resp.StatusCode, goos, goarch)
		}

		// Write to temp file, then replace
		tmp := exe + ".new"
		f, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
		if err != nil {
			return fmt.Errorf("creating temp file: %w", err)
		}
		if _, err := f.ReadFrom(resp.Body); err != nil {
			f.Close()
			os.Remove(tmp)
			return fmt.Errorf("writing download: %w", err)
		}
		f.Close()

		// Replace current binary
		if err := os.Rename(tmp, exe); err != nil {
			os.Remove(tmp)
			return fmt.Errorf("replacing binary (try running as admin): %w", err)
		}

		fmt.Println("✓ flotio updated to latest version")
		return nil
	},
}

// fetchLatestVersion returns the latest tag from GitHub releases, or "" on error.
func fetchLatestVersion() (string, error) {
	resp, err := http.Get("https://api.github.com/repos/flotio-dev/cli/releases/latest")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("github API returned %d", resp.StatusCode)
	}
	var release struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", err
	}
	return strings.TrimPrefix(release.TagName, "v"), nil
}

func init() {
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(updateCmd)
}

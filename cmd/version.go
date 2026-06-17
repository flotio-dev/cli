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

type releaseInfo struct {
	TagName string `json:"tag_name"`
	Body    string `json:"body"`
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Printf("flotio %s  %s/%s\n", version, runtime.GOOS, runtime.GOARCH)

		// Check for updates
		rel, err := fetchLatestRelease()
		if err != nil {
			return nil // silent
		}
		latest := strings.TrimPrefix(rel.TagName, "v")
		if latest != "" && latest != version && version != "dev" {
			fmt.Printf("\nUpdate available: %s → %s\n", version, latest)
			if rel.Body != "" {
				fmt.Println(strings.TrimSpace(rel.Body))
			}
			fmt.Printf("\nRun 'flotio update' to upgrade.\n")
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

If the current binary requires admin privileges to overwrite (e.g. installed
in /usr/local/bin), run the install script instead:
  curl -fsSL https://raw.githubusercontent.com/flotio-dev/cli/main/install.sh | sh`,
	Example: `  flotio update`,
	RunE: func(cmd *cobra.Command, args []string) error {
		exe, err := os.Executable()
		if err != nil {
			return fmt.Errorf("cannot find current executable: %w", err)
		}

		// Check write permission
		if !canWrite(exe) {
			return fmt.Errorf(
				"no write permission for %s\n\n"+
					"The CLI was likely installed with sudo. Update via the install script:\n"+
					"  curl -fsSL https://raw.githubusercontent.com/flotio-dev/cli/main/install.sh | sh",
				exe)
		}

		// Show release notes
		rel, err := fetchLatestRelease()
		if err == nil && rel.Body != "" {
			fmt.Println(strings.TrimSpace(rel.Body))
			fmt.Println()
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
			return fmt.Errorf("replacing binary: %w", err)
		}

		fmt.Println("✓ flotio updated to latest version")
		return nil
	},
}

// canWrite checks if the current process can write to the given file path.
func canWrite(path string) bool {
	// Try opening for append — doesn't modify the file, just tests access
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0)
	if err != nil {
		return false
	}
	f.Close()
	return true
}

// fetchLatestRelease returns the latest GitHub release info.
func fetchLatestRelease() (*releaseInfo, error) {
	resp, err := http.Get("https://api.github.com/repos/flotio-dev/cli/releases/latest")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("github API returned %d", resp.StatusCode)
	}
	var rel releaseInfo
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, err
	}
	return &rel, nil
}

func init() {
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(updateCmd)
}

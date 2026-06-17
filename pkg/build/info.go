// Package build provides compile-time version information
// injected via ldflags.
//
// Build with:
//
//	go build -ldflags "-X github.com/flotio-dev/cli/pkg/build.Version=1.0.0 \
//	                  -X github.com/flotio-dev/cli/pkg/build.Commit=$(git rev-parse --short HEAD) \
//	                  -X github.com/flotio-dev/cli/pkg/build.Date=$(date -u +%Y-%m-%dT%H:%M:%SZ)" .
package build

import (
	"fmt"
	"runtime"
)

// These variables are set at build time via ldflags.
var (
	// Version is the semantic version of the CLI.
	Version = "dev"

	// Commit is the git commit hash at build time.
	Commit = "none"

	// Date is the UTC build timestamp.
	Date = "unknown"
)

// Summary returns a human-readable version string.
func Summary() string {
	return fmt.Sprintf(
		"flotio %s\n  commit:  %s\n  built:   %s\n  go:      %s\n  os/arch: %s/%s",
		Version,
		Commit,
		Date,
		runtime.Version(),
		runtime.GOOS,
		runtime.GOARCH,
	)
}

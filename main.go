// Package main is the entry point for the Flotio CLI.
//
// Flotio CLI is a developer tool for managing cloud infrastructure,
// deploying projects, and interacting with the Flotio platform.
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/flotio-dev/cli/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		msg := err.Error()
		fmt.Fprintf(os.Stderr, "Error: %s\n", msg)

		// If the error looks like a usage mistake (missing args, unknown flag, etc.),
		// suggest --help. Cobra-generated errors follow predictable patterns.
		if isUsageError(msg) {
			cmdName := extractCommand(os.Args)
			if cmdName != "" {
				fmt.Fprintf(os.Stderr, "Run 'flotio %s --help' for usage.\n", cmdName)
			} else {
				fmt.Fprintf(os.Stderr, "Run 'flotio --help' for usage.\n")
			}
		}
		os.Exit(1)
	}
}

// isUsageError returns true for cobra-generated usage errors
// and common mistake patterns that warrant a --help hint.
func isUsageError(msg string) bool {
	patterns := []string{
		"unknown command",
		"unknown flag",
		"unknown shorthand flag",
		"requires at least",
		"requires exactly",
		"accepts at most",
		"invalid argument",
		"flag needs an argument",
		"required flag",
		"is required",
		"not logged in",
	}
	for _, p := range patterns {
		if strings.Contains(strings.ToLower(msg), p) {
			return true
		}
	}
	return false
}

// extractCommand extracts the command path from os.Args (e.g., "project list").
func extractCommand(args []string) string {
	var parts []string
	for _, a := range args[1:] { // skip program name
		if strings.HasPrefix(a, "-") {
			break
		}
		parts = append(parts, a)
	}
	return strings.Join(parts, " ")
}

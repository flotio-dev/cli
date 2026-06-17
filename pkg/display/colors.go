// Package display provides styled terminal output for the Flotio CLI:
// tables with box-drawing borders, colored text, and key-value displays.
package display

import "fmt"

// ANSI escape codes.
const (
	Reset  = "\033[0m"
	Bold   = "\033[1m"
	Dim    = "\033[2m"
	Italic = "\033[3m"

	Red     = "\033[31m"
	Green   = "\033[32m"
	Yellow  = "\033[33m"
	Blue    = "\033[34m"
	Magenta = "\033[35m"
	Cyan    = "\033[36m"
	White   = "\033[37m"
	Gray    = "\033[90m"

	BgRed   = "\033[41m"
	BgGreen = "\033[42m"
)

// Styles for common elements.
var (
	Heading = Bold + Cyan
	Success = Green
	Error   = Red
	Warning = Yellow
	Muted   = Gray
	Label   = Dim
	Value   = Bold + White
	ID      = Bold + Magenta
	Status  = Bold
)

// Success prints "✓ message" in green.
func SuccessPrint(msg string, args ...interface{}) {
	fmt.Printf(Green+"✓ "+Reset+msg+"\n", args...)
}

// ErrorPrint prints "✗ message" in red.
func ErrorPrint(msg string, args ...interface{}) {
	fmt.Printf(Red+"✗ "+Reset+msg+"\n", args...)
}

// HeadingPrint prints a bold cyan heading.
func HeadingPrint(msg string, args ...interface{}) {
	fmt.Printf(Heading+msg+Reset+"\n", args...)
}

// KeyValue prints "  label: value" with dim label and bright value.
func KeyValue(label, value string, args ...interface{}) {
	fmt.Printf("  "+Label+"%-14s"+Reset+" "+Value, label+":")
	fmt.Printf(value+"\n", args...)
}

// StatusBadge returns a colored status string.
func StatusBadge(status string) string {
	switch status {
	case "success", "completed", "active", "running":
		return Green + status + Reset
	case "failed", "error", "cancelled":
		return Red + status + Reset
	case "pending", "waiting":
		return Yellow + status + Reset
	default:
		return status
	}
}

// NoResults prints a muted "no results" line.
func NoResults(label string) {
	fmt.Println(Muted + "  No " + label + " found." + Reset)
}

// Bool returns a colored yes/no string.
func Bool(v bool) string {
	if v {
		return Green + "yes" + Reset
	}
	return Dim + "no" + Reset
}

// Truncate cuts a string to maxLen, appending "…" if truncated.
func Truncate(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen-1]) + "…"
}

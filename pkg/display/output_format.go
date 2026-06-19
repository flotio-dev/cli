package display

import (
	"encoding/json"
	"fmt"
	"os"
)

// outputFormat controls how structured data is rendered: "table" (default) or
// "json". Set once from the global --output flag during PersistentPreRunE.
var outputFormat = "table"

// SetOutputFormat configures the global output mode.
func SetOutputFormat(f string) {
	if f != "" {
		outputFormat = f
	}
}

// JSONOutput reports whether the CLI is in JSON output mode.
func JSONOutput() bool {
	return outputFormat == "json"
}

// PrintJSON pretty-prints v as indented JSON to stdout.
func PrintJSON(v interface{}) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		// Fall back to a minimal representation; never crash on output.
		fmt.Fprintf(os.Stdout, "%v\n", v)
		return
	}
	fmt.Println(string(b))
}

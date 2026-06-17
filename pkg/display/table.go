package display

import (
	"fmt"
	"strings"
)

// Box-drawing characters.
const (
	BoxH  = "─"
	BoxV  = "│"
	BoxTL = "╭"
	BoxTR = "╮"
	BoxBL = "╰"
	BoxBR = "╯"
	BoxML = "├"
	BoxMR = "┤"
	BoxMT = "┬"
	BoxMB = "┴"
	BoxX  = "┼"
)

// Column defines a table column.
type Column struct {
	Header string // Column header text.
	Width  int    // Fixed width (0 = auto-size from data).
	Max    int    // Maximum width for auto-sizing (0 = no limit).
	Align  int    // -1=left, 0=center, 1=right.
}

// Table builds and prints a box-drawing table.
type Table struct {
	Columns []Column
	Rows    [][]string
}

// AddRow adds a row of string cells.
func (t *Table) AddRow(cells ...string) {
	t.Rows = append(t.Rows, cells)
}

// Render prints the table to stdout.
func (t *Table) Render() {
	// Compute column widths
	widths := make([]int, len(t.Columns))
	for i, col := range t.Columns {
		w := len([]rune(col.Header))
		if col.Width > 0 {
			widths[i] = col.Width
		} else {
			for _, row := range t.Rows {
				if i < len(row) {
					cw := len([]rune(stripANSI(row[i])))
					if cw > w {
						w = cw
					}
				}
			}
			if col.Max > 0 && w > col.Max {
				w = col.Max
			}
			widths[i] = w
		}
	}

	// Top border
	t.printBorder(BoxTL, BoxMT, BoxTR, widths)
	// Header
	t.printRow(t.Columns, widths, true)
	// Separator
	t.printBorder(BoxML, BoxX, BoxMR, widths)
	// Data rows
	for _, row := range t.Rows {
		t.printRowData(row, widths)
	}
	// Bottom border
	t.printBorder(BoxBL, BoxMB, BoxBR, widths)
}

func (t *Table) printBorder(left, mid, right string, widths []int) {
	fmt.Print(left)
	for i, w := range widths {
		fmt.Print(strings.Repeat(BoxH, w+2)) // +2 for padding
		if i < len(widths)-1 {
			fmt.Print(mid)
		}
	}
	fmt.Println(right)
}

func (t *Table) printRow(cols []Column, widths []int, header bool) {
	fmt.Print(BoxV)
	for i, w := range widths {
		text := ""
		if header {
			text = Bold + cols[i].Header + Reset
		}
		fmt.Print(" " + padANSI(text, w, cols[i].Align) + " " + BoxV)
	}
	fmt.Println()
}

func (t *Table) printRowData(cells []string, widths []int) {
	fmt.Print(BoxV)
	for i, w := range widths {
		text := ""
		align := -1 // default left
		if i < len(t.Columns) {
			align = t.Columns[i].Align
		}
		if i < len(cells) {
			text = cells[i]
		}
		fmt.Print(" " + padANSI(text, w, align) + " " + BoxV)
	}
	fmt.Println()
}

// padANSI pads a string (possibly containing ANSI codes) to the given width.
// align: -1=left, 0=center, 1=right.
func padANSI(s string, width int, align int) string {
	visible := []rune(stripANSI(s))
	pad := width - len(visible)
	if pad <= 0 {
		// Truncate
		if len(visible) > width {
			return s[:len([]rune(s))-len(visible)+width-1] + "…"
		}
		return s
	}
	switch align {
	case 1: // right
		return strings.Repeat(" ", pad) + s
	case 0: // center
		left := pad / 2
		right := pad - left
		return strings.Repeat(" ", left) + s + strings.Repeat(" ", right)
	default: // left
		return s + strings.Repeat(" ", pad)
	}
}

// stripANSI removes ANSI escape sequences from a string.
func stripANSI(s string) string {
	var b strings.Builder
	inEscape := false
	for _, r := range s {
		if r == '\033' {
			inEscape = true
			continue
		}
		if inEscape {
			if r == 'm' {
				inEscape = false
			}
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

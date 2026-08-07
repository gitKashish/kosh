package ui

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// Table provides a lightweight, zero-dependency way to render aligned columnar data
// in the terminal using the ui package's existing colors and writers.
type Table struct {
	Headers        []string
	Rows           [][]string
	ActiveRowIndex int
}

// NewTable initializes a new table with the given column headers.
func NewTable(headers ...string) *Table {
	return &Table{
		Headers:        headers,
		ActiveRowIndex: -1, // -1 means no active row
	}
}

// AddRow appends a new row to the table.
// Note: Pass raw strings here. If you include ANSI escape codes inside the string,
// the column width calculation will be skewed.
func (t *Table) AddRow(cols ...string) {
	t.Rows = append(t.Rows, cols)
}

// SetActive highlights a specific row index (useful for showing the active profile).
func (t *Table) SetActive(index int) {
	t.ActiveRowIndex = index
}

// Display prints the formatted table to the configured output writer.
func (t *Table) Display() {
	if len(t.Headers) == 0 {
		return
	}

	// 1. Calculate the max width for each column based on content length
	colWidths := make([]int, len(t.Headers))
	for i, h := range t.Headers {
		colWidths[i] = utf8.RuneCountInString(h)
	}
	for _, row := range t.Rows {
		for i, col := range row {
			if i < len(colWidths) {
				length := utf8.RuneCountInString(col)
				if length > colWidths[i] {
					colWidths[i] = length
				}
			}
		}
	}

	// 2. Build the formatting string (e.g., "%-10s   %-20s   %-15s")
	var formatBuilder strings.Builder
	for _, w := range colWidths {
		// 3 spaces padding between columns for readability
		fmt.Fprintf(&formatBuilder, "%%-%ds   ", w)
	}
	formatStr := strings.TrimRight(formatBuilder.String(), " ") + "\n"

	// 3. Print Headers (cyan colored, indented to align with the row pointer)
	fmt.Fprintf(out, "  %s", ColorCyan)
	headerArgs := make([]any, len(t.Headers))
	for i, h := range t.Headers {
		headerArgs[i] = h
	}
	fmt.Fprintf(out, formatStr, headerArgs...)
	fmt.Fprint(out, ColorReset)

	// 4. Print separator line beneath headers
	fmt.Fprintf(out, "  %s", ColorGray)
	for i, w := range colWidths {
		fmt.Fprint(out, strings.Repeat("-", w))
		if i < len(colWidths)-1 {
			fmt.Fprint(out, "   ")
		}
	}
	fmt.Fprint(out, ColorReset+"\n")

	// 5. Print Rows
	for rIdx, row := range t.Rows {
		rowArgs := make([]any, len(t.Headers))
		for i := range t.Headers {
			if i < len(row) {
				rowArgs[i] = row[i]
			} else {
				rowArgs[i] = "" // Pad missing columns with empty strings
			}
		}

		// Highlight the active row with a green pointer
		if rIdx == t.ActiveRowIndex {
			fmt.Fprintf(out, "%s> ", ColorGreen)
			fmt.Fprintf(out, formatStr, rowArgs...)
			fmt.Fprint(out, ColorReset)
		} else {
			fmt.Fprint(out, "  ") // Normal indentation
			fmt.Fprintf(out, formatStr, rowArgs...)
		}
	}
	fmt.Fprintln(out) // Trailing newline for spacing
}

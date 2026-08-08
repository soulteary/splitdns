package output

import (
	"fmt"
	"io"
	"strings"
)

// Table renders rows as an aligned text table with the given headers. Column
// widths are computed from headers and cell contents.
func Table(w io.Writer, headers []string, rows [][]string) {
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = len(h)
	}
	for _, row := range rows {
		for i, cell := range row {
			if i < len(widths) && len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}

	writeRow(w, headers, widths)
	sep := make([]string, len(headers))
	for i := range sep {
		sep[i] = strings.Repeat("-", widths[i])
	}
	writeRow(w, sep, widths)
	for _, row := range rows {
		writeRow(w, row, widths)
	}
}

func writeRow(w io.Writer, cells []string, widths []int) {
	parts := make([]string, len(widths))
	for i := range widths {
		cell := ""
		if i < len(cells) {
			cell = cells[i]
		}
		parts[i] = fmt.Sprintf("%-*s", widths[i], cell)
	}
	fmt.Fprintln(w, strings.TrimRight(strings.Join(parts, "  "), " "))
}

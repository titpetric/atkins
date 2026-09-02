package helpdoc

import (
	"fmt"
	"io"
	"strings"

	"github.com/titpetric/atkins/colors"
)

// table renders a GFM pipe table with padded cells.
//
// The padding is what makes one renderer enough for both destinations: a
// padded table is a readable column layout in a terminal and a valid
// markdown table in a file, so `atkins --help > help.md` and
// `atkins --help` differ only in the colour on the headings.
type table struct {
	w       io.Writer
	headers []string
	rows    [][]string
}

// newTable starts a table with the given column headers.
func newTable(w io.Writer, headers ...string) *table {
	return &table{w: w, headers: headers}
}

// row appends one row. A row shorter than the header list is padded with
// empty cells, and a longer one is truncated, so a caller cannot produce
// a table markdown will not parse.
func (t *table) row(cells ...string) {
	out := make([]string, len(t.headers))
	for i := range out {
		if i < len(cells) {
			out[i] = strings.TrimSpace(cells[i])
		}
	}
	t.rows = append(t.rows, out)
}

// flush writes the table and resets it. A table with no rows writes
// nothing, so a section with nothing to say stays out of the document.
func (t *table) flush() {
	if len(t.rows) == 0 {
		return
	}

	widths := make([]int, len(t.headers))
	for i, header := range t.headers {
		widths[i] = colors.VisualLength(header)
	}
	for _, row := range t.rows {
		for i, cell := range row {
			if width := colors.VisualLength(cell); width > widths[i] {
				widths[i] = width
			}
		}
	}

	t.line(t.headers, widths)

	// The rule row spans the padding as well as the cell, which is the
	// shape mdox writes, so a help document redirected to a file is
	// already formatted the way the repository formats markdown.
	rule := make([]string, len(t.headers))
	for i := range rule {
		rule[i] = strings.Repeat("-", widths[i]+2)
	}
	fmt.Fprintf(t.w, "|%s|\n", strings.Join(rule, "|"))

	for _, row := range t.rows {
		t.line(row, widths)
	}

	t.rows = nil
}

// line writes one table row padded to the column widths.
func (t *table) line(cells []string, widths []int) {
	parts := make([]string, len(cells))
	for i, cell := range cells {
		parts[i] = cell + strings.Repeat(" ", widths[i]-colors.VisualLength(cell))
	}
	fmt.Fprintf(t.w, "| %s |\n", strings.Join(parts, " | "))
}

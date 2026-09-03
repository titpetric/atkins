package markdown

import (
	"regexp"
	"strings"

	"github.com/titpetric/atkins/colors"
)

// Render redraws every GFM pipe table in a markdown document as a
// bordered ANSI table, and leaves everything else untouched.
//
// With color false the document is returned exactly as it was written,
// so `atkins --help > help.md` or a piped `atkins --help | less` still
// gets plain markdown a table-aware reader can parse.
func Render(doc string, color bool) string {
	if !color {
		return doc
	}

	sections := Sections(doc)
	for i, section := range sections {
		if t, ok := ParseTable(section); ok {
			sections[i] = t.render()
		}
	}

	return strings.Join(sections, "\n\n")
}

// codeSpan matches a markdown inline code span: `text`.
var codeSpan = regexp.MustCompile("`([^`]+)`")

// render draws the table with box-drawing borders: a bright header row,
// dim borders, and an inline code span in each cell coloured the way a
// job or flag name is coloured everywhere else in the document.
func (t *Table) render() string {
	cols := len(t.Headers)

	header := colorizeRow(t.Headers, true)
	rows := make([][]string, len(t.Rows))
	for i, row := range t.Rows {
		rows[i] = colorizeRow(row, false)
	}

	widths := make([]int, cols)
	for c, cell := range header {
		widths[c] = colors.VisualLength(cell)
	}
	for _, row := range rows {
		for c, cell := range row {
			if c < cols {
				if w := colors.VisualLength(cell); w > widths[c] {
					widths[c] = w
				}
			}
		}
	}

	var b strings.Builder
	b.WriteString(hrule(widths, "┌", "┬", "┐"))
	b.WriteByte('\n')
	b.WriteString(dataRow(header, widths))
	b.WriteByte('\n')
	b.WriteString(hrule(widths, "├", "┼", "┤"))
	for _, row := range rows {
		b.WriteByte('\n')
		b.WriteString(dataRow(row, widths))
	}
	b.WriteByte('\n')
	b.WriteString(hrule(widths, "└", "┴", "┘"))

	return b.String()
}

// colorizeRow renders every cell's inline code spans in orange, the
// header row in bright white. A cell with no code span is left plain: a
// column like "What it does" is prose, not a value.
func colorizeRow(cells []string, header bool) []string {
	out := make([]string, len(cells))
	for i, cell := range cells {
		cell = codeSpan.ReplaceAllStringFunc(cell, func(m string) string {
			return colors.BrightOrange(m[1 : len(m)-1])
		})
		if header {
			cell = colors.BrightWhite(cell)
		}
		out[i] = cell
	}
	return out
}

// hrule draws a border line, one dash-filled span per column.
func hrule(widths []int, left, mid, right string) string {
	parts := make([]string, len(widths))
	for i, w := range widths {
		parts[i] = strings.Repeat("─", w+2)
	}
	return colors.MediumGray(left + strings.Join(parts, mid) + right)
}

// dataRow pads every cell to its column width and encloses the row in
// dim vertical borders.
func dataRow(cells []string, widths []int) string {
	border := colors.MediumGray("│")

	parts := make([]string, len(widths))
	for i := range widths {
		cell := ""
		if i < len(cells) {
			cell = cells[i]
		}
		pad := widths[i] - colors.VisualLength(cell)
		if pad < 0 {
			pad = 0
		}
		parts[i] = " " + cell + strings.Repeat(" ", pad) + " "
	}

	return border + strings.Join(parts, border) + border
}

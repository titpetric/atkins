package markdown

import "strings"

// Table is a GFM pipe table decoded from a markdown section: a header
// row, and the data rows below the `---` separator.
type Table struct {
	Headers []string
	Rows    [][]string
}

// IsTable reports whether a section is a GFM pipe table: a row of cells
// followed by a separator row of dashes, colons and spaces only.
func IsTable(section string) bool {
	lines := strings.Split(section, "\n")
	if len(lines) < 2 {
		return false
	}
	return isRow(lines[0]) && isSeparatorRow(lines[1])
}

// ParseTable decodes a section as a table. It returns ok=false for a
// section IsTable rejects, so a caller can try it on every section
// without checking first.
func ParseTable(section string) (*Table, bool) {
	if !IsTable(section) {
		return nil, false
	}

	lines := strings.Split(section, "\n")

	t := &Table{Headers: splitRow(lines[0])}
	for _, line := range lines[2:] {
		if strings.TrimSpace(line) == "" {
			continue
		}
		t.Rows = append(t.Rows, splitRow(line))
	}

	return t, true
}

// isRow reports whether a line is a pipe-delimited table row: it starts
// and ends with `|`, once surrounding space is trimmed.
func isRow(line string) bool {
	line = strings.TrimSpace(line)
	return len(line) > 1 && strings.HasPrefix(line, "|") && strings.HasSuffix(line, "|")
}

// isSeparatorRow reports whether a line is the `|---|---|` rule under a
// table header: a row whose every cell is only dashes, colons and spaces.
func isSeparatorRow(line string) bool {
	if !isRow(line) {
		return false
	}
	for _, cell := range splitRow(line) {
		if cell == "" || strings.Trim(cell, "-: ") != "" {
			return false
		}
	}
	return true
}

// splitRow splits a pipe-delimited row into trimmed cells.
func splitRow(line string) []string {
	line = strings.TrimSpace(line)
	line = strings.TrimPrefix(line, "|")
	line = strings.TrimSuffix(line, "|")

	parts := strings.Split(line, "|")
	cells := make([]string, len(parts))
	for i, part := range parts {
		cells[i] = strings.TrimSpace(part)
	}
	return cells
}

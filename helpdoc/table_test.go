package helpdoc

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/titpetric/atkins/colors"
)

// TestTable checks a table is padded to the widest cell and carries the
// GFM rule row.
func TestTable(t *testing.T) {
	var out bytes.Buffer

	table := newTable(&out, "Flag", "What it does")
	table.row("--list", "List jobs")
	table.row("--working-directory", "Change directory")
	table.flush()

	lines := strings.Split(strings.TrimRight(out.String(), "\n"), "\n")
	assert.Len(t, lines, 4)
	assert.Equal(t, "| Flag                | What it does     |", lines[0])
	assert.Equal(t, "|---------------------|------------------|", lines[1])
	assert.Equal(t, "| --list              | List jobs        |", lines[2])
	assert.Equal(t, "| --working-directory | Change directory |", lines[3])
}

// TestTableEmpty checks a table with no rows writes nothing, so a
// section with nothing to report stays out of the document.
func TestTableEmpty(t *testing.T) {
	var out bytes.Buffer

	newTable(&out, "Flag", "What it does").flush()

	assert.Empty(t, out.String())
}

// TestTableShortRow checks a row with fewer cells than headers is padded
// rather than producing a table markdown cannot parse.
func TestTableShortRow(t *testing.T) {
	var out bytes.Buffer

	table := newTable(&out, "Job", "Default", "What it does")
	table.row("build")
	table.flush()

	for _, line := range strings.Split(strings.TrimRight(out.String(), "\n"), "\n") {
		assert.Equal(t, 4, strings.Count(line, "|"), line)
	}
}

// TestTableColoredCell checks padding measures the visible width, so a
// coloured cell does not push its column out.
func TestTableColoredCell(t *testing.T) {
	var out bytes.Buffer

	table := newTable(&out, "Job", "What it does")
	table.row(colors.BrightGreen("build"), "Build app")
	table.flush()

	lines := strings.Split(strings.TrimRight(out.String(), "\n"), "\n")
	assert.Equal(t, colors.VisualLength(lines[0]), colors.VisualLength(lines[2]))
}

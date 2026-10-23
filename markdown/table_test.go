package markdown

import (
	"strings"
	"testing"

	"github.com/titpetric/atkins/colors"
	"github.com/titpetric/atkins/internal/assert"
)

const sampleTable = "| Flag      | What it does     |\n" +
	"|-----------|-------------------|\n" +
	"| `--list`  | List jobs         |\n" +
	"| `--debug` | Print debug data  |"

func TestIsTable(t *testing.T) {
	assert.True(t, IsTable(sampleTable))
	assert.False(t, IsTable("just a paragraph, no pipes here"))
	assert.False(t, IsTable("| only a header |"))
}

func TestParseTable(t *testing.T) {
	table, ok := ParseTable(sampleTable)
	assert.True(t, ok)

	assert.Equal(t, []string{"Flag", "What it does"}, table.Headers)
	assert.Equal(t, [][]string{
		{"`--list`", "List jobs"},
		{"`--debug`", "Print debug data"},
	}, table.Rows)
}

// TestParseTableRejectsProse checks a section that merely starts with a
// pipe-looking line, but has no separator row, is not read as a table.
func TestParseTableRejectsProse(t *testing.T) {
	_, ok := ParseTable("| not a table |\njust a second line")
	assert.False(t, ok)
}

func TestTable(t *testing.T) {
	t.Parallel()

	table := &Table{Headers: []string{"Job", "Runs"}, Rows: [][]string{{"build", "3"}}}
	lines := strings.Split(colors.StripANSI(table.render()), "\n")

	// A border above, the header, a rule, one row and a border below,
	// with every column as wide as its widest cell.
	assert.Equal(t, 5, len(lines))
	assert.Equal(t, "┌───────┬──────┐", lines[0])
	assert.Equal(t, "│ build │ 3    │", lines[3])
}

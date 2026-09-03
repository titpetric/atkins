package markdown

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	require.True(t, ok)

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

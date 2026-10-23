package chart_test

import (
	"strings"
	"testing"

	"github.com/titpetric/atkins/chart"
	"github.com/titpetric/atkins/internal/assert"
)

// lines splits a chart into its rows, dropping the trailing empty one.
func lines(rendered string) []string {
	return strings.Split(strings.TrimRight(rendered, "\n"), "\n")
}

// count returns how many times a rune appears.
func count(text string, r rune) int {
	return strings.Count(text, string(r))
}

func TestFormat(t *testing.T) {
	t.Parallel()

	var format chart.Format = func(v int64) string { return strings.Repeat("x", int(v)) }

	rendered := chart.Horizontal(chart.HorizontalOptions{
		Bars:   []chart.Bar{{Label: "a", Value: 3}},
		Width:  4,
		Format: format,
	})

	assert.True(t, strings.HasSuffix(strings.TrimRight(rendered, "\n"), "  xxx"))
}

func TestBarNegativeCountsAsZero(t *testing.T) {
	t.Parallel()

	rendered := chart.Horizontal(chart.HorizontalOptions{
		Bars:  []chart.Bar{{Label: "up", Value: 4}, {Label: "down", Value: -4}},
		Width: 4,
	})

	rows := lines(rendered)
	assert.Equal(t, 4, count(rows[0], '█'))
	assert.Equal(t, 0, count(rows[1], '█'))
}

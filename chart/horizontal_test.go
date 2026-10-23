package chart_test

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/titpetric/atkins/chart"
	"github.com/titpetric/atkins/internal/assert"
)

func TestHorizontal(t *testing.T) {
	rendered := chart.Horizontal(chart.HorizontalOptions{
		Bars: []chart.Bar{
			{Label: "go", Value: 100},
			{Label: "npm", Value: 50},
			{Label: "docker", Value: 0},
		},
		Width:  8,
		Format: func(v int64) string { return strconv.FormatInt(v, 10) },
	})

	rows := lines(rendered)
	assert.Len(t, rows, 3)

	// Every row carries its name, because a reader of a ranking reads
	// the names.
	assert.Contains(t, rows[0], "go")
	assert.Contains(t, rows[1], "npm")
	assert.Contains(t, rows[2], "docker")

	// The largest fills the width, the half one half of it, and the
	// empty one none.
	assert.Equal(t, 8, count(rows[0], '█'))
	assert.Equal(t, 4, count(rows[1], '█'))
	assert.Equal(t, 0, count(rows[2], '█'))

	// Padding to the widest label starts every bar in one column
	// whatever its name is.
	assert.Equal(t, strings.Index(rows[0], "█"), strings.Index(rows[1], "█"))

	// Values are right-aligned to the widest, so they read as a column.
	assert.Contains(t, rows[0], "100")
	assert.Contains(t, rows[1], " 50")
}

func TestHorizontalNotes(t *testing.T) {
	rendered := chart.Horizontal(chart.HorizontalOptions{
		Bars:   []chart.Bar{{Label: "go", Value: 100}, {Label: "npm", Value: 50}},
		Format: func(v int64) string { return strconv.FormatInt(v, 10) },
		Note: func(bar chart.Bar) string {
			if bar.Label == "npm" {
				return ""
			}
			return fmt.Sprintf("%s runs", bar.Label)
		},
	})

	rows := lines(rendered)
	assert.Contains(t, rows[0], "go runs")
	// An empty note adds no trailing separator.
	assert.Equal(t, strings.TrimRight(rows[1], " "), rows[1])
}

func TestHorizontalWithNothingToDraw(t *testing.T) {
	assert.Equal(t, "", chart.Horizontal(chart.HorizontalOptions{}))

	// Every bar at zero draws the labels and no bars.
	rendered := chart.Horizontal(chart.HorizontalOptions{
		Bars: []chart.Bar{{Label: "go", Value: 0}},
	})
	assert.Contains(t, rendered, "go")
	assert.NotContains(t, rendered, "█")
}

func TestHorizontalWithoutAFormat(t *testing.T) {
	rendered := chart.Horizontal(chart.HorizontalOptions{
		Bars:  []chart.Bar{{Label: "go", Value: 100}},
		Width: 4,
	})

	// No format, no values: the bars alone.
	assert.Contains(t, rendered, "████")
	assert.NotContains(t, rendered, "100")
}

func TestHorizontalColor(t *testing.T) {
	bars := []chart.Bar{{Label: "go", Value: 100}}

	plain := chart.Horizontal(chart.HorizontalOptions{Bars: bars})
	assert.NotContains(t, plain, "\x1b[")

	colored := chart.Horizontal(chart.HorizontalOptions{Bars: bars, Color: true})
	assert.Contains(t, colored, "\x1b[")
}

// A label with characters wider than a byte keeps the column aligned.
func TestHorizontalAlignsRunesNotBytes(t *testing.T) {
	rendered := chart.Horizontal(chart.HorizontalOptions{
		Bars:  []chart.Bar{{Label: "gö", Value: 1}, {Label: "npm", Value: 1}},
		Width: 4,
	})

	rows := lines(rendered)
	assert.Equal(t, strings.Index(rows[0], "█"), strings.Index(rows[1], "█")+1)
}

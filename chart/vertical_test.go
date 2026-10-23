package chart_test

import (
	"strconv"
	"testing"

	"github.com/titpetric/atkins/chart"
	"github.com/titpetric/atkins/internal/assert"
)

func TestVertical(t *testing.T) {
	rendered := chart.Vertical(chart.VerticalOptions{
		Bars: []chart.Bar{
			{Label: "Mon", Value: 0},
			{Label: "Tue", Value: 4},
			{Label: "Wed", Value: 8},
		},
		Height: 4,
		Format: func(v int64) string { return strconv.FormatInt(v, 10) },
	})

	rows := lines(rendered)
	// Four rows of chart, the floor, and the dates under it.
	assert.Len(t, rows, 6)

	// The peak is the only value labelled, and it sits at the top.
	assert.Contains(t, rows[0], "8")
	assert.NotContains(t, rows[1], "8")

	// The tallest bar reaches the top row alone; the half one joins it
	// halfway down, and the empty one is never drawn.
	assert.Equal(t, 1, count(rows[0], '█'))
	assert.Equal(t, 1, count(rows[1], '█'))
	assert.Equal(t, 2, count(rows[2], '█'))
	assert.Equal(t, 2, count(rows[3], '█'))

	// The floor and the dates.
	assert.Contains(t, rows[4], "└")
	assert.Contains(t, rows[5], "Mon")
	assert.Contains(t, rows[5], "Wed")
}

// A bar measures from zero, so an empty day stays empty and a day with
// anything in it shows: a blank column would say nothing ran.
func TestVerticalDrawsTheSmallestValue(t *testing.T) {
	rendered := chart.Vertical(chart.VerticalOptions{
		Bars:   []chart.Bar{{Value: 1}, {Value: 0}, {Value: 10_000}},
		Height: 3,
	})

	bottom := lines(rendered)[2]
	assert.Contains(t, bottom, "▁")
	assert.Contains(t, bottom, "█")
	assert.Contains(t, bottom, " ")
}

func TestVerticalWithNothingToDraw(t *testing.T) {
	assert.Equal(t, "", chart.Vertical(chart.VerticalOptions{}))

	// Every bar at zero draws a floor and no bars, and never divides by
	// a peak of nothing.
	rendered := chart.Vertical(chart.VerticalOptions{Bars: []chart.Bar{{Value: 0}, {Value: 0}}})
	assert.NotContains(t, rendered, "█")
	assert.Contains(t, rendered, "└")
}

func TestVerticalLabels(t *testing.T) {
	// One bar gets its own label and no spacing to a second.
	one := chart.Vertical(chart.VerticalOptions{Bars: []chart.Bar{{Label: "Mon", Value: 1}}})
	assert.Contains(t, one, "Mon")

	// Unlabelled bars draw no axis line at all.
	bare := chart.Vertical(chart.VerticalOptions{Bars: []chart.Bar{{Value: 1}, {Value: 2}}})
	assert.Len(t, lines(bare), 7)

	// Labels wider than the chart keep one space between them rather
	// than running together.
	tight := chart.Vertical(chart.VerticalOptions{
		Bars: []chart.Bar{{Label: "2026-10-21", Value: 1}, {Label: "2026-10-22", Value: 2}},
	})
	assert.Contains(t, tight, "2026-10-21 2026-10-22")
}

func TestVerticalColor(t *testing.T) {
	bars := []chart.Bar{{Label: "Mon", Value: 1}, {Label: "Tue", Value: 2}}

	plain := chart.Vertical(chart.VerticalOptions{Bars: bars})
	assert.NotContains(t, plain, "\x1b[")

	colored := chart.Vertical(chart.VerticalOptions{Bars: bars, Color: true})
	assert.Contains(t, colored, "\x1b[")
}

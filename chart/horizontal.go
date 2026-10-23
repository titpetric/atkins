package chart

import (
	"strings"
)

// HorizontalOptions is a chart of named totals against each other, one
// row each.
type HorizontalOptions struct {
	// Bars are the rows, drawn in the order given.
	Bars []Bar

	// Width is how many cells the longest bar fills. Twenty-four unless
	// set, which leaves room for a label and a value on an eighty
	// column terminal.
	Width int

	// Format renders the value printed after each bar. A nil Format
	// prints no values.
	Format Format

	// Note renders a second, quieter value after the first, such as the
	// count behind a total. Nil prints none.
	Note func(Bar) string

	// Color draws the bars in colour.
	Color bool
}

// Horizontal returns a row per bar, or an empty string for no bars.
func Horizontal(opts HorizontalOptions) string {
	if len(opts.Bars) == 0 {
		return ""
	}

	width := opts.Width
	if width <= 0 {
		width = 24
	}

	most := peak(opts.Bars)

	label := 0
	value := 0
	for _, bar := range opts.Bars {
		if len(bar.Label) > label {
			label = len(bar.Label)
		}
		if opts.Format != nil {
			if rendered := len(opts.Format(bar.Value)); rendered > value {
				value = rendered
			}
		}
	}

	var out strings.Builder
	for _, bar := range opts.Bars {
		out.WriteString(pad(bar.Label, label))
		out.WriteString("  ")
		out.WriteString(accent(row(bar.Value, most, width), opts.Color))

		if opts.Format != nil {
			out.WriteString("  ")
			out.WriteString(padLeft(opts.Format(bar.Value), value))
		}
		if opts.Note != nil {
			if note := opts.Note(bar); note != "" {
				out.WriteString("  ")
				out.WriteString(dim(note, opts.Color))
			}
		}
		out.WriteString("\n")
	}

	return out.String()
}

// row draws one horizontal bar: full cells, then the eighth nearest the
// leftover, then spaces to the chart's width so the values line up.
func row(value, most int64, width int) string {
	filled := eighths(value, most, width)

	var bar strings.Builder
	for cells := range width {
		bar.WriteRune(cell(cells, filled))
	}

	return bar.String()
}

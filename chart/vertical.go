package chart

import (
	"strings"

	"github.com/titpetric/atkins/colors"
)

// VerticalOptions is a chart of a measure over a run of bars, one
// column each, drawn bottom up.
type VerticalOptions struct {
	// Bars are the columns, drawn in the order given. An empty day is a bar
	// of zero: the gaps make a run of days read as time.
	Bars []Bar

	// Height is how many rows tall the chart is. Six unless set.
	Height int

	// Format renders the peak, the one value with a label. A nil Format
	// labels nothing.
	Format Format

	// Color draws the bars and the axis in colour.
	Color bool
}

// Vertical returns a column per bar, bottom up, or an empty string for no bars.
func Vertical(opts VerticalOptions) string {
	if len(opts.Bars) == 0 {
		return ""
	}

	height := opts.Height
	if height <= 0 {
		height = 6
	}

	most := peak(opts.Bars)
	filled := make([]int, 0, len(opts.Bars))
	for _, bar := range opts.Bars {
		filled = append(filled, eighths(bar.Value, most, height))
	}

	// The axis column is as wide as the peak's label. The bars start
	// where the gutter ends on every row.
	ceiling := ""
	if opts.Format != nil && most > 0 {
		ceiling = opts.Format(most)
	}
	gutter := len(ceiling)

	var out strings.Builder
	for row := height - 1; row >= 0; row-- {
		label := ""
		if row == height-1 {
			label = ceiling
		}
		out.WriteString(padLeft(label, gutter))
		out.WriteString(dim(" │ ", opts.Color))

		var bars strings.Builder
		for _, eighth := range filled {
			bars.WriteRune(cell(row, eighth))
		}
		out.WriteString(accent(bars.String(), opts.Color))
		out.WriteString("\n")
	}

	// The floor, then the two dates under its ends.
	out.WriteString(padLeft("0", gutter))
	out.WriteString(dim(" └"+strings.Repeat("─", len(opts.Bars)+1), opts.Color))
	out.WriteString("\n")

	if ends := axis(opts.Bars); ends != "" {
		out.WriteString(strings.Repeat(" ", gutter+3))
		out.WriteString(dim(ends, opts.Color))
		out.WriteString("\n")
	}

	return out.String()
}

// axis returns the labels under the ends of the chart: the first bar's
// on the left and the last bar's on the right, spaced to sit under
// them. One bar, or no labels at all, gets the left one alone.
func axis(bars []Bar) string {
	first := bars[0].Label
	last := bars[len(bars)-1].Label

	if first == "" && last == "" {
		return ""
	}
	if len(bars) == 1 || first == last {
		return first
	}

	width := len(bars)
	if gap := width - len(first) - len(last); gap > 0 {
		return first + strings.Repeat(" ", gap) + last
	}
	return first + " " + last
}

// dim renders the chart's furniture, the axis and the notes.
func dim(text string, color bool) string {
	if !color {
		return text
	}
	return colors.Dim(text)
}

// accent renders the data itself.
func accent(text string, color bool) string {
	if !color {
		return text
	}
	return colors.BrightCyan(text)
}

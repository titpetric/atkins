package chart

import (
	"strings"
	"unicode/utf8"
)

// blocks are the eighths of a filled cell, from one eighth to a full one.
var blocks = []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

// Bar is one bar: what it measures and what to call it.
type Bar struct {
	// Label names the bar. A vertical chart shows it under the first
	// and last bar only; a horizontal chart shows every one.
	Label string

	// Value is what the bar is long in proportion to. Negative counts
	// as zero, because a bar's length is its number and there is no
	// length below none.
	Value int64
}

// Format renders a value as a reader reads it.
type Format func(int64) string

// peak returns the largest value among the bars, and zero when they are
// all empty.
func peak(bars []Bar) int64 {
	var most int64
	for _, bar := range bars {
		if bar.Value > most {
			most = bar.Value
		}
	}
	return most
}

// eighths returns how many eighths of a cell a value fills, out of a
// height in cells, and at least one for a value above zero.
func eighths(value, most int64, height int) int {
	if value <= 0 || most <= 0 {
		return 0
	}

	filled := int(value * int64(height) * 8 / most)
	if filled < 1 {
		filled = 1
	}
	return filled
}

// cell returns the block for one cell of a bar: the row index counted
// from the bottom, against how many eighths the bar fills.
func cell(row, filled int) rune {
	full := filled / 8
	if row < full {
		return '█'
	}
	if row > full {
		return ' '
	}

	remainder := filled % 8
	if remainder == 0 {
		return ' '
	}
	return blocks[remainder-1]
}

// pad returns a string padded to a width in runes with spaces on the
// right. A command of any runes keeps the next column aligned.
func pad(text string, width int) string {
	if gap := width - utf8.RuneCountInString(text); gap > 0 {
		return text + strings.Repeat(" ", gap)
	}
	return text
}

// padLeft returns a string padded to a width with spaces on the left.
func padLeft(text string, width int) string {
	if gap := width - utf8.RuneCountInString(text); gap > 0 {
		return strings.Repeat(" ", gap) + text
	}
	return text
}

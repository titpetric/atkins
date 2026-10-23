// Package chart draws small bar charts for a terminal.
//
// `atkins --stats` shows two shapes: a measure over a run of days, and a
// handful of named totals against each other. Vertical draws the first
// and Horizontal the second. Both use block characters, so they need no
// terminal capability beyond the ones a box-drawing table relies on, and
// they read the same in a pipe as on a screen.
//
// The package draws values and leaves their meaning to the caller. The
// caller supplies the values and the Format that renders one, so the same
// chart serves milliseconds, counts and bytes.
//
// Every bar measures from zero against the largest, so a chart never
// exaggerates a difference by cutting its axis. A value above zero fills
// at least one eighth of a cell: a day holding one short command is a
// day something happened, and a blank column would say it did not. A bar
// taller than a row fills whole cells and ends in the eighth nearest its
// leftover, so a six-row chart has the resolution of forty-eight.
//
// A vertical chart labels its peak at the top of the axis and nothing
// else, because a reader of a month of days reads the shape and the
// ceiling, and a number on every column hides both. The first and last
// labels sit under the ends of the floor, the place for a date range.
//
// A horizontal chart carries a label and a value on every row, because a
// reader of a ranking reads the names: the bars show how far ahead the
// first one is, the thing a column of numbers hides. The caller sorts the
// rows; the chart draws them in the order given.
package chart

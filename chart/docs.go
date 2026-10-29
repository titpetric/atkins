// Package chart draws small bar charts for a terminal.
//
// `atkins --stats` shows two shapes: a measure over a run of days, and a
// handful of named totals side by side. Vertical draws the first and
// Horizontal the second. Both use block characters. They need the same
// terminal support a box-drawing table needs, and they read the same in a
// pipe and on a screen.
//
// The package draws values and leaves their meaning to the caller. The
// caller supplies the values and the Format that renders one. One chart
// serves milliseconds, counts and bytes.
//
// Every bar measures from zero against the largest value, and the axis
// always starts at zero. A value above zero fills at least one eighth of
// a cell: a day holding one short command is a day something happened,
// and its column shows a mark. A bar taller than a row fills whole cells
// and ends in the eighth nearest its leftover. A six-row chart has the
// resolution of forty-eight.
//
// A vertical chart labels its peak at the top of the axis. A reader of a
// month of days reads the shape and the ceiling, and the one label keeps
// both visible. The first and last labels sit under the ends of the
// floor, the place for a date range.
//
// A horizontal chart carries a label and a value on every row. A reader
// of a ranking reads the names, and the bars show how far ahead the first
// one is. The caller sorts the rows; the chart draws them in the order
// given.
package chart

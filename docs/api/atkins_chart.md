# Package ./chart

```go
import (
	"github.com/titpetric/atkins/chart"
)
```

Package `chart` draws small bar charts for a terminal.

`atkins --stats` shows two shapes: a measure over a run of days, and a
handful of named totals side by side. Vertical draws the first and
Horizontal the second. Both use block characters. They need the same
terminal support a box-drawing table needs, and they read the same in a
pipe and on a screen.

The package draws values and leaves their meaning to the caller. The
caller supplies the values and the Format that renders one. One chart
serves milliseconds, counts and bytes.

Every bar measures from zero against the largest value, and the axis
always starts at zero. A value above zero fills at least one eighth of
a cell: a day holding one short command is a day something happened,
and its column shows a mark. A bar taller than a row fills whole cells
and ends in the eighth nearest its leftover. A six-row chart has the
resolution of forty-eight.

A vertical chart labels its peak at the top of the axis. A reader of a
month of days reads the shape and the ceiling, and the one label keeps
both visible. The first and last labels sit under the ends of the
floor, the place for a date range.

A horizontal chart carries a label and a value on every row. A reader
of a ranking reads the names, and the bars show how far ahead the first
one is. The caller sorts the rows; the chart draws them in the order
given.

## Types

<details>
<summary><code>type Bar</code></summary>

```go
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
```

</details>

<details>
<summary><code>type Format</code></summary>

```go
// Format renders a value as a reader reads it.
type Format func(int64) string
```

</details>

<details>
<summary><code>type HorizontalOptions</code></summary>

```go
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
```

</details>

<details>
<summary><code>type VerticalOptions</code></summary>

```go
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
```

</details>

## Function symbols

- `func Horizontal (opts HorizontalOptions) string`
- `func Vertical (opts VerticalOptions) string`

### `Horizontal`

`Horizontal` returns a row per bar, or an empty string for no bars.

```go
func Horizontal(opts HorizontalOptions) string
```

### `Vertical`

`Vertical` returns a column per bar, bottom up, or an empty string for no bars.

```go
func Vertical(opts VerticalOptions) string
```

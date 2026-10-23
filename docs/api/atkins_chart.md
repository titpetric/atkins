# Package ./chart

```go
import (
	"github.com/titpetric/atkins/chart"
)
```

Package `chart` draws small bar charts for a terminal.

It exists for `atkins --stats`, which has two shapes to show: a
measure over a run of days, and a handful of named totals against each
other. Both are drawn with block characters, so they need no terminal
capability beyond the ones a box-drawing table already assumes, and
they read the same in a pipe as on a screen.

Nothing here knows what it is drawing. The caller supplies values and
the function that renders one, so the same chart serves milliseconds,
counts and bytes.

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
	// Bars are the rows, in the order they are drawn. The caller sorts
	// them: a ranking is the caller's question, not the chart's.
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
	// Bars are the columns, in the order they are drawn. A day with
	// nothing in it is a bar of zero, not a missing bar: the gaps are
	// what makes a run of days read as time.
	Bars []Bar

	// Height is how many rows tall the chart is. Six unless set.
	Height int

	// Format renders the peak, which is the only value labelled. A nil
	// Format labels nothing.
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

`Horizontal` draws a row per bar, measured from zero against the
largest.

Every row carries its label and its value, because a reader of a
ranking is reading the names: the bars are there to show how far
ahead the first one is, which is the thing a column of numbers hides.

```go
func Horizontal(opts HorizontalOptions) string
```

### `Vertical`

`Vertical` draws a column per bar, bottom up, measured from zero.

The peak sits at the top of the axis and nothing else is labelled: a
reader of a month of days wants the shape and the ceiling, and a
number on every column hides both. The first and last labels sit
under the ends, which is where a date range belongs.

```go
func Vertical(opts VerticalOptions) string
```

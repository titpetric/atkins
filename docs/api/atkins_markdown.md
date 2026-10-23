# Package ./markdown

```go
import (
	"github.com/titpetric/atkins/markdown"
)
```

Package `markdown` finds structure in a rendered markdown document without
a full parser: the blank-line-separated blocks a document is already
written in, and the GFM pipe tables among them.

It serves one caller: `atkins --help` builds one long markdown
document, and a table in it reads better in a terminal as a bordered,
coloured grid than as pipe-delimited text. Sections splits the document
so Render can find and replace a table block without disturbing the
prose around it. A fenced code block (a line starting with ``` opens
and closes it) stays one section even when it holds a blank line, so a
YAML example never splits mid-fence.

Render redraws every pipe table as a bordered ANSI table, colours every
ATX heading the caller left plain, and leaves everything else alone.
With colour off it returns the document byte for byte, so
`atkins --help > help.md` or a piped `atkins --help | less` still gets
plain markdown a table-aware reader can parse.

## Types

<details>
<summary><code>type Table</code></summary>

```go
// Table is a GFM pipe table decoded from a markdown section: a header
// row, and the data rows below the `---` separator.
type Table struct {
	Headers []string
	Rows    [][]string
}
```

</details>

## Function symbols

- `func IsTable (section string) bool`
- `func ParseTable (section string) (*Table, bool)`
- `func Render (doc string, color bool) string`
- `func Sections (doc string) []string`

### `IsTable`

`IsTable` reports whether a section is a GFM pipe table: a row of cells
followed by a separator row of dashes, colons and spaces only.

```go
func IsTable(section string) bool
```

### `ParseTable`

`ParseTable` returns ok=false for a section IsTable rejects, and the
decoded table otherwise.

```go
func ParseTable(section string) (*Table, bool)
```

### `Render`

`Render` returns doc unchanged without color, and with its tables and
headings drawn in ANSI otherwise.

```go
func Render(doc string, color bool) string
```

### `Sections`

`Sections` splits a document into the blocks a blank line separates, keeping fences whole.

```go
func Sections(doc string) []string
```

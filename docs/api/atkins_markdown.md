# Package ./markdown

```go
import (
	"github.com/titpetric/atkins/markdown"
)
```

Package `markdown` finds structure in a rendered markdown document without
a full parser: the blank-line-separated blocks a document is already
written in, and the GFM pipe tables among them.

It exists for one caller: `atkins --help` builds one long markdown
document, and a table in it reads better in a terminal as a bordered,
coloured grid than as pipe-delimited text. Sections splits the document
so Render can find and replace a table block without disturbing the
prose around it.

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

`ParseTable` decodes a section as a table. It returns ok=false for a
section IsTable rejects, so a caller can try it on every section
without checking first.

```go
func ParseTable(section string) (*Table, bool)
```

### `Render`

`Render` redraws every GFM pipe table in a markdown document as a
bordered ANSI table, colours every ATX heading the caller left plain,
and leaves everything else untouched.

With color false Render returns the document byte for byte,
so `atkins --help > help.md` or a piped `atkins --help | less` still
gets plain markdown a table-aware reader can parse.

```go
func Render(doc string, color bool) string
```

### `Sections`

`Sections` splits a document into the blocks a blank line separates.

A fenced code block (a line starting with ``` opens and closes it) stays
one section even when it contains a blank line, so Sections never splits
a YAML example inside `atkins --help` mid-fence.

```go
func Sections(doc string) []string
```

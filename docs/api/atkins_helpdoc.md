# Package ./helpdoc

```go
import (
	"github.com/titpetric/atkins/helpdoc"
)
```

Package `helpdoc` renders `atkins --help`.

The help reads as a document: the usage line, the commands, every flag
with its default, the skills installed on this machine with the jobs
each contributes, and a reference for the atkins.yml schema. A person
or an agent reading it can run atkins and write a pipeline without
opening a second page. The section order lets a reader stop early: what
to type, then the flags, then the jobs the installed skills add, then
the schema for writing a pipeline of your own.

Write builds the document as plain markdown throughout, with padded
table cells so it reads as columns in a file too. Write adds colour -
on the headings, and on every table, which the markdown package redraws
with borders - only when the caller says the writer is a terminal.
`atkins --help > help.md` writes a plain markdown document.

A person writes schema.md by hand; no generator touches it. It reduces
the JSON Schema `splint -schema -i model` prints: the keys an author
types, in the spellings the model's UnmarshalYAML methods accept,
without the Go field names and the display-only fields a pipeline file
cannot set. Regenerating it from the schema would lose that.
TestSchemaDocumentsModelKeys keeps it honest: it reads the yaml tags off
model.Pipeline, model.Job and model.Step and fails when schema.md
documents a key none of them declares.

## Types

<details>
<summary><code>type Command</code></summary>

```go
// Command is one subcommand in the help document: the word a user types,
// what it does, and the flags it defines.
type Command struct {
	// Name is the word on the command line, e.g. "server".
	Name string

	// Title is one line on what the command does.
	Title string

	// Default marks the command that runs when the user names none. Write
	// prints its flags as the global ones.
	Default bool

	// Flags are the command's own flags, nil when it has none.
	Flags *cli.FlagSet
}
```

</details>

<details>
<summary><code>type Options</code></summary>

```go
// Options is what Write renders.
type Options struct {
	// Name is the executable name in the usage line.
	Name string

	// Version prints under the usage line, empty to leave it out.
	Version string

	// Commands are the subcommands, the default one first.
	Commands []Command

	// Skills are the skill files found on disk, in precedence order.
	// Write deduplicates them by ID and reports the ones the current
	// directory does not activate.
	Skills []*runner.Skill

	// SkillDirs are the directories the skills come from, in
	// precedence order, printed so a reader knows where to add one.
	SkillDirs []string

	// Color turns on ANSI colour for headings. Callers set it from a
	// terminal check on the writer. A redirected help document is
	// plain markdown.
	Color bool
}
```

</details>

## Function symbols

- `func Schema () string`
- `func Write (w io.Writer, opts *Options) error`

### `Schema`

`Schema` returns the atkins.yml reference as markdown, without a heading.

```go
func Schema() string
```

### `Write`

`Write` renders the whole help document to w.

```go
func Write(w io.Writer, opts *Options) error
```

# Package ./helpdoc

```go
import (
	"github.com/titpetric/atkins/helpdoc"
}
```

Package helpdoc renders `atkins --help`.

The help is a document rather than a flag dump: the usage line, the
commands, every flag with its default, the skills installed on this
machine with the jobs each contributes, and a reference for the
atkins.yml schema. Someone reading it, or an agent, should be able to
run atkins and write a pipeline without opening a second page.

The document is built as plain markdown throughout, with padded table
cells so it reads as columns in a file too. Colour - on the headings,
and on every table, redrawn with borders by the markdown package - is
added only when the caller says the writer is a terminal, so
`atkins --help > help.md` is a document and not a screenshot of one.

## Types

```go
// Command is one subcommand in the help document: the word a user types,
// what it does, and the flags it defines.
type Command struct {
	// Name is the word on the command line, e.g. "server".
	Name string

	// Title is one line on what the command does.
	Title string

	// Default marks the command that runs when none is named. Its flags
	// are printed as the global ones.
	Default bool

	// Flags are the command's own flags, nil when it has none.
	Flags *cli.FlagSet
}
```

```go
// Options is what Write renders.
type Options struct {
	// Name is the executable name in the usage line.
	Name string

	// Version is printed under the usage line, empty to leave it out.
	Version string

	// Commands are the subcommands, the default one first.
	Commands []Command

	// Skills are the skill files found on disk, in precedence order.
	// Write deduplicates them by ID and reports the ones the current
	// directory does not activate.
	Skills []*runner.Skill

	// SkillDirs are the directories the skills were read from, in
	// precedence order, printed so a reader knows where to add one.
	SkillDirs []string

	// Color turns on ANSI colour for headings. Callers set it from a
	// terminal check on the writer, so a redirected help document is
	// plain markdown.
	Color bool
}
```

## Function symbols

- `func Schema () string`
- `func Write (w io.Writer, opts *Options) error`

### Schema

Schema returns the atkins.yml reference as markdown, without a heading
of its own.

```go
func Schema() string
```

### Write

Write renders the whole help document.

The sections are ordered so a reader can stop early: what to type,
then the flags, then what the installed skills offer, then the schema
for writing a pipeline of your own.

```go
func Write(w io.Writer, opts *Options) error
```

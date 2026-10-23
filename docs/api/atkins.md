# Package atkins

```go
import (
	"github.com/titpetric/atkins"
)
```

## Types

<details>
<summary><code>type Options</code></summary>

```go
// Options holds pipeline command-line arguments
type Options struct {
	File             string
	Jobs             []string
	Args             []string
	List             bool
	All              bool
	Lint             bool
	Debug            bool
	LogFile          string
	FinalOnly        bool
	WorkingDirectory string
	Jail             bool
	JSON             bool
	YAML             bool
	Version          bool

	// Stats reports the commands recorded in the event database by the
	// time they have taken, and runs no pipeline. Local records the run
	// nowhere.
	Stats bool
	Local bool

	// Config opens the configuration menu for .atkins/config.yml.
	Config bool

	// Vendor copies the skills this repository uses into .atkins/skills.
	// It reports the selection and writes nothing unless Write is true.
	Vendor bool
	Write  bool

	FlagSet *cli.FlagSet
}
```

</details>

## Vars

<details>
<summary><code>var Version, Commit, CommitTime, Branch</code></summary>

```go
// Version information injected at build time via ldflags
var (
	Version    = "dev"
	Commit     = "unknown"
	CommitTime = "unknown"
	Branch     = "unknown"
)
```

</details>

## Function symbols

- `func HelpFlags () *pflag.FlagSet`
- `func HelpRequested (flags *pflag.FlagSet, args []string) bool`
- `func NewOptions () *Options`
- `func Pipeline () *cli.Command`
- `func WriteHelp (w io.Writer) error`
- `func (*Options) Bind (fs *cli.FlagSet)`

### `HelpFlags`

`HelpFlags` returns one flag set carrying every flag atkins defines.
Help detection has to tell which of them take a value.

```go
func HelpFlags() *pflag.FlagSet
```

### `HelpRequested`

`HelpRequested` reports whether a command line requests the help
document and no run.

A -h or --help that is the value of a string flag is not a help
request: `atkins -x "--help"` runs a prompt. The flag set says which
flags take a value, so the same rules tell the two cases apart that
pflag parses by. Everything after a bare -- is an argument.

```go
func HelpRequested(flags *pflag.FlagSet, args []string) bool
```

### `Pipeline`

`Pipeline` returns a cli.Command that runs the atkins command pipeline.

```go
func Pipeline() *cli.Command
```

### `WriteHelp`

`WriteHelp` renders `atkins --help` to a writer.

WriteHelp scans the skills and loads none: the document lists every
installed skill, marking what this directory does not activate, because a
reader asking what atkins can do is not always standing in a
repository that answers.

```go
func WriteHelp(w io.Writer) error
```

### `NewOptions`

```go
func NewOptions() *Options
```

### `Bind`

```go
func (*Options) Bind(fs *cli.FlagSet)
```

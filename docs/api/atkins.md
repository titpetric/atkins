# Package atkins

```go
import (
	"github.com/titpetric/atkins"
)
```

## Types

<details>
<summary><code>type CommandReport</code></summary>

```go
// CommandReport is one command key and what it has cost. The key is
// the command's words up to its first flag, at most three and no path
// or value, as storage.Key names it: `docker compose up`, `go test`.
type CommandReport struct {
	Command   string     `json:"command" yaml:"command"`
	Tool      string     `json:"tool" yaml:"tool"`
	Runs      int64      `json:"runs" yaml:"runs"`
	TotalMs   int64      `json:"total_ms" yaml:"total_ms"`
	AverageMs int64      `json:"average_ms" yaml:"average_ms"`
	MaxMs     int64      `json:"max_ms" yaml:"max_ms"`
	Failures  int64      `json:"failures" yaml:"failures"`
	LastRunAt *time.Time `json:"last_run_at,omitempty" yaml:"last_run_at,omitempty"`
}
```

</details>

<details>
<summary><code>type CommandUsageReport</code></summary>

```go
// CommandUsageReport is one command key on one day.
type CommandUsageReport struct {
	Date    string `json:"date" yaml:"date"`
	Command string `json:"command" yaml:"command"`
	Tool    string `json:"tool" yaml:"tool"`
	Runs    int64  `json:"runs" yaml:"runs"`
	TotalMs int64  `json:"total_ms" yaml:"total_ms"`
}
```

</details>

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

	// Limit keeps the newest commands --stats --all lists. Zero keeps
	// every command of the window.
	Limit int

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

<details>
<summary><code>type SampleReport</code></summary>

```go
// SampleReport is one recorded command. It names the command by its
// key and never by its command line, because a command line carries
// its values and a value can be a token.
type SampleReport struct {
	// At is when the command started, in UTC to the millisecond.
	At string `json:"at" yaml:"at"`

	Command    string `json:"command" yaml:"command"`
	Tool       string `json:"tool" yaml:"tool"`
	Kind       string `json:"kind" yaml:"kind"`
	DurationMs int64  `json:"duration_ms" yaml:"duration_ms"`
	ExitCode   int64  `json:"exit_code" yaml:"exit_code"`
	Failed     bool   `json:"failed" yaml:"failed"`

	// Pipeline and Job are what ran it, and InvocationID the atkins
	// process, shared by every command of one run.
	Pipeline     string `json:"pipeline,omitempty" yaml:"pipeline,omitempty"`
	Job          string `json:"job,omitempty" yaml:"job,omitempty"`
	InvocationID string `json:"invocation_id" yaml:"invocation_id"`
}
```

</details>

<details>
<summary><code>type StatsReport</code></summary>

```go
// StatsReport is everything `atkins --stats` has to say: what this
// checkout has been running, and what every checkout on the machine has
// been running between them.
//
// A run records to both databases, so the two answer different
// questions of the same rows. The project's says what standing here
// costs; the workspace's says where a week went across every project
// atkins ran in.
type StatsReport struct {
	// Project is the checkout's own history, absent for a directory
	// that belongs to no project and for one whose database is the
	// workspace's.
	Project *StatsScope `json:"project,omitempty" yaml:"project,omitempty"`

	// Workspace is every project's history together.
	Workspace *StatsScope `json:"workspace,omitempty" yaml:"workspace,omitempty"`
}
```

</details>

<details>
<summary><code>type StatsScope</code></summary>

```go
// StatsScope is one database's history, and everything a reader of
// `--stats --json` needs to draw it again elsewhere.
//
// Durations are milliseconds throughout, as the database keeps them,
// because a reader charting them wants a number and not a word.
type StatsScope struct {
	// Name is what the scope is called, project or workspace.
	Name string `json:"name" yaml:"name"`

	// Repository is the checkout the rows came from, as its remote
	// names it; empty for the workspace, which holds many, and for a
	// project with no remote.
	Repository string `json:"repository,omitempty" yaml:"repository,omitempty"`

	// Database is the file the rows came from, and Days the window
	// they cover, which is the retention window.
	Database string `json:"database" yaml:"database"`
	Days     int    `json:"days" yaml:"days"`

	// From and To bound the window, as dates.
	From string `json:"from" yaml:"from"`
	To   string `json:"to" yaml:"to"`

	// Runs and TotalMs are the whole window: how many commands ran and
	// how long they took between them.
	Runs    int64 `json:"runs" yaml:"runs"`
	TotalMs int64 `json:"total_ms" yaml:"total_ms"`

	// Commands are the command keys by total time.
	Commands []CommandReport `json:"commands" yaml:"commands"`

	// Tools are the tools by total time, each a roll-up of its
	// commands.
	Tools []ToolReport `json:"tools" yaml:"tools"`

	// Usage is a row per day per tool, oldest first: the series a chart
	// of tool use over time is drawn from. A day a tool did not run has
	// no row.
	Usage []UsageReport `json:"usage" yaml:"usage"`

	// CommandUsage is a row per day per command key, oldest first: the
	// series a chart stacks by command.
	CommandUsage []CommandUsageReport `json:"command_usage" yaml:"command_usage"`

	// Samples is every command of the window one by one, oldest first,
	// with --all: the rows a reader buckets finer than a day.
	// --limit keeps the newest.
	Samples []SampleReport `json:"samples,omitempty" yaml:"samples,omitempty"`
}
```

</details>

<details>
<summary><code>type ToolReport</code></summary>

```go
// ToolReport is one tool and what it has cost, across every form of it.
type ToolReport struct {
	Tool     string `json:"tool" yaml:"tool"`
	Commands int64  `json:"commands" yaml:"commands"`
	Runs     int64  `json:"runs" yaml:"runs"`
	TotalMs  int64  `json:"total_ms" yaml:"total_ms"`
	Failures int64  `json:"failures" yaml:"failures"`
}
```

</details>

<details>
<summary><code>type UsageReport</code></summary>

```go
// UsageReport is one tool on one day.
type UsageReport struct {
	Date    string `json:"date" yaml:"date"`
	Tool    string `json:"tool" yaml:"tool"`
	Runs    int64  `json:"runs" yaml:"runs"`
	TotalMs int64  `json:"total_ms" yaml:"total_ms"`
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
- `func (*StatsReport) Scopes () []*StatsScope`

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

### `Scopes`

`Scopes` returns the scopes to report, the project first.

```go
func (*StatsReport) Scopes() []*StatsScope
```

### `NewOptions`

```go
func NewOptions() *Options
```

### `Bind`

```go
func (*Options) Bind(fs *cli.FlagSet)
```

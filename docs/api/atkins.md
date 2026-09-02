# Package atkins

```go
import (
	"github.com/titpetric/atkins"
}
```

## Types

<details>
<summary><code>type Options</code></summary>

```go
// Options holds pipeline command-line arguments
type Options struct {
	File             string
	Jobs             []string
	List             bool
	Lint             bool
	Debug            bool
	LogFile          string
	FinalOnly        bool
	WorkingDirectory string
	Jail             bool
	JSON             bool
	YAML             bool
	Version          bool
	Agent            bool
	Exec             string

	// CI/CD client flags. Login and Register take the server URL;
	// Logout applies to the server last logged in to. Dispatch hands
	// the run to an agent instead of running it here, and Local runs
	// here without recording anything on the server.
	Login    string
	Register string
	Logout   bool
	Dispatch bool
	Local    bool

	// Config opens the configuration menu for .atkins/config.yml.
	Config bool

	// Vendor copies the skills this repository uses into .atkins/skills.
	// It reports the selection and writes nothing unless Write is set.
	Vendor bool
	Write  bool

	FlagSet *cli.FlagSet
}
```

</details>

<details>
<summary><code>type ServerOptions</code></summary>

```go
// ServerOptions holds `atkins server` command-line arguments.
// Every flag overrides the resolved configuration; an empty one leaves
// the configured value alone.
type ServerOptions struct {
	// Addr is the listen address.
	Addr string

	// Database is the platform DSN, e.g. sqlite://file:atkins.db or
	// mysql://user:pass@tcp(host)/atkins.
	Database string

	// ArtefactDir is where the bytes of job artefacts are stored.
	ArtefactDir string

	// SigningKey signs access tokens. Required.
	SigningKey string

	// AgentToken is the shared secret agents enrol with.
	AgentToken string

	// AllowRegistration keeps /api/user/register open after the first
	// user exists.
	AllowRegistration bool
}
```

</details>

<details>
<summary><code>type WorkerOptions</code></summary>

```go
// WorkerOptions holds `atkins worker` command-line arguments.
// Flags override the resolved configuration, which already carries the
// document values and the ATKINS_* overlay. An empty flag leaves the
// configured value alone, so the three layers stack rather than fight.
type WorkerOptions struct {
	Server   string
	Token    string
	Email    string
	Password string
	AgentID  string
	Labels   string
	DataDir  string
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
- `func Server () *cli.Command`
- `func Worker () *cli.Command`
- `func WriteHelp (w io.Writer) error`
- `func (*Options) Bind (fs *cli.FlagSet)`
- `func (*ServerOptions) Bind (fs *pflag.FlagSet)`
- `func (*WorkerOptions) Bind (fs *pflag.FlagSet)`

### HelpFlags

HelpFlags returns one flag set carrying every flag atkins defines, for
the three commands together. Help detection needs to know which of
them take a value, and the three sets share no names.

```go
func HelpFlags() *pflag.FlagSet
```

### HelpRequested

HelpRequested reports whether a command line is asking for the help
document rather than a run.

A -h or --help that is the value of a string flag is not a help
request: `atkins -x "--help"` runs a prompt. The flag set says which
flags take a value, so the two cases are told apart by the same rules
pflag parses by. Everything after a bare -- is an argument.

```go
func HelpRequested(flags *pflag.FlagSet, args []string) bool
```

### Pipeline

Pipeline provides a cli.Command that runs the atkins command pipeline.

```go
func Pipeline() *cli.Command
```

### Server

Server provides a cli.Command that runs the atkins CI/CD server.

```go
func Server() *cli.Command
```

### Worker

Worker provides a cli.Command that runs the atkins CI/CD agent.

```go
func Worker() *cli.Command
```

### WriteHelp

WriteHelp renders `atkins --help` to a writer.

The skills are scanned rather than loaded: the document lists what is
installed, marking what this directory does not activate, because a
reader asking what atkins can do is not always standing in a
repository that answers.

```go
func WriteHelp(w io.Writer) error
```

### Bind

Bind registers the server flags.

```go
func (*ServerOptions) Bind(fs *pflag.FlagSet)
```

### Bind

Bind registers the worker flags.

```go
func (*WorkerOptions) Bind(fs *pflag.FlagSet)
```

### NewOptions

```go
func NewOptions() *Options
```

### Bind

```go
func (*Options) Bind(fs *cli.FlagSet)
```

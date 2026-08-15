# Package atkins

```go
import (
	"github.com/titpetric/atkins"
}
```

## Types

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
	// Logout applies to the server last logged in to. Local runs here
	// instead of handing the job to the server.
	Login    string
	Register string
	Logout   bool
	Local    bool

	// Config opens the configuration menu for .atkins/config.yml.
	Config bool

	FlagSet *cli.FlagSet
}
```

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

## Vars

```go
// Version information injected at build time via ldflags
var (
	Version    = "dev"
	Commit     = "unknown"
	CommitTime = "unknown"
	Branch     = "unknown"
)
```

## Function symbols

- `func NewOptions () *Options`
- `func Pipeline () *cli.Command`
- `func Server () *cli.Command`
- `func Worker () *cli.Command`
- `func (*Options) Bind (fs *cli.FlagSet)`
- `func (*ServerOptions) Bind (fs *pflag.FlagSet)`
- `func (*WorkerOptions) Bind (fs *pflag.FlagSet)`

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

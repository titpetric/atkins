# Package ./server

```go
import (
	"github.com/titpetric/atkins/server"
)
```

Package `server` is the atkins CI/CD server: a platform.Module that
authenticates atkins clients and records the jobs they dispatch.

The shape of the system is small. `atkins --login https://domain` stores a credential. From then on every atkins run
posts to /api/dispatch with three things: which git repository, which
directory inside its work tree, and the atkins command. The server
writes a job row and returns an ID; atkins runs its own job
dispatch from there, and agents claim queued work over /api/job/claim.

Mount it in a platform server:

```go
svc := platform.New(platform.NewOptions())
svc.Register(server.NewModule(server.NewOptions()))
svc.Start(ctx)
```

## Types

<details>
<summary><code>type Module</code></summary>

```go
// Module implements the platform module contract for the atkins server.
type Module struct {
	platform.UnimplementedModule

	opts *Options
	api  *api.Handlers
	web  *web.Handlers

	jobs      *storage.JobStorage
	artefacts *storage.JobArtefactStorage
	settings  *storage.SettingStorage

	// tracer records the background sweeps. It holds the platform's
	// recorder when the host enabled telemetry, and nil otherwise;
	// every oida entry point tolerates a nil one.
	tracer *oida.Tracer

	// cancel stops the background sweeps: expired agent leases, and
	// retention. done waits for both.
	cancel context.CancelFunc
	done   sync.WaitGroup
}
```

</details>

<details>
<summary><code>type Options</code></summary>

```go
// Options configures the atkins CI/CD server module.
// Values come from .atkins/config.yml with the ATKINS_* overlay already
// applied; see FromConfig. The struct stays plain so a caller embedding
// the module can build one by hand.
type Options struct {
	// SigningKey signs access tokens. Changing it invalidates every
	// issued token, which is the intended way to log everyone out.
	SigningKey string

	// AgentToken is the shared secret an agent presents to
	// /api/agent/enrol. Without one, no agent can join and the queue
	// stays empty of workers.
	AgentToken string

	// Connection is the platform database connection name. Empty
	// selects storage.ConnectionName ("atkins", from
	// PLATFORM_DB_ATKINS, falling back to PLATFORM_DB_DEFAULT).
	Connection string

	// ArtefactDir is the root the server writes the bytes of job
	// artefacts under. Empty selects DefaultArtefactDir.
	ArtefactDir string

	// TokenTTL is the access token lifetime.
	TokenTTL time.Duration

	// SessionTTL is how long a login lasts before a re-login.
	SessionTTL time.Duration

	// AllowRegistration opens /api/user/register. Registration always
	// stays open while the instance has no human users, so an operator
	// can bootstrap a fresh server with `atkins --register`.
	AllowRegistration bool

	// MaxJobDepth bounds job nesting. A job at the limit cannot
	// dispatch children.
	MaxJobDepth int64

	// LeaseTTL is how long an agent holds a claimed job before the
	// server reclaims it as timed out.
	LeaseTTL time.Duration

	// ReclaimInterval is how often the server sweeps expired leases. Zero
	// disables the sweep.
	ReclaimInterval time.Duration

	// RetentionInterval is how often job.retention and
	// job.log_retention apply. Zero disables the sweep. This is a
	// start-up flag and not a setting because it describes the machine
	// (how much load it will take to keep the tables trimmed), while the
	// windows themselves are policy an admin changes from the API.
	RetentionInterval time.Duration
}
```

</details>

## Consts

<details>
<summary><code>const DefaultArtefactDir</code></summary>

```go
// DefaultArtefactDir is where artefact bytes go when nothing says
// otherwise: beside the database, where the rest of a small instance's
// state already lives.
const DefaultArtefactDir = "artefacts"
```

</details>

<details>
<summary><code>const Name</code></summary>

```go
// Name is the module name, and the name of the database connection the
// module uses (PLATFORM_DB_ATKINS).
const Name = storage.ConnectionName
```

</details>

<details>
<summary><code>const EnvSigningKey, EnvAgentToken</code></summary>

```go
// Environment variable names, kept here because error messages point at
// them. The configuration package owns the actual overlay.
const (
	EnvSigningKey = "ATKINS_SIGNING_KEY"
	EnvAgentToken = "ATKINS_AGENT_TOKEN"
)
```

</details>

## Function symbols

- `func FromConfig (cfg config.ServerConfig) *Options`
- `func NewModule (opts *Options) *Module`
- `func NewOptions () *Options`
- `func (*Module) Mount (_ context.Context, r platform.Router) error`
- `func (*Module) Name () string`
- `func (*Module) Start (ctx context.Context) error`
- `func (*Module) Stop (context.Context) error`

### `FromConfig`

`FromConfig` builds module options from a resolved server configuration.

```go
func FromConfig(cfg config.ServerConfig) *Options
```

### `NewModule`

`NewModule` returns the atkins server module. A nil opts selects
NewOptions, which reads the environment.

```go
func NewModule(opts *Options) *Module
```

### `NewOptions`

`NewOptions` returns options from the embedded default configuration.
It covers a caller that has no resolved configuration.

```go
func NewOptions() *Options
```

### `Mount`

`Mount` registers the API and page routes.

```go
func (*Module) Mount(_ context.Context, r platform.Router) error
```

### `Name`

`Name` returns the module name.

```go
func (*Module) Name() string
```

### `Start`

`Start` connects the database, applies migrations and wires handlers.

A missing signing key is fatal, with no default: a server that
signs tokens with a well-known key is a server anyone can mint an
admin token for.

```go
func (*Module) Start(ctx context.Context) error
```

### `Stop`

`Stop` halts the background sweeps and waits for them to finish.

```go
func (*Module) Stop(context.Context) error
```

# Package ./worker

```go
import (
	"github.com/titpetric/atkins/worker"
)
```

Package `worker` is the atkins CI/CD agent.

An agent claims jobs from a server, reproduces the checkout each job
came from, runs the command, streams the output back and reports the
outcome. This half lets `atkins` on a laptop pass its work to a
machine that has the tooling.

The agent keeps a repository cache under DataDir so repeated jobs for
one repository fetch and skip the clone, and builds every job its own
work tree under <DataDir>/work.

## Types

<details>
<summary><code>type Checkout</code></summary>

```go
// Checkout records what the agent actually put in the work tree.
// Ref is the effective ref: the one the job named, or the default branch
// resolved for a job that named nothing. CommitSHA is what that ref
// pointed at when the job ran - a tag moves, so nobody can reproduce a
// run that records "v1.2.3" alone.
type Checkout struct {
	Ref       string
	CommitSHA string

	// Branch holds the branch name when Ref named a branch and is empty for a tag or a
	// bare commit. A job reads it as ATKINS_BRANCH.
	Branch string
}
```

</details>

<details>
<summary><code>type Options</code></summary>

```go
// Options configures an agent.
// Values come from .atkins/config.yml with the ATKINS_* overlay already
// applied; see FromConfig. The struct stays plain so a caller embedding
// the package can build one by hand.
type Options struct {
	// Server is the atkins server to claim jobs from. Empty uses the
	// stored default from `atkins --login`.
	Server string

	// Token is the shared enrolment secret, and the normal way an
	// agent joins: one secret in the environment, traded for the
	// rotating credentials the agent then uses.
	Token string

	// Email and Password log the agent in as an ordinary account that
	// an admin has flagged as an agent. Used when Token is empty.
	Email    string
	Password string

	// AgentID identifies this worker in job leases. Empty uses the
	// hostname.
	AgentID string

	// Labels advertise what this agent can run. A job requiring labels
	// only lands here if this agent advertises every one of them.
	Labels []string

	// DataDir holds the repository cache and job work trees.
	DataDir string

	// PollInterval is how long to wait after an empty claim.
	PollInterval time.Duration

	// JobTimeout bounds a single job. The agent kills the command and
	// reports a timeout and does not hold the lease forever.
	JobTimeout time.Duration

	// HeartbeatInterval is how often the agent renews a running job's lease.
	HeartbeatInterval time.Duration

	// Shell runs the job command. It receives `-c <command>`.
	Shell string

	// PreferHTTPS rewrites ssh remotes to https before cloning. A
	// container tends to have no key agent, so `git@host:owner/name.git`
	// would fail where `https://host/owner/name.git` works.
	PreferHTTPS bool
}
```

</details>

<details>
<summary><code>type Result</code></summary>

```go
// Result is the outcome of running a job command.
type Result struct {
	ExitCode int
	TimedOut bool
	Error    string
}
```

</details>

<details>
<summary><code>type Worker</code></summary>

```go
// Worker claims and runs jobs for one server.
type Worker struct {
	opts   *Options
	client *client.Client

	policy policy
	ssh    sshKeys
}
```

</details>

<details>
<summary><code>type Workspace</code></summary>

```go
// Workspace is a checkout prepared for one job.
type Workspace struct {
	// Root is the work tree the job runs under.
	Root string

	// Dir is the directory the command runs in: Root joined with the
	// job's working directory.
	Dir string

	// Checkout is what the work tree ended up at.
	Checkout Checkout

	// Artefacts is the directory the job copies files into to have them
	// kept. It sits beside the work tree, outside it, so
	// staging a file does not dirty the checkout the job is testing.
	Artefacts string

	// Env is what the workspace contributes to the job's environment:
	// the paths it laid out and the checkout it produced. prepare fills
	// it, and a caller holding the workspace may add to it before the
	// command starts. The values here win over the ones the agent
	// derives from the job.
	Env runner.Env
}
```

</details>

## Consts

<details>
<summary><code>const EnvToken</code></summary>

```go
// EnvToken is the enrolment secret's environment name. This file names it
// because the "no credentials configured" error points at it.
const EnvToken = "ATKINS_AGENT_TOKEN"
```

</details>

## Function symbols

- `func CleanWorkingDirectory (dir string) string`
- `func CloneURL (remoteURL,slug string, preferHTTPS bool) string`
- `func FromConfig (cfg *config.Config) *Options`
- `func New (ctx context.Context, opts *Options) (*Worker, error)`
- `func NewOptions () *Options`
- `func (*Worker) Run (ctx context.Context) error`

### `CleanWorkingDirectory`

`CleanWorkingDirectory` normalizes a job's working directory into a
clean path relative to the repository root, or "" for the root.

The server sanitizes this before storing it, so this is the second
gate on the same decision. It earns its place: the value arrives over
the network and is about to become a directory the agent runs a shell
command in, and an agent should not be one server bug away from
executing outside its checkout.

```go
func CleanWorkingDirectory(dir string) string
```

### `CloneURL`

`CloneURL` decides what to hand to `git clone`.

With preferHTTPS, an ssh remote is rewritten to https using the slug
the server derived. A container has no key agent, so the ssh form
would fail on a repository that clones fine anonymously over https.

```go
func CloneURL(remoteURL, slug string, preferHTTPS bool) string
```

### `FromConfig`

`FromConfig` builds agent options from a resolved configuration.

```go
func FromConfig(cfg *config.Config) *Options
```

### `New`

`New` returns a Worker, authenticating against the server.

The agent logs in like any other user. With Register on and no account
matching the credentials, it creates one: on a fresh
instance there is nobody to create it for them.

```go
func New(ctx context.Context, opts *Options) (*Worker, error)
```

### `NewOptions`

`NewOptions` returns options from the embedded default configuration.
A caller that has no resolved configuration falls back to it.

```go
func NewOptions() *Options
```

### `Run`

`Run` claims and executes jobs until the context ends.

```go
func (*Worker) Run(ctx context.Context) error
```

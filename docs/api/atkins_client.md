# Package ./client

```go
import (
	"github.com/titpetric/atkins/client"
}
```

Package client talks to an atkins CI/CD server.

It covers the whole client side of the feature: `atkins --login https://domain` and `atkins --register` obtain a credential, and every
later atkins run posts to /api/dispatch so the server knows which
repository, which directory in its work tree, and which command ran.

The package deliberately depends on net/http alone. Everything it
sends and receives is declared in types.go rather than imported from
server/api, so building the CLI does not pull in the server's
database and routing dependencies.

## Types

<details>
<summary><code>type APIError</code></summary>

```go
// APIError is a non-2xx response from the server.
type APIError struct {
	StatusCode int
	Message    string
}
```

</details>

<details>
<summary><code>type AgentSSHKey</code></summary>

```go
// AgentSSHKey is a deploy key handed to an agent.
type AgentSSHKey struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Host        string `json:"host"`
	PrivateKey  string `json:"private_key"`
	KnownHosts  string `json:"known_hosts"`
	Fingerprint string `json:"fingerprint"`
}
```

</details>

<details>
<summary><code>type Artefact</code></summary>

```go
// Artefact mirrors api.ArtefactView: one file a job produced.
type Artefact struct {
	ID          string `json:"id"`
	JobID       string `json:"job_id"`
	Path        string `json:"path"`
	Size        int64  `json:"size"`
	ContentType string `json:"content_type"`
	Checksum    string `json:"checksum"`
	CreatedAt   string `json:"created_at"`
	URL         string `json:"url"`
}
```

</details>

<details>
<summary><code>type ArtefactUpload</code></summary>

```go
// ArtefactUpload is one file being pushed to the server.
type ArtefactUpload struct {
	// Path is the name the pipeline gave the file, relative to the
	// directory the job ran in.
	Path string

	// ContentType is the media type, guessed from the extension.
	ContentType string

	// Checksum is the SHA256 the agent computed while reading the
	// file. The server compares it against what arrives, so a
	// truncated upload is refused rather than stored.
	Checksum string

	// Content is the file. It is streamed rather than buffered: an
	// artefact is as large as the server's limit allows.
	Content io.Reader
}
```

</details>

<details>
<summary><code>type Checkout</code></summary>

```go
// Checkout is what the client reports about the working copy a run
// happens in. It is the whole of what the server needs to reproduce the
// run elsewhere: which repository, where inside it, at what revision.
type Checkout struct {
	// Root is the absolute path of the repository work tree.
	Root string

	// RemoteURL is the `origin` remote, or the first remote defined.
	RemoteURL string

	// WorkingDirectory is the run directory relative to Root. Empty
	// when the run happens at the repository root.
	WorkingDirectory string

	Branch        string
	Revision      string
	DefaultBranch string

	// Dirty reports uncommitted changes in the work tree, tracked or
	// not. A local run of a dirty tree is ordinary; a dispatched one
	// would build something else.
	Dirty bool

	// Unpushed reports that no remote has HEAD, so nothing can clone it.
	Unpushed bool
}
```

</details>

<details>
<summary><code>type ClaimRequest</code></summary>

```go
// ClaimRequest is the body of POST /api/job/claim.
type ClaimRequest struct {
	AgentID string   `json:"agent_id"`
	Labels  []string `json:"labels,omitempty"`
}
```

</details>

<details>
<summary><code>type ClaimResponse</code></summary>

```go
// ClaimResponse is returned when an agent leases a job. The repository
// travels with the job so an agent can start cloning without a second
// round trip.
type ClaimResponse struct {
	Job        *Job        `json:"job"`
	Repository *Repository `json:"repository"`
}
```

</details>

<details>
<summary><code>type Client</code></summary>

```go
// Client is an HTTP client for one atkins server.
type Client struct {
	server string
	http   *http.Client

	// store and credential are set for authenticated clients. The
	// store is written back when a refresh rotates the tokens.
	store      *Store
	credential *Credential
}
```

</details>

<details>
<summary><code>type Credential</code></summary>

```go
// Credential is a stored login for one server.
type Credential struct {
	Server       string `json:"server"`
	UserID       string `json:"user_id"`
	Username     string `json:"username"`
	Token        string `json:"token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresAt    int64  `json:"expires_at"`
}
```

</details>

<details>
<summary><code>type DispatchOptions</code></summary>

```go
// DispatchOptions describes the run being handed over.
type DispatchOptions struct {
	// Directory is where atkins was invoked. Empty means the process
	// working directory.
	Directory string

	// Command is the atkins invocation. Empty means os.Args.
	Command string

	// Server overrides which logged-in server to dispatch to.
	Server string
}
```

</details>

<details>
<summary><code>type DispatchRequest</code></summary>

```go
// DispatchRequest is the body of POST /api/dispatch.
type DispatchRequest struct {
	Repository       RepositoryPayload `json:"repository"`
	WorkingDirectory string            `json:"working_directory"`
	Command          string            `json:"command"`
	ParentID         string            `json:"parent_id,omitempty"`
	Labels           []string          `json:"labels,omitempty"`
	Params           map[string]any    `json:"params,omitempty"`
	Artefacts        []string          `json:"artefacts,omitempty"`

	// Agent names the machine already running the job. Set, the job is
	// recorded as running there instead of queued for an agent to
	// claim: it is how a run that happens on a laptop still appears on
	// the server, with the log and the outcome an agent would have
	// reported.
	Agent string `json:"agent,omitempty"`
}
```

</details>

<details>
<summary><code>type DispatchResponse</code></summary>

```go
// DispatchResponse is returned by POST /api/dispatch.
type DispatchResponse struct {
	JobID          string `json:"job_id"`
	ParentID       string `json:"parent_id,omitempty"`
	RootID         string `json:"root_id"`
	Depth          int64  `json:"depth"`
	RepositoryID   string `json:"repository_id"`
	RepositorySlug string `json:"repository_slug"`
	Status         string `json:"status"`

	// ViewToken opens the job page in a browser without a session. A
	// server that keeps jobs private returns one; a public one returns
	// nothing and the plain job URL opens.
	ViewToken string `json:"view_token,omitempty"`
}
```

</details>

<details>
<summary><code>type Dispatched</code></summary>

```go
// Dispatched is a job handed to a server.
// A nil *Dispatched means the run was not delegated and the caller
// should execute it locally.
type Dispatched struct {
	// JobID is the server's ID for the run.
	JobID string

	// URL is where the job is watched in a browser. It is the only
	// thing atkins prints when it delegates a run.
	URL string

	// RepositorySlug is the normalized repository the job belongs to.
	RepositorySlug string
}
```

</details>

<details>
<summary><code>type EnrolRequest</code></summary>

```go
// EnrolRequest is the body of POST /api/agent/enrol.
type EnrolRequest struct {
	Token   string   `json:"token"`
	AgentID string   `json:"agent_id"`
	Labels  []string `json:"labels,omitempty"`
}
```

</details>

<details>
<summary><code>type Job</code></summary>

```go
// Job is the subset of a queued job an agent needs to run it. It
// mirrors the columns of server/model.Job that matter on this side.
type Job struct {
	ID               string `json:"id"`
	ParentID         string `json:"parent_id"`
	RootID           string `json:"root_id"`
	RepositoryID     string `json:"repository_id"`
	WorkingDirectory string `json:"working_directory"`
	Command          string `json:"command"`
	Ref              string `json:"ref"`
	CommitSHA        string `json:"commit_sha"`
	CloneDepth       int64  `json:"clone_depth"`
	Labels           string `json:"labels"`
	Params           string `json:"params"`
	Status           string `json:"status"`

	// ArtefactPaths are the comma separated globs the agent collects
	// after the command exits.
	ArtefactPaths string `json:"artefact_paths"`
}
```

</details>

<details>
<summary><code>type JobCheckoutRequest</code></summary>

```go
// JobCheckoutRequest is the body of POST /api/job/{jobID}/checkout: what
// the agent actually put in the work tree.
type JobCheckoutRequest struct {
	Ref       string `json:"ref"`
	CommitSHA string `json:"commit_sha"`
}
```

</details>

<details>
<summary><code>type JobLogRequest</code></summary>

```go
// JobLogRequest is the body of POST /api/job/{jobID}/log.
type JobLogRequest struct {
	Stream  string `json:"stream"`
	Content string `json:"content"`
}
```

</details>

<details>
<summary><code>type JobStatusRequest</code></summary>

```go
// JobStatusRequest is the body of POST /api/job/{jobID}/status.
type JobStatusRequest struct {
	Status   string `json:"status"`
	ExitCode int64  `json:"exit_code"`
	Error    string `json:"error,omitempty"`
}
```

</details>

<details>
<summary><code>type LoginRequest</code></summary>

```go
// LoginRequest is the body of POST /api/user/login.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Hostname string `json:"hostname"`
}
```

</details>

<details>
<summary><code>type LogoutRequest</code></summary>

```go
// LogoutRequest is the body of POST /api/user/logout.
type LogoutRequest struct {
	RefreshToken string `json:"refresh_token"`
}
```

</details>

<details>
<summary><code>type PolicyResponse</code></summary>

```go
// PolicyResponse is what an agent may run.
type PolicyResponse struct {
	Policy   string   `json:"policy"`
	Patterns []string `json:"patterns"`
}
```

</details>

<details>
<summary><code>type Prompter</code></summary>

```go
// Prompter reads credentials from the terminal.
type Prompter struct {
	in  *bufio.Reader
	out io.Writer

	// fd is the file descriptor used for the no-echo password read.
	fd int
}
```

</details>

<details>
<summary><code>type RecordOptions</code></summary>

```go
// RecordOptions describes the local run being logged.
type RecordOptions struct {
	// Directory is where atkins was invoked. Empty means the process
	// working directory.
	Directory string

	// Command is the atkins invocation. Empty means os.Args.
	Command string

	// Server overrides which logged-in server to record on.
	Server string
}
```

</details>

<details>
<summary><code>type Recorder</code></summary>

```go
// Recorder is a run happening here that is logged on a CI/CD server.
// It is the other half of dispatch: the pipeline executes on this
// machine, as it does on a machine that never logged in, and the server
// ends up with the job an agent would have produced — the same command,
// checkout, output, exit code and duration. A history of what a team
// runs should not depend on where the tooling happens to be installed.
//
// A nil *Recorder is the unrecorded run, and every method tolerates it,
// so the caller has no branch to write.
type Recorder struct {
	client  *Client
	jobID   string
	url     string
	agentID string

	mu     sync.Mutex
	buffer strings.Builder

	// stop ends the lease renewals.
	stop func()
}
```

</details>

<details>
<summary><code>type RefreshRequest</code></summary>

```go
// RefreshRequest is the body of POST /api/user/refreshToken.
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}
```

</details>

<details>
<summary><code>type RegisterRequest</code></summary>

```go
// RegisterRequest is the body of POST /api/user/register.
type RegisterRequest struct {
	Email    string `json:"email"`
	Username string `json:"username"`
	FullName string `json:"full_name"`
	Password string `json:"password"`
}
```

</details>

<details>
<summary><code>type Repository</code></summary>

```go
// Repository is the checkout an agent has to reproduce.
type Repository struct {
	ID            string `json:"id"`
	Slug          string `json:"slug"`
	RemoteURL     string `json:"remote_url"`
	DefaultBranch string `json:"default_branch"`
}
```

</details>

<details>
<summary><code>type RepositoryPayload</code></summary>

```go
// RepositoryPayload describes the checkout a run happens in.
// Ref is what the agent checks out: a branch, a tag, a commit sha or a
// fully qualified refname. The client fills it with the commit it is
// sitting on, because a run dispatched from a laptop should build that
// commit rather than whatever the branch has moved to by the time an
// agent picks the job up.
type RepositoryPayload struct {
	RemoteURL     string `json:"remote_url"`
	Ref           string `json:"ref,omitempty"`
	DefaultBranch string `json:"default_branch"`
}
```

</details>

<details>
<summary><code>type Store</code></summary>

```go
// Store is the on-disk credential file. It holds one credential per
// server so a machine can talk to more than one atkins instance, plus
// the server used when none is named.
type Store struct {
	Default string                 `json:"default"`
	Servers map[string]*Credential `json:"servers"`

	// path is where the store was loaded from and where Save writes.
	path string
}
```

</details>

<details>
<summary><code>type TokenResponse</code></summary>

```go
// TokenResponse is returned by login, register and refresh.
type TokenResponse struct {
	UserID       string `json:"user_id"`
	Username     string `json:"username"`
	Token        string `json:"token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresAt    int64  `json:"expires_at"`
}
```

</details>

## Consts

<details>
<summary><code>const DefaultTimeout</code></summary>

```go
// DefaultTimeout bounds a single API call. Dispatch happens on the hot
// path of every atkins run, so it must never hang a build.
const DefaultTimeout = 10 * time.Second
```

</details>

<details>
<summary><code>const HeaderArtefactChecksum</code></summary>

```go
// HeaderArtefactChecksum carries ArtefactUpload.Checksum, mirroring
// server/api.
const HeaderArtefactChecksum = "X-Atkins-Checksum"
```

</details>

<details>
<summary><code>const MinPasswordLength</code></summary>

```go
// MinPasswordLength mirrors what the server accepts, so a typo is
// caught at the prompt rather than after a round trip.
const MinPasswordLength = 8
```

</details>

<details>
<summary><code>const UploadTimeout</code></summary>

```go
// UploadTimeout bounds one artefact transfer. It is generous because an
// artefact is measured in megabytes rather than in fields, and nothing
// is waiting on it: the job has already finished by the time it runs.
const UploadTimeout = 5 * time.Minute
```

</details>

<details>
<summary><code>const EnvJobID, EnvParentJobID, EnvRootJobID, EnvJobParams, EnvArtefacts, EnvServer, EnvLabels, EnvNoDispatch, EnvCI</code></summary>

```go
// Environment variables a dispatched run exports to its steps. They are
// the CI/CD contract: a command running under an agent can tell it is
// part of a job, address that job, and dispatch children under it.
const (
	// EnvJobID is the job the command is running as. A nested atkins
	// invocation reads it and dispatches its own job as a child.
	EnvJobID = "ATKINS_JOB_ID"

	// EnvParentJobID is the job that dispatched this one, if any.
	EnvParentJobID = "ATKINS_PARENT_JOB_ID"

	// EnvRootJobID is the top of this job's tree, stable across nesting.
	EnvRootJobID = "ATKINS_ROOT_JOB_ID"

	// EnvJobParams is the JSON object the job was dispatched with.
	EnvJobParams = "ATKINS_JOB_PARAMS"

	// EnvArtefacts is a directory the job copies files into to have
	// them kept. Whatever is in it when the command exits is uploaded
	// and attached to the job.
	//
	// It is the declaration that needs no schema: a pipeline says what
	// it wants kept by putting it somewhere, which works for any
	// command in any language without atkins having to parse its
	// output or its configuration.
	EnvArtefacts = "ATKINS_ARTEFACTS"

	// EnvServer selects which logged-in server to dispatch to when a
	// machine has credentials for more than one.
	EnvServer = "ATKINS_SERVER"

	// EnvLabels constrains which agents may run the dispatched job,
	// comma separated.
	EnvLabels = "ATKINS_LABELS"

	// EnvNoDispatch runs locally instead of dispatching, without
	// logging out. An agent sets it for the command it runs, which is
	// what stops a job from dispatching itself forever.
	EnvNoDispatch = "ATKINS_NO_DISPATCH"

	// EnvCI is set for every run an agent performs, matching what
	// other CI systems export.
	EnvCI = "CI"
)
```

</details>

<details>
<summary><code>const EnvEmail, EnvUsername, EnvPassword</code></summary>

```go
// Environment overrides for non-interactive login and registration.
// They exist so a container or a provisioning script can attach a
// machine without a human at the keyboard.
const (
	EnvEmail    = "ATKINS_EMAIL"
	EnvUsername = "ATKINS_USERNAME"
	EnvPassword = "ATKINS_PASSWORD"
)
```

</details>

<details>
<summary><code>const PolicyOpen, PolicyAllowlist</code></summary>

```go
// Repository policy values, mirroring server/model.
const (
	PolicyOpen      = "open"
	PolicyAllowlist = "allowlist"
)
```

</details>

<details>
<summary><code>const StreamOutput, StreamError</code></summary>

```go
// Log stream names, mirroring server/storage.
const (
	StreamOutput = "output"
	StreamError  = "error"
)
```

</details>

<details>
<summary><code>const StatusPassed, StatusFailed, StatusTimeout, StatusCancelled</code></summary>

```go
// Job status values reported by the CLI. They mirror the server's
// terminal statuses in server/model.
const (
	StatusPassed    = "passed"
	StatusFailed    = "failed"
	StatusTimeout   = "timeout"
	StatusCancelled = "cancelled"
)
```

</details>

## Vars

<details>
<summary><code>var ErrDispatchDisabled</code></summary>

```go
// ErrDispatchDisabled is returned when the environment forbids handing
// the run over: an agent sets it for the command it runs, which is what
// stops a job from dispatching itself forever.
var ErrDispatchDisabled = errors.New(EnvNoDispatch + " is set")
```

</details>

<details>
<summary><code>var ErrNoTerminal</code></summary>

```go
// ErrNoTerminal is returned when a password is needed but stdin isn't a
// terminal and no environment override was provided.
var ErrNoTerminal = errors.New("stdin is not a terminal: set ATKINS_PASSWORD to log in non-interactively")
```

</details>

<details>
<summary><code>var ErrNotARepository</code></summary>

```go
// ErrNotARepository is returned when a directory is not inside a git
// work tree. Dispatch is skipped in that case: without a repository
// there is nothing for another machine to check out.
var ErrNotARepository = errors.New("not inside a git repository")
```

</details>

<details>
<summary><code>var ErrNotLoggedIn</code></summary>

```go
// ErrNotLoggedIn is returned when no credential is stored for a server.
var ErrNotLoggedIn = errors.New("not logged in: run `atkins --login <url>`")
```

</details>

<details>
<summary><code>var UserAgent</code></summary>

```go
// UserAgent is sent with every request. The main package overwrites it
// with the build version, which is what a server log needs to tell an
// old client from a new one.
var UserAgent = "atkins"
```

</details>

<details>
<summary><code>var ErrDirtyCheckout, ErrUnpushedCheckout</code></summary>

```go
// Errors reported when a checkout cannot be reproduced elsewhere.
var (
	// ErrDirtyCheckout is a work tree with uncommitted changes.
	ErrDirtyCheckout = errors.New("the working tree has uncommitted changes")

	// ErrUnpushedCheckout is a commit no remote has.
	ErrUnpushedCheckout = errors.New("HEAD has not been pushed to a remote")
)
```

</details>

## Function symbols

- `func Command (argv []string) string`
- `func Configure (value config.ClientConfig)`
- `func CredentialsPath () (string, error)`
- `func DetectCheckout (dir string) (*Checkout, error)`
- `func Dispatch (ctx context.Context, opts DispatchOptions) (*Dispatched, error)`
- `func DispatchDisabled () bool`
- `func Labels () []string`
- `func LoadStore () (*Store, error)`
- `func New (server string) *Client`
- `func NewPrompter () *Prompter`
- `func NewPrompterFrom (in io.Reader, out io.Writer) *Prompter`
- `func NormalizeServer (server string) string`
- `func Open (server string) (*Client, error)`
- `func Record (ctx context.Context, opts RecordOptions) *Recorder`
- `func RunLogin (ctx context.Context, server string) error`
- `func RunLogout (ctx context.Context, server string) error`
- `func RunRegister (ctx context.Context, server string) error`
- `func Settings () config.ClientConfig`
- `func SplitLabels (value string) []string`
- `func (*APIError) Error () string`
- `func (*Checkout) Payload () RepositoryPayload`
- `func (*Checkout) Publishable () error`
- `func (*Client) AppendLog (ctx context.Context, jobID,stream,content string) error`
- `func (*Client) Artefacts (ctx context.Context, jobID string) ([]Artefact, error)`
- `func (*Client) Claim (ctx context.Context, agentID string, labels []string) (*ClaimResponse, error)`
- `func (*Client) Dispatch (ctx context.Context, req DispatchRequest) (*DispatchResponse, error)`
- `func (*Client) Enrol (ctx context.Context, req EnrolRequest) (*Credential, error)`
- `func (*Client) Heartbeat (ctx context.Context, jobID,agentID string) error`
- `func (*Client) JobURL (jobID,viewToken string) string`
- `func (*Client) Login (ctx context.Context, req LoginRequest) (*Credential, error)`
- `func (*Client) Logout (ctx context.Context) error`
- `func (*Client) Policy (ctx context.Context) (*PolicyResponse, error)`
- `func (*Client) Refresh (ctx context.Context) error`
- `func (*Client) Register (ctx context.Context, req RegisterRequest) (*Credential, error)`
- `func (*Client) ReportCheckout (ctx context.Context, jobID string, req JobCheckoutRequest) error`
- `func (*Client) ReportStatus (ctx context.Context, jobID string, req JobStatusRequest) error`
- `func (*Client) SSHKeys (ctx context.Context) ([]AgentSSHKey, error)`
- `func (*Client) Server () string`
- `func (*Client) UploadArtefact (ctx context.Context, jobID string, upload ArtefactUpload) (*Artefact, error)`
- `func (*Client) Username () string`
- `func (*Credential) Expired () bool`
- `func (*Prompter) Line (label,env string) (string, error)`
- `func (*Prompter) Password (label string) (string, error)`
- `func (*Recorder) Cancelled (ctx context.Context)`
- `func (*Recorder) Finish (ctx context.Context, exitCode int, runErr error)`
- `func (*Recorder) URL () string`
- `func (*Recorder) Write (p []byte) (int, error)`
- `func (*Store) Get (server string) (*Credential, bool)`
- `func (*Store) Remove (server string)`
- `func (*Store) Save () error`
- `func (*Store) Set (credential *Credential)`

### Command

Command renders an argv as the command to record.

argv[0] is replaced with `atkins` rather than reduced to its base
name: the recorded command is re-run on another machine, where the
binary is called `atkins`, and what this one happens to be named is a
fact about this laptop. A build under test as `./bin/atkins-linux-amd64`,
the `atkins.old` the install job leaves behind, or a distro package
named `atkins-cli` would otherwise queue a command no agent can run,
and the job comes back as exit 127 with an empty log.

A run of something other than atkins is what the explicit `command`
override on the trigger endpoint is for.

```go
func Command(argv []string) string
```

### Configure

Configure installs the resolved client configuration.

```go
func Configure(value config.ClientConfig)
```

### CredentialsPath

CredentialsPath returns the location of the credentials file.

The configured path wins; ATKINS_CREDENTIALS reaches it through the
config overlay. The environment is still read directly as a fallback
so a command that never loaded configuration — a test, or a tool
embedding this package — still honours it.

```go
func CredentialsPath() (string, error)
```

### DetectCheckout

DetectCheckout inspects the git work tree containing dir.

A repository with no remote is rejected along with a plain directory:
the point of dispatch is that some other machine can fetch the same
code, and a purely local repository fails that test.

```go
func DetectCheckout(dir string) (*Checkout, error)
```

### Dispatch

Dispatch hands the current run to the CI/CD server this machine is
logged in to, and returns where to watch it.

Handing a run over is asked for — by --dispatch, or by
`client.dispatch` in the configuration — so every reason it can't
happen is an error rather than a quiet local run. The one that isn't
worth stopping over is a repository nobody can clone, and that is
refused before the job exists: an unpushed commit dispatches a job
that dies in `git checkout` on a machine nobody is watching.

```go
func Dispatch(ctx context.Context, opts DispatchOptions) (*Dispatched, error)
```

### DispatchDisabled

DispatchDisabled reports whether ATKINS_NO_DISPATCH is set to a
true-ish value.

```go
func DispatchDisabled() bool
```

### Labels

Labels returns the labels constraining which agents may run this
machine's jobs.

```go
func Labels() []string
```

### LoadStore

LoadStore reads the credential file. A missing file is not an error;
it yields an empty store ready to be written to.

```go
func LoadStore() (*Store, error)
```

### New

New returns an unauthenticated client for a server. It is what the
login and register flows use before a credential exists.

```go
func New(server string) *Client
```

### NewPrompter

NewPrompter returns a Prompter over stdin and stderr.

Prompts go to stderr so that `atkins --login` can be used in a
pipeline without the prompt text landing in captured stdout.

```go
func NewPrompter() *Prompter
```

### NewPrompterFrom

NewPrompterFrom returns a Prompter reading from in and writing to out.

The reader is not a terminal, so Password refuses unless the value
comes from the environment — which is the behaviour a script gets,
and the behaviour worth testing.

```go
func NewPrompterFrom(in io.Reader, out io.Writer) *Prompter
```

### NormalizeServer

NormalizeServer trims a server URL to its canonical form so
`https://ci.example.com/` and `https://ci.example.com` are one entry.

```go
func NormalizeServer(server string) string
```

### Open

Open returns an authenticated client for a server, or for the stored
default server when server is empty.

It returns ErrNotLoggedIn when there is no credential, which callers
on the dispatch path treat as "this machine isn't attached to a CI
server" rather than as a failure.

```go
func Open(server string) (*Client, error)
```

### Record

Record opens a job for a run about to happen here.

It returns nil when there is nothing to record against: no
credential, no git repository, recording turned off, or a server that
can't be reached. Recording is bookkeeping, and losing it must not
cost the run — unlike dispatch, which is the run.

```go
func Record(ctx context.Context, opts RecordOptions) *Recorder
```

### RunLogin

RunLogin drives `atkins --login https://domain`.

It prompts for an email and password, exchanges them for tokens and
writes them to ~/.atkins/credentials.json. From then on every atkins
run in a git repository dispatches to this server.

```go
func RunLogin(ctx context.Context, server string) error
```

### RunLogout

RunLogout drives `atkins --logout`. An empty server logs out of the
default server.

```go
func RunLogout(ctx context.Context, server string) error
```

### RunRegister

RunRegister drives `atkins --register https://domain`.

Registration collects a username, an email and a password, and logs
the new account in on success, so a fresh instance goes from nothing
to a usable credential in one command.

```go
func RunRegister(ctx context.Context, server string) error
```

### Settings

Settings returns the configuration in force.

```go
func Settings() config.ClientConfig
```

### SplitLabels

SplitLabels parses a comma separated label list.

```go
func SplitLabels(value string) []string
```

### Error

Error implements error.

```go
func (*APIError) Error() string
```

### Payload

Payload converts the checkout into a dispatch payload.

The ref is the commit rather than the branch. A dispatched run belongs
to the code in front of the person who started it, and a branch name
would let the agent build whatever that branch has moved to by the
time the job is claimed.

```go
func (*Checkout) Payload() RepositoryPayload
```

### Publishable

Publishable reports whether another machine could reproduce this
checkout from the repository's remote.

A dispatched run names the commit it was started from, so a tree that
only exists on this disk queues a job that fails in `git checkout`
minutes later. The refusal belongs here, where the fix — commit and
push, or run locally — is still in front of the person.

```go
func (*Checkout) Publishable() error
```

### AppendLog

AppendLog records a chunk of job output.

```go
func (*Client) AppendLog(ctx context.Context, jobID, stream, content string) error
```

### Artefacts

Artefacts lists the files a job produced.

```go
func (*Client) Artefacts(ctx context.Context, jobID string) ([]Artefact, error)
```

### Claim

Claim leases the oldest pending job this agent can run.

A nil job with a nil error means the queue held nothing for us, which
is the normal outcome of a poll and not a condition worth logging.

```go
func (*Client) Claim(ctx context.Context, agentID string, labels []string) (*ClaimResponse, error)
```

### Dispatch

Dispatch records a job for the current run and returns its ID.

```go
func (*Client) Dispatch(ctx context.Context, req DispatchRequest) (*DispatchResponse, error)
```

### Enrol

Enrol trades the shared agent token for agent credentials.

The credentials are persisted like any login, so a restarted agent
with the same identity resumes without re-enrolling.

```go
func (*Client) Enrol(ctx context.Context, req EnrolRequest) (*Credential, error)
```

### Heartbeat

Heartbeat extends this agent's lease on a job.

```go
func (*Client) Heartbeat(ctx context.Context, jobID, agentID string) error
```

### JobURL

JobURL is where a job is watched in a browser. It is the one thing
atkins prints when it hands a run to a server.

A server that keeps jobs private hands back a view token with the
job, and it belongs in the URL: the page has no session to check, and
the whole point of the line atkins prints is that pasting it into a
browser opens the run. An empty token leaves the URL as it was.

```go
func (*Client) JobURL(jobID, viewToken string) string
```

### Login

Login exchanges email and password for tokens and stores them.

```go
func (*Client) Login(ctx context.Context, req LoginRequest) (*Credential, error)
```

### Logout

Logout revokes the stored session and forgets the credential.

The local credential is dropped even when the server call fails: a
user asking to log out of an unreachable server should still end up
logged out on this machine.

```go
func (*Client) Logout(ctx context.Context) error
```

### Policy

Policy returns the repository policy the agent must enforce.

```go
func (*Client) Policy(ctx context.Context) (*PolicyResponse, error)
```

### Refresh

Refresh exchanges the refresh token for a new access token and
persists the rotated pair.

```go
func (*Client) Refresh(ctx context.Context) error
```

### Register

Register creates an account and stores the credential it returns.

```go
func (*Client) Register(ctx context.Context, req RegisterRequest) (*Credential, error)
```

### ReportCheckout

ReportCheckout records the ref and commit an agent checked out.

```go
func (*Client) ReportCheckout(ctx context.Context, jobID string, req JobCheckoutRequest) error
```

### ReportStatus

ReportStatus settles a job into a terminal state.

```go
func (*Client) ReportStatus(ctx context.Context, jobID string, req JobStatusRequest) error
```

### SSHKeys

SSHKeys returns the deploy keys the agent should install.

```go
func (*Client) SSHKeys(ctx context.Context) ([]AgentSSHKey, error)
```

### Server

Server returns the server URL the client talks to.

```go
func (*Client) Server() string
```

### UploadArtefact

UploadArtefact pushes one file a job produced.

The body is the file itself. A multipart envelope would buy nothing
here: there is one part, and the two fields around it fit in a query
parameter and a header.

```go
func (*Client) UploadArtefact(ctx context.Context, jobID string, upload ArtefactUpload) (*Artefact, error)
```

### Username

Username returns the logged-in username, if any.

```go
func (*Client) Username() string
```

### Expired

Expired reports whether the access token is past, or nearly past, its
expiry. The skew means a request never goes out with a token that
expires while it's in flight.

```go
func (*Credential) Expired() bool
```

### Line

Line prompts for a value, returning the trimmed input. When env names
a set environment variable, its value is used and no prompt is shown.

```go
func (*Prompter) Line(label, env string) (string, error)
```

### Password

Password prompts for a secret without echoing it.

```go
func (*Prompter) Password(label string) (string, error)
```

### Cancelled

Cancelled settles the job as cancelled, for a run the user stopped.

```go
func (*Recorder) Cancelled(ctx context.Context)
```

### Finish

Finish flushes what was written and settles the job.

The status is derived from the same exit code the shell sees, so a
recorded run and the terminal it ran in never disagree.

```go
func (*Recorder) Finish(ctx context.Context, exitCode int, runErr error)
```

### URL

URL is where the run is watched in a browser, empty when unrecorded.

```go
func (*Recorder) URL() string
```

### Write

Write buffers output for the job log. It makes a Recorder an
io.Writer, which is what lets the runner hand it a transcript without
knowing what a job is.

```go
func (*Recorder) Write(p []byte) (int, error)
```

### Get

Get returns the credential for a server, or for the default server
when server is empty.

```go
func (*Store) Get(server string) (*Credential, bool)
```

### Remove

Remove drops the credential for a server.

```go
func (*Store) Remove(server string)
```

### Save

Save writes the store back to disk.

The file holds bearer tokens, so both it and its directory are
created 0600/0700 and the write goes through a temporary file: a
crash mid-write should not leave a truncated credential behind.

```go
func (*Store) Save() error
```

### Set

Set stores a credential and makes its server the default.

```go
func (*Store) Set(credential *Credential)
```

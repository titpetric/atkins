# Package ./server/storage

```go
import (
	"github.com/titpetric/atkins/server/storage"
}
```

Package storage owns the SQL for the atkins CI/CD server.

Queries run through github.com/titpetric/pdo, whose generic methods
(Get[T], Select[T], Insert, Update) scan straight into the mig-generated
model types. A *pdo.PDO is request-scoped and not safe for concurrent
use, so every method allocates one over the shared *sqlx.DB pool.

Callers outside this package should not issue SQL; add a method here
instead so transactions and scoping stay in one place.

## Types

```go
// CheckoutRequest is what an agent reports having checked out.
type CheckoutRequest struct {
	// Ref is the effective ref: the one the job named, or the default
	// branch the agent resolved for a job that named none.
	Ref string

	// CommitSHA is the commit the work tree was placed at.
	CommitSHA string
}
```

```go
// CreateRequest is the input for creating a user. It is what
// `atkins --register` collects at the prompt.
type CreateRequest struct {
	Email    string `json:"email"`
	Username string `json:"username"`
	FullName string `json:"full_name"`
	Password string `json:"password"`
}
```

```go
// Flags is the administrative state of a user. A nil field is left
// alone, so an admin can revoke one privilege without restating the
// others.
type Flags struct {
	IsAdmin  *bool `json:"is_admin"`
	IsActive *bool `json:"is_active"`
	IsAgent  *bool `json:"is_agent"`
}
```

```go
// JobLogStorage persists output captured from a job.
type JobLogStorage struct {
	db *sqlx.DB
}
```

```go
// JobRequest is the input for dispatching a job.
type JobRequest struct {
	// ParentID is the dispatching job, when a running pipeline queues
	// work of its own. Empty for a root job.
	ParentID string

	RepositoryID string
	UserID       string

	// WorkingDirectory is relative to the repository root. It is where
	// the agent runs the command, and it is what lets one repository
	// carry many independently dispatched projects.
	WorkingDirectory string

	// Command is the atkins invocation to run, verbatim.
	Command string

	// Ref is what to check out: a branch, a tag, a commit sha or a
	// fully qualified refname. Empty means the repository's default
	// branch, resolved by the agent when the job runs rather than here,
	// so a nightly trigger follows the branch it is pointed at.
	Ref string

	// CloneDepth limits the history of the job's work tree. 0 is the
	// whole history.
	CloneDepth int64

	Labels []string

	// Params is a JSON object handed to the job as ATKINS_JOB_PARAMS.
	Params string
}
```

```go
// JobStorage persists dispatched jobs and their lifecycle.
type JobStorage struct {
	db *sqlx.DB

	// maxDepth bounds how deep a job tree may nest. A pipeline that
	// dispatches a child that dispatches a child is useful; one that
	// does so without a floor is a fork bomb with a queue in front.
	maxDepth int64

	// leaseTTL is how long an agent holds a claimed job before the
	// server considers it dead and reclaims the job.
	leaseTTL time.Duration
}
```

```go
// ListFilter narrows a job listing. A zero filter lists everything.
type ListFilter struct {
	RepositoryID string
	UserID       string
	RootID       string
	Status       model.JobStatus
	Limit        int
}
```

```go
// RepositoryRequest is the repository detail a client reports.
type RepositoryRequest struct {
	RemoteURL     string `json:"remote_url"`
	DefaultBranch string `json:"default_branch"`
}
```

```go
// RepositoryRuleStorage persists the repository allowlist.
type RepositoryRuleStorage struct {
	db *sqlx.DB
}
```

```go
// RepositoryStorage persists the git repositories the server has seen.
type RepositoryStorage struct {
	db *sqlx.DB
}
```

```go
// RuleRequest is the input for creating an allowlist rule.
type RuleRequest struct {
	// Pattern is matched against the repository slug. See
	// model.MatchRepository for the syntax.
	Pattern string `json:"pattern"`

	Description string `json:"description"`
}
```

```go
// SSHKeyRequest is the input for adding a key.
type SSHKeyRequest struct {
	// Name identifies the key to operators. It is unique.
	Name string `json:"name"`

	// Host scopes the key to one git host, e.g. github.com. Empty
	// offers the key for any host.
	Host string `json:"host"`

	// PrivateKey is the PEM-encoded key. It must not be passphrase
	// protected: an agent has nobody to ask.
	PrivateKey string `json:"private_key"`

	// KnownHosts pins the host keys the agent will accept. When
	// empty, the agent trusts the host on first use.
	KnownHosts string `json:"known_hosts"`
}
```

```go
// SSHKeyStorage persists the deploy keys agents clone with.
// The private material never leaves this package except through
// ListForAgent, which is reachable only by an enrolled agent. Admin
// listings carry the fingerprint and the public key, which is what an
// operator needs to install the key on the forge.
type SSHKeyStorage struct {
	db *sqlx.DB
}
```

```go
// SessionRequest carries the client detail recorded with a login. All
// fields are advisory; they exist so a user can tell their machines
// apart when reviewing active sessions.
type SessionRequest struct {
	Hostname   string
	UserAgent  string
	RemoteAddr string
}
```

```go
// SessionStorage persists CLI login sessions and their refresh tokens.
type SessionStorage struct {
	db *sqlx.DB

	// ttl is how long a session (and therefore its refresh token)
	// stays valid without a re-login.
	ttl time.Duration
}
```

```go
// SettingStorage reads and writes runtime configuration.
// Values are cached in memory because they are read on the dispatch
// path, which runs on every atkins invocation. The cache is invalidated
// on write, and this process is the only writer.
type SettingStorage struct {
	db *sqlx.DB

	mu    sync.RWMutex
	cache map[string]string
}
```

```go
// SettingValue is a definition together with its effective value.
type SettingValue struct {
	model.SettingDefinition

	Value     string `json:"value"`
	IsDefault bool   `json:"is_default"`
}
```

```go
// StatusRequest is the terminal report an agent (or the CLI itself)
// sends when a job settles.
type StatusRequest struct {
	Status   model.JobStatus
	ExitCode int64
	Error    string
}
```

```go
// UserStorage persists users and verifies their credentials.
type UserStorage struct {
	db *sqlx.DB
}
```

## Consts

```go
// ConnectionName is the platform database connection the server prefers.
// Configure either of:
//
//	PLATFORM_DB_ATKINS="sqlite://file:atkins.db"
//	PLATFORM_DB_DEFAULT="sqlite://file:atkins.db"
//
// The named connection wins when both are set. With neither, the
// platform default is an in-memory sqlite database, which is useful for
// tests and useless for a server: the server command sets
// PLATFORM_DB_DEFAULT to a file when the environment doesn't.
const ConnectionName = "atkins"
```

```go
// DefaultSessionTTL is how long a CLI login lasts before the user has to
// run `atkins --login` again. It's long on purpose: the access token is
// short-lived and refreshed silently, so this is the "how often do I get
// interrupted" knob.
const DefaultSessionTTL = 30 * 24 * time.Hour
```

```go
// MaxLogChunk bounds one appended chunk. A job that prints a gigabyte
// should fill the page slowly rather than the database quickly.
const MaxLogChunk = 256 * 1024
```

```go
// Project is the migration project name recorded in the migrations table.
const Project = "atkins"
```

```go
// Defaults applied when the corresponding JobStorage fields are zero.
const (
	// DefaultMaxDepth allows a dispatcher job to fan out to children
	// and those children to fan out once more.
	DefaultMaxDepth = 3

	// DefaultLeaseTTL is how long a claimed job stays claimed without
	// a heartbeat or a terminal report.
	DefaultLeaseTTL = 15 * time.Minute
)
```

```go
// Stream names for job output.
const (
	// StreamOutput is combined stdout and stderr from the command.
	StreamOutput = "output"

	// StreamError is the agent's own commentary: clone failures,
	// timeouts, anything that happened around the command rather than
	// inside it.
	StreamError = "error"
)
```

## Function symbols

- `func DB (ctx context.Context, name string) (*sqlx.DB, error)`
- `func Migrate (ctx context.Context, db *sqlx.DB, schema fs.FS) error`
- `func NewJobLogStorage (db *sqlx.DB) *JobLogStorage`
- `func NewJobStorage (db *sqlx.DB, maxDepth int64, leaseTTL time.Duration) *JobStorage`
- `func NewRepositoryRuleStorage (db *sqlx.DB) *RepositoryRuleStorage`
- `func NewRepositoryStorage (db *sqlx.DB) *RepositoryStorage`
- `func NewSSHKeyStorage (db *sqlx.DB) *SSHKeyStorage`
- `func NewSessionStorage (db *sqlx.DB, ttl time.Duration) *SessionStorage`
- `func NewSettingStorage (db *sqlx.DB) *SettingStorage`
- `func NewUserStorage (db *sqlx.DB) *UserStorage`
- `func (*JobLogStorage) Append (ctx context.Context, jobID,stream,content string) error`
- `func (*JobLogStorage) List (ctx context.Context, jobID string) ([]model.JobLog, error)`
- `func (*JobStorage) Claim (ctx context.Context, agentID string, labels []string) (*model.Job, error)`
- `func (*JobStorage) Create (ctx context.Context, req JobRequest) (*model.Job, error)`
- `func (*JobStorage) Finish (ctx context.Context, jobID string, req StatusRequest) (*model.Job, error)`
- `func (*JobStorage) Get (ctx context.Context, id string) (*model.Job, error)`
- `func (*JobStorage) Heartbeat (ctx context.Context, jobID,agentID string) error`
- `func (*JobStorage) List (ctx context.Context, filter ListFilter) ([]model.Job, error)`
- `func (*JobStorage) ReclaimExpired (ctx context.Context) (int64, error)`
- `func (*JobStorage) RecordCheckout (ctx context.Context, jobID string, req CheckoutRequest) error`
- `func (*RepositoryRuleStorage) Allowed (ctx context.Context, slug string) (bool, error)`
- `func (*RepositoryRuleStorage) Create (ctx context.Context, userID string, req RuleRequest) (*model.RepositoryRule, error)`
- `func (*RepositoryRuleStorage) Delete (ctx context.Context, id string) error`
- `func (*RepositoryRuleStorage) Get (ctx context.Context, id string) (*model.RepositoryRule, error)`
- `func (*RepositoryRuleStorage) List (ctx context.Context) ([]model.RepositoryRule, error)`
- `func (*RepositoryRuleStorage) ListActive (ctx context.Context) ([]model.RepositoryRule, error)`
- `func (*RepositoryRuleStorage) SetActive (ctx context.Context, id string, active bool) error`
- `func (*RepositoryStorage) Ensure (ctx context.Context, userID string, req RepositoryRequest) (*model.Repository, error)`
- `func (*RepositoryStorage) Get (ctx context.Context, id string) (*model.Repository, error)`
- `func (*RepositoryStorage) GetBySlug (ctx context.Context, slug string) (*model.Repository, error)`
- `func (*RepositoryStorage) List (ctx context.Context, limit int) ([]model.Repository, error)`
- `func (*SSHKeyStorage) Create (ctx context.Context, userID string, req SSHKeyRequest) (*model.SSHKey, error)`
- `func (*SSHKeyStorage) Delete (ctx context.Context, id string) error`
- `func (*SSHKeyStorage) List (ctx context.Context) ([]model.SSHKey, error)`
- `func (*SSHKeyStorage) ListForAgent (ctx context.Context) ([]model.SSHKey, error)`
- `func (*SSHKeyStorage) MarkUsed (ctx context.Context, ids []string)`
- `func (*SSHKeyStorage) SetActive (ctx context.Context, id string, active bool) error`
- `func (*SessionStorage) Create (ctx context.Context, userID string, req SessionRequest) (*model.Session, error)`
- `func (*SessionStorage) Get (ctx context.Context, id string) (*model.Session, error)`
- `func (*SessionStorage) GetByRefreshToken (ctx context.Context, token string) (*model.Session, error)`
- `func (*SessionStorage) Revoke (ctx context.Context, sessionID string) error`
- `func (*SessionStorage) RevokeByRefreshToken (ctx context.Context, token string) error`
- `func (*SessionStorage) Rotate (ctx context.Context, sessionID string) (*model.Session, error)`
- `func (*SessionStorage) Touch (ctx context.Context, sessionID string) error`
- `func (*SettingStorage) All () []SettingValue`
- `func (*SettingStorage) Bool (name string) bool`
- `func (*SettingStorage) Duration (name string) time.Duration`
- `func (*SettingStorage) Get (name string) string`
- `func (*SettingStorage) Int (name string) int64`
- `func (*SettingStorage) Load (ctx context.Context) error`
- `func (*SettingStorage) Reset (ctx context.Context, name string) error`
- `func (*SettingStorage) Set (ctx context.Context, name,value,userID string) error`
- `func (*UserStorage) Authenticate (ctx context.Context, email,password string) (*model.User, error)`
- `func (*UserStorage) Count (ctx context.Context) (int64, error)`
- `func (*UserStorage) CountAdmins (ctx context.Context) (int64, error)`
- `func (*UserStorage) Create (ctx context.Context, req CreateRequest) (*model.User, error)`
- `func (*UserStorage) EnsureAgent (ctx context.Context, agentID string) (*model.User, error)`
- `func (*UserStorage) Get (ctx context.Context, id string) (*model.User, error)`
- `func (*UserStorage) GetByEmail (ctx context.Context, email string) (*model.User, error)`
- `func (*UserStorage) List (ctx context.Context) ([]model.User, error)`
- `func (*UserStorage) SetFlags (ctx context.Context, id string, flags Flags) (*model.User, error)`

### DB

DB returns the shared connection pool for the atkins server.

An empty name selects ConnectionName. Naming it explicitly lets one
process host more than one atkins database, which is what the module
tests use to keep their fixtures apart: the platform caches pools by
connection name for the life of the process.

```go
func DB(ctx context.Context, name string) (*sqlx.DB, error)
```

### Migrate

Migrate applies SQL migrations from the given filesystem to the database.

```go
func Migrate(ctx context.Context, db *sqlx.DB, schema fs.FS) error
```

### NewJobLogStorage

NewJobLogStorage returns a JobLogStorage backed by the given pool.

```go
func NewJobLogStorage(db *sqlx.DB) *JobLogStorage
```

### NewJobStorage

NewJobStorage returns a JobStorage backed by the given pool.
Zero values for maxDepth and leaseTTL select the package defaults.

```go
func NewJobStorage(db *sqlx.DB, maxDepth int64, leaseTTL time.Duration) *JobStorage
```

### NewRepositoryRuleStorage

NewRepositoryRuleStorage returns a RepositoryRuleStorage.

```go
func NewRepositoryRuleStorage(db *sqlx.DB) *RepositoryRuleStorage
```

### NewRepositoryStorage

NewRepositoryStorage returns a RepositoryStorage backed by the pool.

```go
func NewRepositoryStorage(db *sqlx.DB) *RepositoryStorage
```

### NewSSHKeyStorage

NewSSHKeyStorage returns an SSHKeyStorage backed by the given pool.

```go
func NewSSHKeyStorage(db *sqlx.DB) *SSHKeyStorage
```

### NewSessionStorage

NewSessionStorage returns a SessionStorage backed by the given pool.
A zero ttl selects DefaultSessionTTL.

```go
func NewSessionStorage(db *sqlx.DB, ttl time.Duration) *SessionStorage
```

### NewSettingStorage

NewSettingStorage returns a SettingStorage backed by the given pool.

```go
func NewSettingStorage(db *sqlx.DB) *SettingStorage
```

### NewUserStorage

NewUserStorage returns a UserStorage backed by the given pool.

```go
func NewUserStorage(db *sqlx.DB) *UserStorage
```

### Append

Append adds a chunk of output to a job.

Chunks are numbered per job so the page can render them in the order
the agent produced them, without relying on timestamp resolution.

```go
func (*JobLogStorage) Append(ctx context.Context, jobID, stream, content string) error
```

### List

List returns the log chunks for a job, in the order they arrived.

```go
func (*JobLogStorage) List(ctx context.Context, jobID string) ([]model.JobLog, error)
```

### Claim

Claim takes an exclusive lease on the oldest pending job an agent can
run, or returns sql.ErrNoRows when the queue holds nothing for it.

The claim runs in a transaction and the UPDATE re-asserts
`status = 'pending'`, so two agents racing for the same row produce
one winner and one empty poll rather than a job running twice.

```go
func (*JobStorage) Claim(ctx context.Context, agentID string, labels []string) (*model.Job, error)
```

### Create

Create records a dispatched job.

Depth and root are derived from the parent rather than trusted from
the client: a job that lies about its depth would sidestep the
runaway-dispatch limit.

```go
func (*JobStorage) Create(ctx context.Context, req JobRequest) (*model.Job, error)
```

### Finish

Finish settles a job into a terminal state.

The update is guarded on the job not already being terminal so a late
report from a timed-out agent can't overwrite the outcome the server
already recorded.

```go
func (*JobStorage) Finish(ctx context.Context, jobID string, req StatusRequest) (*model.Job, error)
```

### Get

Get returns a job by ID.

```go
func (*JobStorage) Get(ctx context.Context, id string) (*model.Job, error)
```

### Heartbeat

Heartbeat extends the lease on a claimed job. Only the agent holding
the job may extend it.

```go
func (*JobStorage) Heartbeat(ctx context.Context, jobID, agentID string) error
```

### List

List returns jobs matching the filter, newest first.

```go
func (*JobStorage) List(ctx context.Context, filter ListFilter) ([]model.Job, error)
```

### ReclaimExpired

ReclaimExpired marks running jobs whose lease has lapsed as timed out
and returns how many were reclaimed. The server runs this on a ticker
so an agent that disappears mid-job doesn't strand its work.

```go
func (*JobStorage) ReclaimExpired(ctx context.Context) (int64, error)
```

### RecordCheckout

RecordCheckout stores what an agent checked out for a job.

It is guarded on the job still running, so a late report from an agent
whose lease was already swept cannot rewrite the checkout of the run
that replaced it.

```go
func (*JobStorage) RecordCheckout(ctx context.Context, jobID string, req CheckoutRequest) error
```

### Allowed

Allowed reports whether a repository slug satisfies the allowlist.

The caller decides whether the allowlist applies at all; this only
answers "does any active rule match", which is false for an empty
list. In allowlist mode that is the intended answer: an operator who
turns the policy on without writing a rule has asked for nothing to
run.

```go
func (*RepositoryRuleStorage) Allowed(ctx context.Context, slug string) (bool, error)
```

### Create

Create adds an allowlist rule.

```go
func (*RepositoryRuleStorage) Create(ctx context.Context, userID string, req RuleRequest) (*model.RepositoryRule, error)
```

### Delete

Delete soft-deletes a rule.

```go
func (*RepositoryRuleStorage) Delete(ctx context.Context, id string) error
```

### Get

Get returns a rule by ID.

```go
func (*RepositoryRuleStorage) Get(ctx context.Context, id string) (*model.RepositoryRule, error)
```

### List

List returns the rules, newest first.

```go
func (*RepositoryRuleStorage) List(ctx context.Context) ([]model.RepositoryRule, error)
```

### ListActive

ListActive returns only the rules that are in force.

```go
func (*RepositoryRuleStorage) ListActive(ctx context.Context) ([]model.RepositoryRule, error)
```

### SetActive

SetActive enables or disables a rule without deleting it.

```go
func (*RepositoryRuleStorage) SetActive(ctx context.Context, id string, active bool) error
```

### Ensure

Ensure returns the repository for the given remote, creating it on
first sight.

Dispatch is the only caller and it happens on every atkins run, so
this is deliberately upsert-shaped: the client should never have to
register a repository before using it. The slug is the identity, so
the same repository cloned over ssh and https is one row.

```go
func (*RepositoryStorage) Ensure(ctx context.Context, userID string, req RepositoryRequest) (*model.Repository, error)
```

### Get

Get returns a repository by ID.

```go
func (*RepositoryStorage) Get(ctx context.Context, id string) (*model.Repository, error)
```

### GetBySlug

GetBySlug returns a repository by its normalized slug.

```go
func (*RepositoryStorage) GetBySlug(ctx context.Context, slug string) (*model.Repository, error)
```

### List

List returns live repositories, newest first.

```go
func (*RepositoryStorage) List(ctx context.Context, limit int) ([]model.Repository, error)
```

### Create

Create stores a key, deriving its public half and fingerprint so an
operator never has to paste those separately.

```go
func (*SSHKeyStorage) Create(ctx context.Context, userID string, req SSHKeyRequest) (*model.SSHKey, error)
```

### Delete

Delete soft-deletes a key.

```go
func (*SSHKeyStorage) Delete(ctx context.Context, id string) error
```

### List

List returns the keys, newest first, including private material.
Callers rendering these to a user must project them first.

```go
func (*SSHKeyStorage) List(ctx context.Context) ([]model.SSHKey, error)
```

### ListForAgent

ListForAgent returns the active keys an agent should install.

```go
func (*SSHKeyStorage) ListForAgent(ctx context.Context) ([]model.SSHKey, error)
```

### MarkUsed

MarkUsed records that an agent installed the key. Best effort.

```go
func (*SSHKeyStorage) MarkUsed(ctx context.Context, ids []string)
```

### SetActive

SetActive enables or disables a key.

```go
func (*SSHKeyStorage) SetActive(ctx context.Context, id string, active bool) error
```

### Create

Create issues a new session with a fresh refresh token.

```go
func (*SessionStorage) Create(ctx context.Context, userID string, req SessionRequest) (*model.Session, error)
```

### Get

Get returns a session by ID regardless of its state.

```go
func (*SessionStorage) Get(ctx context.Context, id string) (*model.Session, error)
```

### GetByRefreshToken

GetByRefreshToken returns a live session for the given refresh token.
A revoked or expired session is reported as such rather than as a
missing row, so the client can tell "log in again" from "bad token".

```go
func (*SessionStorage) GetByRefreshToken(ctx context.Context, token string) (*model.Session, error)
```

### Revoke

Revoke marks a session as logged out. Revoking an already-revoked or
unknown session is not an error: logout is idempotent.

```go
func (*SessionStorage) Revoke(ctx context.Context, sessionID string) error
```

### RevokeByRefreshToken

RevokeByRefreshToken revokes the session holding the given token.

```go
func (*SessionStorage) RevokeByRefreshToken(ctx context.Context, token string) error
```

### Rotate

Rotate replaces the refresh token on a session and extends its expiry.

The old token stops working the moment this returns: a refresh token
is single-use, so a leaked one is detectable (the legitimate client's
next refresh fails) and short-lived.

```go
func (*SessionStorage) Rotate(ctx context.Context, sessionID string) (*model.Session, error)
```

### Touch

Touch records that a session was seen. Best-effort; failures here are
not worth failing a request over.

```go
func (*SessionStorage) Touch(ctx context.Context, sessionID string) error
```

### All

All returns every setting with its effective value, in registry
order, so an admin sees the whole surface rather than only what has
been overridden.

```go
func (*SettingStorage) All() []SettingValue
```

### Bool

Bool returns a setting parsed as a boolean.

```go
func (*SettingStorage) Bool(name string) bool
```

### Duration

Duration returns a setting parsed as a duration.

```go
func (*SettingStorage) Duration(name string) time.Duration
```

### Get

Get returns the effective value for a setting: the stored override,
or the registry default.

```go
func (*SettingStorage) Get(name string) string
```

### Int

Int returns a setting parsed as a whole number.

```go
func (*SettingStorage) Int(name string) int64
```

### Load

Load fills the cache from the database.

```go
func (*SettingStorage) Load(ctx context.Context) error
```

### Reset

Reset drops an override, returning the setting to its default.

```go
func (*SettingStorage) Reset(ctx context.Context, name string) error
```

### Set

Set stores an override after validating it against the registry.

```go
func (*SettingStorage) Set(ctx context.Context, name, value, userID string) error
```

### Authenticate

Authenticate verifies an email/password pair and returns the user.

Every failure mode collapses into model.ErrInvalidCredentials except
an explicitly deactivated account, which is worth telling the caller
about since retrying the password won't help.

```go
func (*UserStorage) Authenticate(ctx context.Context, email, password string) (*model.User, error)
```

### Count

Count returns the number of live human users.

Agents are excluded on purpose. It is what decides whether the next
registration is the bootstrap one, and an agent enrolling first
should neither claim that slot nor close the door behind it.

```go
func (*UserStorage) Count(ctx context.Context) (int64, error)
```

### CountAdmins

CountAdmins returns how many active admins remain. It exists so the
API can refuse to remove the last one and lock everybody out.

```go
func (*UserStorage) CountAdmins(ctx context.Context) (int64, error)
```

### Create

Create inserts a user with a bcrypt-hashed password.

The first user on a fresh server is made an admin: a self-hosted
instance has nobody to grant the first grant, so the bootstrap has to
come from somewhere.

```go
func (*UserStorage) Create(ctx context.Context, req CreateRequest) (*model.User, error)
```

### EnsureAgent

EnsureAgent returns the account for an enrolling agent, creating it
on first contact.

Agents authenticate with a shared enrolment token rather than a
password, so the account gets a random one it never learns: there is
no password login path for an agent to leak.

```go
func (*UserStorage) EnsureAgent(ctx context.Context, agentID string) (*model.User, error)
```

### Get

Get returns a user by ID. Soft-deleted users are not returned.

```go
func (*UserStorage) Get(ctx context.Context, id string) (*model.User, error)
```

### GetByEmail

GetByEmail returns a user by email. Soft-deleted users are not returned.

```go
func (*UserStorage) GetByEmail(ctx context.Context, email string) (*model.User, error)
```

### List

List returns live users, newest first.

```go
func (*UserStorage) List(ctx context.Context) ([]model.User, error)
```

### SetFlags

SetFlags updates a user's administrative state.

```go
func (*UserStorage) SetFlags(ctx context.Context, id string, flags Flags) (*model.User, error)
```

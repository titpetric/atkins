# Package ./server/model

```go
import (
	"github.com/titpetric/atkins/server/model"
}
```

## Types

```go
// Job generated for db table `job`.
type Job struct {
	// ID
	ID string `db:"id" json:"id"`

	// Parent ID
	ParentID string `db:"parent_id" json:"parent_id"`

	// Root ID
	RootID string `db:"root_id" json:"root_id"`

	// Depth
	Depth int64 `db:"depth" json:"depth"`

	// Repository ID
	RepositoryID string `db:"repository_id" json:"repository_id"`

	// User ID
	UserID string `db:"user_id" json:"user_id"`

	// Working Directory
	WorkingDirectory string `db:"working_directory" json:"working_directory"`

	// Command
	Command string `db:"command" json:"command"`

	// Branch
	Branch string `db:"branch" json:"branch"`

	// Revision
	Revision string `db:"revision" json:"revision"`

	// Labels
	Labels string `db:"labels" json:"labels"`

	// Params
	Params string `db:"params" json:"params"`

	// Status
	Status string `db:"status" json:"status"`

	// Exit Code
	ExitCode int64 `db:"exit_code" json:"exit_code"`

	// Error
	Error string `db:"error" json:"error"`

	// Agent ID
	AgentID string `db:"agent_id" json:"agent_id"`

	// Lease Expires At
	LeaseExpiresAt *time.Time `db:"lease_expires_at" json:"lease_expires_at"`

	// Started At
	StartedAt *time.Time `db:"started_at" json:"started_at"`

	// Finished At
	FinishedAt *time.Time `db:"finished_at" json:"finished_at"`

	// Created At
	CreatedAt *time.Time `db:"created_at" json:"created_at"`

	// Updated At
	UpdatedAt *time.Time `db:"updated_at" json:"updated_at"`
}
```

```go
// JobLog generated for db table `job_log`.
// Job Log.
type JobLog struct {
	// ID
	ID string `db:"id" json:"id"`

	// Job ID
	JobID string `db:"job_id" json:"job_id"`

	// Seq
	Seq int64 `db:"seq" json:"seq"`

	// Stream
	Stream string `db:"stream" json:"stream"`

	// Content
	Content string `db:"content" json:"content"`

	// Created At
	CreatedAt *time.Time `db:"created_at" json:"created_at"`
}
```

```go
// JobStatus is the lifecycle state of a job. The values mirror the
// CHECK constraint in schema/job.up.sql; adding one means adding it in
// both places.
type JobStatus = string
```

```go
// Migrations generated for db table `migrations`.
type Migrations struct {
	// Project
	Project string `db:"project" json:"project"`

	// Filename
	Filename string `db:"filename" json:"filename"`

	// Statement Index
	StatementIndex int64 `db:"statement_index" json:"statement_index"`

	// Status
	Status string `db:"status" json:"status"`
}
```

```go
// QueryConfig is a function-chaining SQL statement type.
type QueryConfig struct {
	Table       string
	Columns     []string
	Where       string
	OrderBy     string
	LimitStart  int
	LimitOffset int
	Statement   string
}
```

```go
// QueryOption is implemented by each data model type.
type QueryOption interface {
	WithTable(name string) QueryOption
	WithColumns(cols []string) QueryOption
	WithWhere(clause string) QueryOption
	WithOrderBy(clause string) QueryOption
	WithLimit(start, offset int) QueryOption
	WithStatement(stmt string) QueryOption
}
```

```go
// Repository generated for db table `repository`.
type Repository struct {
	// ID
	ID string `db:"id" json:"id"`

	// Slug
	Slug string `db:"slug" json:"slug"`

	// Remote URL
	RemoteURL string `db:"remote_url" json:"remote_url"`

	// Default Branch
	DefaultBranch string `db:"default_branch" json:"default_branch"`

	// Created By User ID
	CreatedByUserID string `db:"created_by_user_id" json:"created_by_user_id"`

	// Is Active
	IsActive bool `db:"is_active" json:"is_active"`

	// Created At
	CreatedAt *time.Time `db:"created_at" json:"created_at"`

	// Updated At
	UpdatedAt *time.Time `db:"updated_at" json:"updated_at"`

	// Deleted At
	DeletedAt *time.Time `db:"deleted_at" json:"deleted_at"`
}
```

```go
// RepositoryPolicy decides which repositories agents may build.
type RepositoryPolicy = string
```

```go
// RepositoryRule generated for db table `repository_rule`.
// Repository Rule.
type RepositoryRule struct {
	// ID
	ID string `db:"id" json:"id"`

	// Pattern
	Pattern string `db:"pattern" json:"pattern"`

	// Description
	Description string `db:"description" json:"description"`

	// Is Active
	IsActive bool `db:"is_active" json:"is_active"`

	// Created By User ID
	CreatedByUserID string `db:"created_by_user_id" json:"created_by_user_id"`

	// Created At
	CreatedAt *time.Time `db:"created_at" json:"created_at"`

	// Updated At
	UpdatedAt *time.Time `db:"updated_at" json:"updated_at"`

	// Deleted At
	DeletedAt *time.Time `db:"deleted_at" json:"deleted_at"`
}
```

```go
// SSHKey generated for db table `ssh_key`.
// SSH Key.
type SSHKey struct {
	// ID
	ID string `db:"id" json:"id"`

	// Name
	Name string `db:"name" json:"name"`

	// Host
	Host string `db:"host" json:"host"`

	// Private Key
	PrivateKey string `db:"private_key" json:"private_key"`

	// Public Key
	PublicKey string `db:"public_key" json:"public_key"`

	// Fingerprint
	Fingerprint string `db:"fingerprint" json:"fingerprint"`

	// Known Hosts
	KnownHosts string `db:"known_hosts" json:"known_hosts"`

	// Is Active
	IsActive bool `db:"is_active" json:"is_active"`

	// Created By User ID
	CreatedByUserID string `db:"created_by_user_id" json:"created_by_user_id"`

	// Last Used At
	LastUsedAt *time.Time `db:"last_used_at" json:"last_used_at"`

	// Created At
	CreatedAt *time.Time `db:"created_at" json:"created_at"`

	// Updated At
	UpdatedAt *time.Time `db:"updated_at" json:"updated_at"`

	// Deleted At
	DeletedAt *time.Time `db:"deleted_at" json:"deleted_at"`
}
```

```go
// Session generated for db table `session`.
type Session struct {
	// ID
	ID string `db:"id" json:"id"`

	// User ID
	UserID string `db:"user_id" json:"user_id"`

	// Refresh Token
	RefreshToken string `db:"refresh_token" json:"refresh_token"`

	// Hostname
	Hostname string `db:"hostname" json:"hostname"`

	// User Agent
	UserAgent string `db:"user_agent" json:"user_agent"`

	// Remote Addr
	RemoteAddr string `db:"remote_addr" json:"remote_addr"`

	// Last Seen At
	LastSeenAt *time.Time `db:"last_seen_at" json:"last_seen_at"`

	// Expires At
	ExpiresAt *time.Time `db:"expires_at" json:"expires_at"`

	// Revoked At
	RevokedAt *time.Time `db:"revoked_at" json:"revoked_at"`

	// Created At
	CreatedAt *time.Time `db:"created_at" json:"created_at"`

	// Updated At
	UpdatedAt *time.Time `db:"updated_at" json:"updated_at"`
}
```

```go
// Setting generated for db table `setting`.
type Setting struct {
	// Name
	Name string `db:"name" json:"name"`

	// Value
	Value string `db:"value" json:"value"`

	// Updated By User ID
	UpdatedByUserID string `db:"updated_by_user_id" json:"updated_by_user_id"`

	// Created At
	CreatedAt *time.Time `db:"created_at" json:"created_at"`

	// Updated At
	UpdatedAt *time.Time `db:"updated_at" json:"updated_at"`
}
```

```go
// SettingDefinition describes one configurable value.
type SettingDefinition struct {
	Name        string      `json:"name"`
	Kind        SettingKind `json:"kind"`
	Default     string      `json:"default"`
	Description string      `json:"description"`

	// Values enumerates the accepted values for KindEnum.
	Values []string `json:"values,omitempty"`
}
```

```go
// SettingKind is how a setting's value is validated and parsed.
type SettingKind string
```

```go
// User generated for db table `user`.
type User struct {
	// ID
	ID string `db:"id" json:"id"`

	// Email
	Email string `db:"email" json:"email"`

	// Username
	Username string `db:"username" json:"username"`

	// Full Name
	FullName string `db:"full_name" json:"full_name"`

	// Password
	Password string `db:"password" json:"password"`

	// Is Admin
	IsAdmin bool `db:"is_admin" json:"is_admin"`

	// Is Active
	IsActive bool `db:"is_active" json:"is_active"`

	// Created At
	CreatedAt *time.Time `db:"created_at" json:"created_at"`

	// Updated At
	UpdatedAt *time.Time `db:"updated_at" json:"updated_at"`

	// Deleted At
	DeletedAt *time.Time `db:"deleted_at" json:"deleted_at"`

	// Is Agent
	IsAgent bool `db:"is_agent" json:"is_agent"`
}
```

## Consts

```go
// JobLogTable is the name of the table in the DB.
const JobLogTable = "`job_log`"
```

```go
// JobTable is the name of the table in the DB.
const JobTable = "`job`"
```

```go
// MigrationsTable is the name of the table in the DB.
const MigrationsTable = "`migrations`"
```

```go
// RepositoryRuleTable is the name of the table in the DB.
const RepositoryRuleTable = "`repository_rule`"
```

```go
// RepositoryTable is the name of the table in the DB.
const RepositoryTable = "`repository`"
```

```go
// SSHKeyTable is the name of the table in the DB.
const SSHKeyTable = "`ssh_key`"
```

```go
// SessionTable is the name of the table in the DB.
const SessionTable = "`session`"
```

```go
// SettingTable is the name of the table in the DB.
const SettingTable = "`setting`"
```

```go
// UserTable is the name of the table in the DB.
const UserTable = "`user`"
```

```go
// Setting kinds.
const (
	KindString   SettingKind = "string"
	KindBool     SettingKind = "bool"
	KindInt      SettingKind = "int"
	KindDuration SettingKind = "duration"
	KindEnum     SettingKind = "enum"
)
```

```go
// Setting names. These are the configuration an admin can change at
// runtime; everything else is a process flag and needs a restart.
const (
	// SettingRepositoryPolicy is "open" or "allowlist".
	SettingRepositoryPolicy = "repository.policy"

	// SettingRegistrationOpen decides whether anyone may create an
	// account. The first account is always allowed regardless.
	SettingRegistrationOpen = "registration.open"

	// SettingJobMaxDepth bounds how deep a job may dispatch children.
	SettingJobMaxDepth = "job.max_depth"

	// SettingJobLeaseTTL is how long an agent may hold a job without a
	// heartbeat before the server reclaims it.
	SettingJobLeaseTTL = "job.lease_ttl"

	// SettingJobRetention is how long finished jobs and their output
	// are kept. Zero keeps them forever.
	SettingJobRetention = "job.retention"
)
```

```go
// Job status values.
// A job starts pending. An agent claiming it moves it to running and
// holds a lease. From running it settles into exactly one terminal
// state: passed, failed, timeout or cancelled.
const (
	// JobStatusPending is a queued job that no agent has claimed.
	JobStatusPending JobStatus = "pending"

	// JobStatusRunning is a job claimed by an agent holding a lease.
	JobStatusRunning JobStatus = "running"

	// JobStatusPassed is a job whose command exited zero.
	JobStatusPassed JobStatus = "passed"

	// JobStatusFailed is a job whose command exited non-zero.
	JobStatusFailed JobStatus = "failed"

	// JobStatusTimeout is a job whose agent lease expired without a
	// terminal report. The server reclaims these for retry.
	JobStatusTimeout JobStatus = "timeout"

	// JobStatusCancelled is a job stopped before it settled.
	JobStatusCancelled JobStatus = "cancelled"
)
```

```go
// Repository policy values.
const (
	// PolicyOpen builds any repository a logged-in user dispatches.
	// It is the default: a personal instance has no third party to
	// defend against, and requiring a rule before the first run would
	// make the tool feel broken.
	PolicyOpen RepositoryPolicy = "open"

	// PolicyAllowlist builds only repositories matching an active
	// rule. With no rules configured, nothing runs — which is the
	// point of turning it on.
	PolicyAllowlist RepositoryPolicy = "allowlist"
)
```

## Vars

```go
// JobFields is a list of all columns in the DB table.
var JobFields = []string{"id", "parent_id", "root_id", "depth", "repository_id", "user_id", "working_directory", "command", "branch", "revision", "labels", "params", "status", "exit_code", "error", "agent_id", "lease_expires_at", "started_at", "finished_at", "created_at", "updated_at"}
```

```go
// JobLogFields is a list of all columns in the DB table.
var JobLogFields = []string{"id", "job_id", "seq", "stream", "content", "created_at"}
```

```go
// JobLogPrimaryFields are the primary key fields in the DB table.
var JobLogPrimaryFields = []string{"id"}
```

```go
// JobPrimaryFields are the primary key fields in the DB table.
var JobPrimaryFields = []string{"id"}
```

```go
// MigrationsFields is a list of all columns in the DB table.
var MigrationsFields = []string{"project", "filename", "statement_index", "status"}
```

```go
// MigrationsPrimaryFields are the primary key fields in the DB table.
var MigrationsPrimaryFields = []string{"project", "filename"}
```

```go
// RepositoryFields is a list of all columns in the DB table.
var RepositoryFields = []string{"id", "slug", "remote_url", "default_branch", "created_by_user_id", "is_active", "created_at", "updated_at", "deleted_at"}
```

```go
// RepositoryPrimaryFields are the primary key fields in the DB table.
var RepositoryPrimaryFields = []string{"id"}
```

```go
// RepositoryRuleFields is a list of all columns in the DB table.
var RepositoryRuleFields = []string{"id", "pattern", "description", "is_active", "created_by_user_id", "created_at", "updated_at", "deleted_at"}
```

```go
// RepositoryRulePrimaryFields are the primary key fields in the DB table.
var RepositoryRulePrimaryFields = []string{"id"}
```

```go
// SSHKeyFields is a list of all columns in the DB table.
var SSHKeyFields = []string{"id", "name", "host", "private_key", "public_key", "fingerprint", "known_hosts", "is_active", "created_by_user_id", "last_used_at", "created_at", "updated_at", "deleted_at"}
```

```go
// SSHKeyPrimaryFields are the primary key fields in the DB table.
var SSHKeyPrimaryFields = []string{"id"}
```

```go
// SessionFields is a list of all columns in the DB table.
var SessionFields = []string{"id", "user_id", "refresh_token", "hostname", "user_agent", "remote_addr", "last_seen_at", "expires_at", "revoked_at", "created_at", "updated_at"}
```

```go
// SessionPrimaryFields are the primary key fields in the DB table.
var SessionPrimaryFields = []string{"id"}
```

```go
// SettingFields is a list of all columns in the DB table.
var SettingFields = []string{"name", "value", "updated_by_user_id", "created_at", "updated_at"}
```

```go
// SettingPrimaryFields are the primary key fields in the DB table.
var SettingPrimaryFields = []string{"name"}
```

```go
// UserFields is a list of all columns in the DB table.
var UserFields = []string{"id", "email", "username", "full_name", "password", "is_admin", "is_active", "created_at", "updated_at", "deleted_at", "is_agent"}
```

```go
// UserPrimaryFields are the primary key fields in the DB table.
var UserPrimaryFields = []string{"id"}
```

```go
// Errors returned by the storage layer. Handlers map these onto status
// codes; anything not listed here is a 5xx as far as the API is concerned.
var (
	// ErrInvalidCredentials is returned when the email/password pair
	// does not match an active user. It is deliberately the same error
	// for "no such user" and "wrong password" so the login endpoint
	// cannot be used to enumerate accounts.
	ErrInvalidCredentials = errors.New("invalid credentials")

	// ErrUserInactive is returned when a user exists but has been
	// deactivated. Login is refused.
	ErrUserInactive = errors.New("user is not active")

	// ErrEmailTaken is returned when registering an existing email.
	ErrEmailTaken = errors.New("email already registered")

	// ErrUsernameTaken is returned when registering an existing username.
	ErrUsernameTaken = errors.New("username already taken")

	// ErrRegistrationClosed is returned when registration is disabled
	// and at least one user already exists.
	ErrRegistrationClosed = errors.New("registration is closed")

	// ErrSessionExpired is returned when a refresh token is past its
	// expiry. The client has to log in again.
	ErrSessionExpired = errors.New("session expired")

	// ErrSessionRevoked is returned when a refresh token belongs to a
	// session that was logged out.
	ErrSessionRevoked = errors.New("session revoked")

	// ErrInvalidRepository is returned when a dispatch request carries
	// no usable git remote.
	ErrInvalidRepository = errors.New("invalid repository")

	// ErrMaxDepthExceeded is returned when a job tries to dispatch a
	// child past the configured nesting limit. It's the backstop
	// against a pipeline that dispatches itself forever.
	ErrMaxDepthExceeded = errors.New("maximum job depth exceeded")

	// ErrRepositoryNotAllowed is returned when the server runs an
	// allowlist policy and no rule matches the repository.
	ErrRepositoryNotAllowed = errors.New("repository is not on the allowlist")

	// ErrInvalidPattern is returned for an empty allowlist pattern.
	ErrInvalidPattern = errors.New("pattern is required")

	// ErrRuleNotFound is returned when an allowlist rule does not
	// exist, or has already been deleted.
	ErrRuleNotFound = errors.New("repository rule not found")

	// ErrSSHKeyNotFound is returned when an ssh key does not exist.
	ErrSSHKeyNotFound = errors.New("ssh key not found")

	// ErrInvalidSSHKey is returned when a submitted key is not a
	// usable private key.
	ErrInvalidSSHKey = errors.New("not a usable ssh private key")

	// ErrForbidden is returned when a user is authenticated but not
	// permitted to perform the action.
	ErrForbidden = errors.New("insufficient permissions")

	// ErrInvalidAgentToken is returned when agent enrolment presents
	// the wrong token, or the server has no enrolment token set.
	ErrInvalidAgentToken = errors.New("invalid agent enrolment token")
)
```

## Function symbols

- `func LookupSetting (name string) (SettingDefinition, bool)`
- `func MatchRepository (pattern,slug string) bool`
- `func RepositorySlug (remoteURL string) string`
- `func SettingDefinitions () []SettingDefinition`
- `func TerminalJobStatus (status JobStatus) bool`
- `func ValidJobStatus (status JobStatus) bool`
- `func ValidRepositoryPolicy (policy RepositoryPolicy) bool`
- `func WithColumns (cols []string) QueryOption`
- `func WithLimit (start,offset int) QueryOption`
- `func WithOrderBy (clause string) QueryOption`
- `func WithStatement (stmt string) QueryOption`
- `func WithTable (name string) QueryOption`
- `func WithWhere (clause string) QueryOption`
- `func (*Job) Delete (opts ...QueryOption) string`
- `func (*Job) GetAgentID () string`
- `func (*Job) GetBranch () string`
- `func (*Job) GetCommand () string`
- `func (*Job) GetCreatedAt () *time.Time`
- `func (*Job) GetDepth () int64`
- `func (*Job) GetError () string`
- `func (*Job) GetExitCode () int64`
- `func (*Job) GetFinishedAt () *time.Time`
- `func (*Job) GetID () string`
- `func (*Job) GetLabels () string`
- `func (*Job) GetLeaseExpiresAt () *time.Time`
- `func (*Job) GetParams () string`
- `func (*Job) GetParentID () string`
- `func (*Job) GetRepositoryID () string`
- `func (*Job) GetRevision () string`
- `func (*Job) GetRootID () string`
- `func (*Job) GetStartedAt () *time.Time`
- `func (*Job) GetStatus () string`
- `func (*Job) GetUpdatedAt () *time.Time`
- `func (*Job) GetUserID () string`
- `func (*Job) GetWorkingDirectory () string`
- `func (*Job) Insert (opts ...QueryOption) string`
- `func (*Job) IsTerminal () bool`
- `func (*Job) Select (opts ...QueryOption) string`
- `func (*Job) SetAgentID (val string)`
- `func (*Job) SetBranch (val string)`
- `func (*Job) SetCommand (val string)`
- `func (*Job) SetCreatedAt (stamp time.Time)`
- `func (*Job) SetDepth (val int64)`
- `func (*Job) SetError (val string)`
- `func (*Job) SetExitCode (val int64)`
- `func (*Job) SetFinishedAt (stamp time.Time)`
- `func (*Job) SetID (val string)`
- `func (*Job) SetLabels (val string)`
- `func (*Job) SetLeaseExpiresAt (stamp time.Time)`
- `func (*Job) SetParams (val string)`
- `func (*Job) SetParentID (val string)`
- `func (*Job) SetRepositoryID (val string)`
- `func (*Job) SetRevision (val string)`
- `func (*Job) SetRootID (val string)`
- `func (*Job) SetStartedAt (stamp time.Time)`
- `func (*Job) SetStatus (val string)`
- `func (*Job) SetUpdatedAt (stamp time.Time)`
- `func (*Job) SetUserID (val string)`
- `func (*Job) SetWorkingDirectory (val string)`
- `func (*Job) Update (opts ...QueryOption) string`
- `func (*JobLog) Delete (opts ...QueryOption) string`
- `func (*JobLog) GetContent () string`
- `func (*JobLog) GetCreatedAt () *time.Time`
- `func (*JobLog) GetID () string`
- `func (*JobLog) GetJobID () string`
- `func (*JobLog) GetSeq () int64`
- `func (*JobLog) GetStream () string`
- `func (*JobLog) Insert (opts ...QueryOption) string`
- `func (*JobLog) Select (opts ...QueryOption) string`
- `func (*JobLog) SetContent (val string)`
- `func (*JobLog) SetCreatedAt (stamp time.Time)`
- `func (*JobLog) SetID (val string)`
- `func (*JobLog) SetJobID (val string)`
- `func (*JobLog) SetSeq (val int64)`
- `func (*JobLog) SetStream (val string)`
- `func (*JobLog) Update (opts ...QueryOption) string`
- `func (*Migrations) Delete (opts ...QueryOption) string`
- `func (*Migrations) GetFilename () string`
- `func (*Migrations) GetProject () string`
- `func (*Migrations) GetStatementIndex () int64`
- `func (*Migrations) GetStatus () string`
- `func (*Migrations) Insert (opts ...QueryOption) string`
- `func (*Migrations) Select (opts ...QueryOption) string`
- `func (*Migrations) SetFilename (val string)`
- `func (*Migrations) SetProject (val string)`
- `func (*Migrations) SetStatementIndex (val int64)`
- `func (*Migrations) SetStatus (val string)`
- `func (*Migrations) Update (opts ...QueryOption) string`
- `func (*QueryConfig) Apply (opts ...QueryOption) *QueryConfig`
- `func (*QueryConfig) WithColumns (cols []string) QueryOption`
- `func (*QueryConfig) WithLimit (start,offset int) QueryOption`
- `func (*QueryConfig) WithOrderBy (clause string) QueryOption`
- `func (*QueryConfig) WithStatement (stmt string) QueryOption`
- `func (*QueryConfig) WithTable (name string) QueryOption`
- `func (*QueryConfig) WithWhere (clause string) QueryOption`
- `func (*Repository) Delete (opts ...QueryOption) string`
- `func (*Repository) GetCreatedAt () *time.Time`
- `func (*Repository) GetCreatedByUserID () string`
- `func (*Repository) GetDefaultBranch () string`
- `func (*Repository) GetDeletedAt () *time.Time`
- `func (*Repository) GetID () string`
- `func (*Repository) GetIsActive () bool`
- `func (*Repository) GetRemoteURL () string`
- `func (*Repository) GetSlug () string`
- `func (*Repository) GetUpdatedAt () *time.Time`
- `func (*Repository) Insert (opts ...QueryOption) string`
- `func (*Repository) Select (opts ...QueryOption) string`
- `func (*Repository) SetCreatedAt (stamp time.Time)`
- `func (*Repository) SetCreatedByUserID (val string)`
- `func (*Repository) SetDefaultBranch (val string)`
- `func (*Repository) SetDeletedAt (stamp time.Time)`
- `func (*Repository) SetID (val string)`
- `func (*Repository) SetIsActive (val bool)`
- `func (*Repository) SetRemoteURL (val string)`
- `func (*Repository) SetSlug (val string)`
- `func (*Repository) SetUpdatedAt (stamp time.Time)`
- `func (*Repository) Update (opts ...QueryOption) string`
- `func (*RepositoryRule) Delete (opts ...QueryOption) string`
- `func (*RepositoryRule) GetCreatedAt () *time.Time`
- `func (*RepositoryRule) GetCreatedByUserID () string`
- `func (*RepositoryRule) GetDeletedAt () *time.Time`
- `func (*RepositoryRule) GetDescription () string`
- `func (*RepositoryRule) GetID () string`
- `func (*RepositoryRule) GetIsActive () bool`
- `func (*RepositoryRule) GetPattern () string`
- `func (*RepositoryRule) GetUpdatedAt () *time.Time`
- `func (*RepositoryRule) Insert (opts ...QueryOption) string`
- `func (*RepositoryRule) Select (opts ...QueryOption) string`
- `func (*RepositoryRule) SetCreatedAt (stamp time.Time)`
- `func (*RepositoryRule) SetCreatedByUserID (val string)`
- `func (*RepositoryRule) SetDeletedAt (stamp time.Time)`
- `func (*RepositoryRule) SetDescription (val string)`
- `func (*RepositoryRule) SetID (val string)`
- `func (*RepositoryRule) SetIsActive (val bool)`
- `func (*RepositoryRule) SetPattern (val string)`
- `func (*RepositoryRule) SetUpdatedAt (stamp time.Time)`
- `func (*RepositoryRule) Update (opts ...QueryOption) string`
- `func (*SSHKey) Delete (opts ...QueryOption) string`
- `func (*SSHKey) GetCreatedAt () *time.Time`
- `func (*SSHKey) GetCreatedByUserID () string`
- `func (*SSHKey) GetDeletedAt () *time.Time`
- `func (*SSHKey) GetFingerprint () string`
- `func (*SSHKey) GetHost () string`
- `func (*SSHKey) GetID () string`
- `func (*SSHKey) GetIsActive () bool`
- `func (*SSHKey) GetKnownHosts () string`
- `func (*SSHKey) GetLastUsedAt () *time.Time`
- `func (*SSHKey) GetName () string`
- `func (*SSHKey) GetPrivateKey () string`
- `func (*SSHKey) GetPublicKey () string`
- `func (*SSHKey) GetUpdatedAt () *time.Time`
- `func (*SSHKey) Insert (opts ...QueryOption) string`
- `func (*SSHKey) Select (opts ...QueryOption) string`
- `func (*SSHKey) SetCreatedAt (stamp time.Time)`
- `func (*SSHKey) SetCreatedByUserID (val string)`
- `func (*SSHKey) SetDeletedAt (stamp time.Time)`
- `func (*SSHKey) SetFingerprint (val string)`
- `func (*SSHKey) SetHost (val string)`
- `func (*SSHKey) SetID (val string)`
- `func (*SSHKey) SetIsActive (val bool)`
- `func (*SSHKey) SetKnownHosts (val string)`
- `func (*SSHKey) SetLastUsedAt (stamp time.Time)`
- `func (*SSHKey) SetName (val string)`
- `func (*SSHKey) SetPrivateKey (val string)`
- `func (*SSHKey) SetPublicKey (val string)`
- `func (*SSHKey) SetUpdatedAt (stamp time.Time)`
- `func (*SSHKey) Update (opts ...QueryOption) string`
- `func (*Session) Delete (opts ...QueryOption) string`
- `func (*Session) GetCreatedAt () *time.Time`
- `func (*Session) GetExpiresAt () *time.Time`
- `func (*Session) GetHostname () string`
- `func (*Session) GetID () string`
- `func (*Session) GetLastSeenAt () *time.Time`
- `func (*Session) GetRefreshToken () string`
- `func (*Session) GetRemoteAddr () string`
- `func (*Session) GetRevokedAt () *time.Time`
- `func (*Session) GetUpdatedAt () *time.Time`
- `func (*Session) GetUserAgent () string`
- `func (*Session) GetUserID () string`
- `func (*Session) Insert (opts ...QueryOption) string`
- `func (*Session) Select (opts ...QueryOption) string`
- `func (*Session) SetCreatedAt (stamp time.Time)`
- `func (*Session) SetExpiresAt (stamp time.Time)`
- `func (*Session) SetHostname (val string)`
- `func (*Session) SetID (val string)`
- `func (*Session) SetLastSeenAt (stamp time.Time)`
- `func (*Session) SetRefreshToken (val string)`
- `func (*Session) SetRemoteAddr (val string)`
- `func (*Session) SetRevokedAt (stamp time.Time)`
- `func (*Session) SetUpdatedAt (stamp time.Time)`
- `func (*Session) SetUserAgent (val string)`
- `func (*Session) SetUserID (val string)`
- `func (*Session) Update (opts ...QueryOption) string`
- `func (*Setting) Delete (opts ...QueryOption) string`
- `func (*Setting) GetCreatedAt () *time.Time`
- `func (*Setting) GetName () string`
- `func (*Setting) GetUpdatedAt () *time.Time`
- `func (*Setting) GetUpdatedByUserID () string`
- `func (*Setting) GetValue () string`
- `func (*Setting) Insert (opts ...QueryOption) string`
- `func (*Setting) Select (opts ...QueryOption) string`
- `func (*Setting) SetCreatedAt (stamp time.Time)`
- `func (*Setting) SetName (val string)`
- `func (*Setting) SetUpdatedAt (stamp time.Time)`
- `func (*Setting) SetUpdatedByUserID (val string)`
- `func (*Setting) SetValue (val string)`
- `func (*Setting) Update (opts ...QueryOption) string`
- `func (*User) Delete (opts ...QueryOption) string`
- `func (*User) GetCreatedAt () *time.Time`
- `func (*User) GetDeletedAt () *time.Time`
- `func (*User) GetEmail () string`
- `func (*User) GetFullName () string`
- `func (*User) GetID () string`
- `func (*User) GetIsActive () bool`
- `func (*User) GetIsAdmin () bool`
- `func (*User) GetIsAgent () bool`
- `func (*User) GetPassword () string`
- `func (*User) GetUpdatedAt () *time.Time`
- `func (*User) GetUsername () string`
- `func (*User) Insert (opts ...QueryOption) string`
- `func (*User) Select (opts ...QueryOption) string`
- `func (*User) SetCreatedAt (stamp time.Time)`
- `func (*User) SetDeletedAt (stamp time.Time)`
- `func (*User) SetEmail (val string)`
- `func (*User) SetFullName (val string)`
- `func (*User) SetID (val string)`
- `func (*User) SetIsActive (val bool)`
- `func (*User) SetIsAdmin (val bool)`
- `func (*User) SetIsAgent (val bool)`
- `func (*User) SetPassword (val string)`
- `func (*User) SetUpdatedAt (stamp time.Time)`
- `func (*User) SetUsername (val string)`
- `func (*User) Update (opts ...QueryOption) string`
- `func (SettingDefinition) ValidateSetting (value string) error`

### LookupSetting

LookupSetting returns the definition for a name.

```go
func LookupSetting(name string) (SettingDefinition, bool)
```

### MatchRepository

MatchRepository reports whether a repository slug matches a pattern.

Patterns are written against the normalized slug (`host/owner/name`):

```
github.com/titpetric/atkins   one repository
github.com/titpetric/*        every repository of one owner
github.com/**                 every repository on one host
**                            everything
```

`*` matches within a path segment; `**` crosses segments. Matching is
case-insensitive because slugs are lowercased.

```go
func MatchRepository(pattern, slug string) bool
```

### RepositorySlug

RepositorySlug normalizes a git remote URL into a stable identity of
the form `host/owner/name`. The slug is what the server deduplicates
on, so `git@github.com:titpetric/atkins.git` and
`https://github.com/titpetric/atkins` are one repository.

An unrecognized remote is returned trimmed and lowercased rather than
rejected; a private or self-hosted remote should still be usable
without the server knowing its URL shape.

```go
func RepositorySlug(remoteURL string) string
```

### SettingDefinitions

SettingDefinitions returns the registry.

```go
func SettingDefinitions() []SettingDefinition
```

### TerminalJobStatus

TerminalJobStatus reports whether status is a settled state.

```go
func TerminalJobStatus(status JobStatus) bool
```

### ValidJobStatus

ValidJobStatus reports whether status is a known job status.

```go
func ValidJobStatus(status JobStatus) bool
```

### ValidRepositoryPolicy

ValidRepositoryPolicy reports whether policy is a known value.

```go
func ValidRepositoryPolicy(policy RepositoryPolicy) bool
```

### WithColumns

WithColumns will set the columns to use for the query.

```go
func WithColumns(cols []string) QueryOption
```

### WithLimit

WithLimit will set the limit clause parameters for the query.

```go
func WithLimit(start, offset int) QueryOption
```

### WithOrderBy

WithOrderBy will set the order by clause for the query.

```go
func WithOrderBy(clause string) QueryOption
```

### WithStatement

WithStatement will change the statement for the query.

```go
func WithStatement(stmt string) QueryOption
```

### WithTable

WithTable will set the table name for the query.

```go
func WithTable(name string) QueryOption
```

### WithWhere

WithWhere will set the where condition for the query.

```go
func WithWhere(clause string) QueryOption
```

### Delete

Delete starts building a DELETE query.

```go
func (*Job) Delete(opts ...QueryOption) string
```

### GetAgentID

GetAgentID will return the value of AgentID.

```go
func (*Job) GetAgentID() string
```

### GetBranch

GetBranch will return the value of Branch.

```go
func (*Job) GetBranch() string
```

### GetCommand

GetCommand will return the value of Command.

```go
func (*Job) GetCommand() string
```

### GetCreatedAt

GetCreatedAt will return the value of CreatedAt.

```go
func (*Job) GetCreatedAt() *time.Time
```

### GetDepth

GetDepth will return the value of Depth.

```go
func (*Job) GetDepth() int64
```

### GetError

GetError will return the value of Error.

```go
func (*Job) GetError() string
```

### GetExitCode

GetExitCode will return the value of ExitCode.

```go
func (*Job) GetExitCode() int64
```

### GetFinishedAt

GetFinishedAt will return the value of FinishedAt.

```go
func (*Job) GetFinishedAt() *time.Time
```

### GetID

GetID will return the value of ID.

```go
func (*Job) GetID() string
```

### GetLabels

GetLabels will return the value of Labels.

```go
func (*Job) GetLabels() string
```

### GetLeaseExpiresAt

GetLeaseExpiresAt will return the value of LeaseExpiresAt.

```go
func (*Job) GetLeaseExpiresAt() *time.Time
```

### GetParams

GetParams will return the value of Params.

```go
func (*Job) GetParams() string
```

### GetParentID

GetParentID will return the value of ParentID.

```go
func (*Job) GetParentID() string
```

### GetRepositoryID

GetRepositoryID will return the value of RepositoryID.

```go
func (*Job) GetRepositoryID() string
```

### GetRevision

GetRevision will return the value of Revision.

```go
func (*Job) GetRevision() string
```

### GetRootID

GetRootID will return the value of RootID.

```go
func (*Job) GetRootID() string
```

### GetStartedAt

GetStartedAt will return the value of StartedAt.

```go
func (*Job) GetStartedAt() *time.Time
```

### GetStatus

GetStatus will return the value of Status.

```go
func (*Job) GetStatus() string
```

### GetUpdatedAt

GetUpdatedAt will return the value of UpdatedAt.

```go
func (*Job) GetUpdatedAt() *time.Time
```

### GetUserID

GetUserID will return the value of UserID.

```go
func (*Job) GetUserID() string
```

### GetWorkingDirectory

GetWorkingDirectory will return the value of WorkingDirectory.

```go
func (*Job) GetWorkingDirectory() string
```

### Insert

Insert starts building an INSERT INTO query.

```go
func (*Job) Insert(opts ...QueryOption) string
```

### IsTerminal

IsTerminal reports whether the job has settled.

```go
func (*Job) IsTerminal() bool
```

### Select

Select starts building a SELECT query.

```go
func (*Job) Select(opts ...QueryOption) string
```

### SetAgentID

SetAgentID sets AgentID to the provided value.

```go
func (*Job) SetAgentID(val string)
```

### SetBranch

SetBranch sets Branch to the provided value.

```go
func (*Job) SetBranch(val string)
```

### SetCommand

SetCommand sets Command to the provided value.

```go
func (*Job) SetCommand(val string)
```

### SetCreatedAt

SetCreatedAt sets CreatedAt to the provided value.

```go
func (*Job) SetCreatedAt(stamp time.Time)
```

### SetDepth

SetDepth sets Depth to the provided value.

```go
func (*Job) SetDepth(val int64)
```

### SetError

SetError sets Error to the provided value.

```go
func (*Job) SetError(val string)
```

### SetExitCode

SetExitCode sets ExitCode to the provided value.

```go
func (*Job) SetExitCode(val int64)
```

### SetFinishedAt

SetFinishedAt sets FinishedAt to the provided value.

```go
func (*Job) SetFinishedAt(stamp time.Time)
```

### SetID

SetID sets ID to the provided value.

```go
func (*Job) SetID(val string)
```

### SetLabels

SetLabels sets Labels to the provided value.

```go
func (*Job) SetLabels(val string)
```

### SetLeaseExpiresAt

SetLeaseExpiresAt sets LeaseExpiresAt to the provided value.

```go
func (*Job) SetLeaseExpiresAt(stamp time.Time)
```

### SetParams

SetParams sets Params to the provided value.

```go
func (*Job) SetParams(val string)
```

### SetParentID

SetParentID sets ParentID to the provided value.

```go
func (*Job) SetParentID(val string)
```

### SetRepositoryID

SetRepositoryID sets RepositoryID to the provided value.

```go
func (*Job) SetRepositoryID(val string)
```

### SetRevision

SetRevision sets Revision to the provided value.

```go
func (*Job) SetRevision(val string)
```

### SetRootID

SetRootID sets RootID to the provided value.

```go
func (*Job) SetRootID(val string)
```

### SetStartedAt

SetStartedAt sets StartedAt to the provided value.

```go
func (*Job) SetStartedAt(stamp time.Time)
```

### SetStatus

SetStatus sets Status to the provided value.

```go
func (*Job) SetStatus(val string)
```

### SetUpdatedAt

SetUpdatedAt sets UpdatedAt to the provided value.

```go
func (*Job) SetUpdatedAt(stamp time.Time)
```

### SetUserID

SetUserID sets UserID to the provided value.

```go
func (*Job) SetUserID(val string)
```

### SetWorkingDirectory

SetWorkingDirectory sets WorkingDirectory to the provided value.

```go
func (*Job) SetWorkingDirectory(val string)
```

### Update

Update starts building a UPDATE query.

```go
func (*Job) Update(opts ...QueryOption) string
```

### Delete

Delete starts building a DELETE query.

```go
func (*JobLog) Delete(opts ...QueryOption) string
```

### GetContent

GetContent will return the value of Content.

```go
func (*JobLog) GetContent() string
```

### GetCreatedAt

GetCreatedAt will return the value of CreatedAt.

```go
func (*JobLog) GetCreatedAt() *time.Time
```

### GetID

GetID will return the value of ID.

```go
func (*JobLog) GetID() string
```

### GetJobID

GetJobID will return the value of JobID.

```go
func (*JobLog) GetJobID() string
```

### GetSeq

GetSeq will return the value of Seq.

```go
func (*JobLog) GetSeq() int64
```

### GetStream

GetStream will return the value of Stream.

```go
func (*JobLog) GetStream() string
```

### Insert

Insert starts building an INSERT INTO query.

```go
func (*JobLog) Insert(opts ...QueryOption) string
```

### Select

Select starts building a SELECT query.

```go
func (*JobLog) Select(opts ...QueryOption) string
```

### SetContent

SetContent sets Content to the provided value.

```go
func (*JobLog) SetContent(val string)
```

### SetCreatedAt

SetCreatedAt sets CreatedAt to the provided value.

```go
func (*JobLog) SetCreatedAt(stamp time.Time)
```

### SetID

SetID sets ID to the provided value.

```go
func (*JobLog) SetID(val string)
```

### SetJobID

SetJobID sets JobID to the provided value.

```go
func (*JobLog) SetJobID(val string)
```

### SetSeq

SetSeq sets Seq to the provided value.

```go
func (*JobLog) SetSeq(val int64)
```

### SetStream

SetStream sets Stream to the provided value.

```go
func (*JobLog) SetStream(val string)
```

### Update

Update starts building a UPDATE query.

```go
func (*JobLog) Update(opts ...QueryOption) string
```

### Delete

Delete starts building a DELETE query.

```go
func (*Migrations) Delete(opts ...QueryOption) string
```

### GetFilename

GetFilename will return the value of Filename.

```go
func (*Migrations) GetFilename() string
```

### GetProject

GetProject will return the value of Project.

```go
func (*Migrations) GetProject() string
```

### GetStatementIndex

GetStatementIndex will return the value of StatementIndex.

```go
func (*Migrations) GetStatementIndex() int64
```

### GetStatus

GetStatus will return the value of Status.

```go
func (*Migrations) GetStatus() string
```

### Insert

Insert starts building an INSERT INTO query.

```go
func (*Migrations) Insert(opts ...QueryOption) string
```

### Select

Select starts building a SELECT query.

```go
func (*Migrations) Select(opts ...QueryOption) string
```

### SetFilename

SetFilename sets Filename to the provided value.

```go
func (*Migrations) SetFilename(val string)
```

### SetProject

SetProject sets Project to the provided value.

```go
func (*Migrations) SetProject(val string)
```

### SetStatementIndex

SetStatementIndex sets StatementIndex to the provided value.

```go
func (*Migrations) SetStatementIndex(val int64)
```

### SetStatus

SetStatus sets Status to the provided value.

```go
func (*Migrations) SetStatus(val string)
```

### Update

Update starts building a UPDATE query.

```go
func (*Migrations) Update(opts ...QueryOption) string
```

### Apply

Apply will use passed query options to populate the query.

```go
func (*QueryConfig) Apply(opts ...QueryOption) *QueryConfig
```

### WithColumns

WithColumns will set the columns to use for the query.

```go
func (*QueryConfig) WithColumns(cols []string) QueryOption
```

### WithLimit

WithLimit will set the limit clause parameters for the query.

```go
func (*QueryConfig) WithLimit(start, offset int) QueryOption
```

### WithOrderBy

WithOrderBy will set the order by clause for the query.

```go
func (*QueryConfig) WithOrderBy(clause string) QueryOption
```

### WithStatement

WithStatement will change the statement for the query.

```go
func (*QueryConfig) WithStatement(stmt string) QueryOption
```

### WithTable

WithTable will set the table name for the query.

```go
func (*QueryConfig) WithTable(name string) QueryOption
```

### WithWhere

WithWhere will set the where condition for the query.

```go
func (*QueryConfig) WithWhere(clause string) QueryOption
```

### Delete

Delete starts building a DELETE query.

```go
func (*Repository) Delete(opts ...QueryOption) string
```

### GetCreatedAt

GetCreatedAt will return the value of CreatedAt.

```go
func (*Repository) GetCreatedAt() *time.Time
```

### GetCreatedByUserID

GetCreatedByUserID will return the value of CreatedByUserID.

```go
func (*Repository) GetCreatedByUserID() string
```

### GetDefaultBranch

GetDefaultBranch will return the value of DefaultBranch.

```go
func (*Repository) GetDefaultBranch() string
```

### GetDeletedAt

GetDeletedAt will return the value of DeletedAt.

```go
func (*Repository) GetDeletedAt() *time.Time
```

### GetID

GetID will return the value of ID.

```go
func (*Repository) GetID() string
```

### GetIsActive

GetIsActive will return the value of IsActive.

```go
func (*Repository) GetIsActive() bool
```

### GetRemoteURL

GetRemoteURL will return the value of RemoteURL.

```go
func (*Repository) GetRemoteURL() string
```

### GetSlug

GetSlug will return the value of Slug.

```go
func (*Repository) GetSlug() string
```

### GetUpdatedAt

GetUpdatedAt will return the value of UpdatedAt.

```go
func (*Repository) GetUpdatedAt() *time.Time
```

### Insert

Insert starts building an INSERT INTO query.

```go
func (*Repository) Insert(opts ...QueryOption) string
```

### Select

Select starts building a SELECT query.

```go
func (*Repository) Select(opts ...QueryOption) string
```

### SetCreatedAt

SetCreatedAt sets CreatedAt to the provided value.

```go
func (*Repository) SetCreatedAt(stamp time.Time)
```

### SetCreatedByUserID

SetCreatedByUserID sets CreatedByUserID to the provided value.

```go
func (*Repository) SetCreatedByUserID(val string)
```

### SetDefaultBranch

SetDefaultBranch sets DefaultBranch to the provided value.

```go
func (*Repository) SetDefaultBranch(val string)
```

### SetDeletedAt

SetDeletedAt sets DeletedAt to the provided value.

```go
func (*Repository) SetDeletedAt(stamp time.Time)
```

### SetID

SetID sets ID to the provided value.

```go
func (*Repository) SetID(val string)
```

### SetIsActive

SetIsActive sets IsActive to the provided value.

```go
func (*Repository) SetIsActive(val bool)
```

### SetRemoteURL

SetRemoteURL sets RemoteURL to the provided value.

```go
func (*Repository) SetRemoteURL(val string)
```

### SetSlug

SetSlug sets Slug to the provided value.

```go
func (*Repository) SetSlug(val string)
```

### SetUpdatedAt

SetUpdatedAt sets UpdatedAt to the provided value.

```go
func (*Repository) SetUpdatedAt(stamp time.Time)
```

### Update

Update starts building a UPDATE query.

```go
func (*Repository) Update(opts ...QueryOption) string
```

### Delete

Delete starts building a DELETE query.

```go
func (*RepositoryRule) Delete(opts ...QueryOption) string
```

### GetCreatedAt

GetCreatedAt will return the value of CreatedAt.

```go
func (*RepositoryRule) GetCreatedAt() *time.Time
```

### GetCreatedByUserID

GetCreatedByUserID will return the value of CreatedByUserID.

```go
func (*RepositoryRule) GetCreatedByUserID() string
```

### GetDeletedAt

GetDeletedAt will return the value of DeletedAt.

```go
func (*RepositoryRule) GetDeletedAt() *time.Time
```

### GetDescription

GetDescription will return the value of Description.

```go
func (*RepositoryRule) GetDescription() string
```

### GetID

GetID will return the value of ID.

```go
func (*RepositoryRule) GetID() string
```

### GetIsActive

GetIsActive will return the value of IsActive.

```go
func (*RepositoryRule) GetIsActive() bool
```

### GetPattern

GetPattern will return the value of Pattern.

```go
func (*RepositoryRule) GetPattern() string
```

### GetUpdatedAt

GetUpdatedAt will return the value of UpdatedAt.

```go
func (*RepositoryRule) GetUpdatedAt() *time.Time
```

### Insert

Insert starts building an INSERT INTO query.

```go
func (*RepositoryRule) Insert(opts ...QueryOption) string
```

### Select

Select starts building a SELECT query.

```go
func (*RepositoryRule) Select(opts ...QueryOption) string
```

### SetCreatedAt

SetCreatedAt sets CreatedAt to the provided value.

```go
func (*RepositoryRule) SetCreatedAt(stamp time.Time)
```

### SetCreatedByUserID

SetCreatedByUserID sets CreatedByUserID to the provided value.

```go
func (*RepositoryRule) SetCreatedByUserID(val string)
```

### SetDeletedAt

SetDeletedAt sets DeletedAt to the provided value.

```go
func (*RepositoryRule) SetDeletedAt(stamp time.Time)
```

### SetDescription

SetDescription sets Description to the provided value.

```go
func (*RepositoryRule) SetDescription(val string)
```

### SetID

SetID sets ID to the provided value.

```go
func (*RepositoryRule) SetID(val string)
```

### SetIsActive

SetIsActive sets IsActive to the provided value.

```go
func (*RepositoryRule) SetIsActive(val bool)
```

### SetPattern

SetPattern sets Pattern to the provided value.

```go
func (*RepositoryRule) SetPattern(val string)
```

### SetUpdatedAt

SetUpdatedAt sets UpdatedAt to the provided value.

```go
func (*RepositoryRule) SetUpdatedAt(stamp time.Time)
```

### Update

Update starts building a UPDATE query.

```go
func (*RepositoryRule) Update(opts ...QueryOption) string
```

### Delete

Delete starts building a DELETE query.

```go
func (*SSHKey) Delete(opts ...QueryOption) string
```

### GetCreatedAt

GetCreatedAt will return the value of CreatedAt.

```go
func (*SSHKey) GetCreatedAt() *time.Time
```

### GetCreatedByUserID

GetCreatedByUserID will return the value of CreatedByUserID.

```go
func (*SSHKey) GetCreatedByUserID() string
```

### GetDeletedAt

GetDeletedAt will return the value of DeletedAt.

```go
func (*SSHKey) GetDeletedAt() *time.Time
```

### GetFingerprint

GetFingerprint will return the value of Fingerprint.

```go
func (*SSHKey) GetFingerprint() string
```

### GetHost

GetHost will return the value of Host.

```go
func (*SSHKey) GetHost() string
```

### GetID

GetID will return the value of ID.

```go
func (*SSHKey) GetID() string
```

### GetIsActive

GetIsActive will return the value of IsActive.

```go
func (*SSHKey) GetIsActive() bool
```

### GetKnownHosts

GetKnownHosts will return the value of KnownHosts.

```go
func (*SSHKey) GetKnownHosts() string
```

### GetLastUsedAt

GetLastUsedAt will return the value of LastUsedAt.

```go
func (*SSHKey) GetLastUsedAt() *time.Time
```

### GetName

GetName will return the value of Name.

```go
func (*SSHKey) GetName() string
```

### GetPrivateKey

GetPrivateKey will return the value of PrivateKey.

```go
func (*SSHKey) GetPrivateKey() string
```

### GetPublicKey

GetPublicKey will return the value of PublicKey.

```go
func (*SSHKey) GetPublicKey() string
```

### GetUpdatedAt

GetUpdatedAt will return the value of UpdatedAt.

```go
func (*SSHKey) GetUpdatedAt() *time.Time
```

### Insert

Insert starts building an INSERT INTO query.

```go
func (*SSHKey) Insert(opts ...QueryOption) string
```

### Select

Select starts building a SELECT query.

```go
func (*SSHKey) Select(opts ...QueryOption) string
```

### SetCreatedAt

SetCreatedAt sets CreatedAt to the provided value.

```go
func (*SSHKey) SetCreatedAt(stamp time.Time)
```

### SetCreatedByUserID

SetCreatedByUserID sets CreatedByUserID to the provided value.

```go
func (*SSHKey) SetCreatedByUserID(val string)
```

### SetDeletedAt

SetDeletedAt sets DeletedAt to the provided value.

```go
func (*SSHKey) SetDeletedAt(stamp time.Time)
```

### SetFingerprint

SetFingerprint sets Fingerprint to the provided value.

```go
func (*SSHKey) SetFingerprint(val string)
```

### SetHost

SetHost sets Host to the provided value.

```go
func (*SSHKey) SetHost(val string)
```

### SetID

SetID sets ID to the provided value.

```go
func (*SSHKey) SetID(val string)
```

### SetIsActive

SetIsActive sets IsActive to the provided value.

```go
func (*SSHKey) SetIsActive(val bool)
```

### SetKnownHosts

SetKnownHosts sets KnownHosts to the provided value.

```go
func (*SSHKey) SetKnownHosts(val string)
```

### SetLastUsedAt

SetLastUsedAt sets LastUsedAt to the provided value.

```go
func (*SSHKey) SetLastUsedAt(stamp time.Time)
```

### SetName

SetName sets Name to the provided value.

```go
func (*SSHKey) SetName(val string)
```

### SetPrivateKey

SetPrivateKey sets PrivateKey to the provided value.

```go
func (*SSHKey) SetPrivateKey(val string)
```

### SetPublicKey

SetPublicKey sets PublicKey to the provided value.

```go
func (*SSHKey) SetPublicKey(val string)
```

### SetUpdatedAt

SetUpdatedAt sets UpdatedAt to the provided value.

```go
func (*SSHKey) SetUpdatedAt(stamp time.Time)
```

### Update

Update starts building a UPDATE query.

```go
func (*SSHKey) Update(opts ...QueryOption) string
```

### Delete

Delete starts building a DELETE query.

```go
func (*Session) Delete(opts ...QueryOption) string
```

### GetCreatedAt

GetCreatedAt will return the value of CreatedAt.

```go
func (*Session) GetCreatedAt() *time.Time
```

### GetExpiresAt

GetExpiresAt will return the value of ExpiresAt.

```go
func (*Session) GetExpiresAt() *time.Time
```

### GetHostname

GetHostname will return the value of Hostname.

```go
func (*Session) GetHostname() string
```

### GetID

GetID will return the value of ID.

```go
func (*Session) GetID() string
```

### GetLastSeenAt

GetLastSeenAt will return the value of LastSeenAt.

```go
func (*Session) GetLastSeenAt() *time.Time
```

### GetRefreshToken

GetRefreshToken will return the value of RefreshToken.

```go
func (*Session) GetRefreshToken() string
```

### GetRemoteAddr

GetRemoteAddr will return the value of RemoteAddr.

```go
func (*Session) GetRemoteAddr() string
```

### GetRevokedAt

GetRevokedAt will return the value of RevokedAt.

```go
func (*Session) GetRevokedAt() *time.Time
```

### GetUpdatedAt

GetUpdatedAt will return the value of UpdatedAt.

```go
func (*Session) GetUpdatedAt() *time.Time
```

### GetUserAgent

GetUserAgent will return the value of UserAgent.

```go
func (*Session) GetUserAgent() string
```

### GetUserID

GetUserID will return the value of UserID.

```go
func (*Session) GetUserID() string
```

### Insert

Insert starts building an INSERT INTO query.

```go
func (*Session) Insert(opts ...QueryOption) string
```

### Select

Select starts building a SELECT query.

```go
func (*Session) Select(opts ...QueryOption) string
```

### SetCreatedAt

SetCreatedAt sets CreatedAt to the provided value.

```go
func (*Session) SetCreatedAt(stamp time.Time)
```

### SetExpiresAt

SetExpiresAt sets ExpiresAt to the provided value.

```go
func (*Session) SetExpiresAt(stamp time.Time)
```

### SetHostname

SetHostname sets Hostname to the provided value.

```go
func (*Session) SetHostname(val string)
```

### SetID

SetID sets ID to the provided value.

```go
func (*Session) SetID(val string)
```

### SetLastSeenAt

SetLastSeenAt sets LastSeenAt to the provided value.

```go
func (*Session) SetLastSeenAt(stamp time.Time)
```

### SetRefreshToken

SetRefreshToken sets RefreshToken to the provided value.

```go
func (*Session) SetRefreshToken(val string)
```

### SetRemoteAddr

SetRemoteAddr sets RemoteAddr to the provided value.

```go
func (*Session) SetRemoteAddr(val string)
```

### SetRevokedAt

SetRevokedAt sets RevokedAt to the provided value.

```go
func (*Session) SetRevokedAt(stamp time.Time)
```

### SetUpdatedAt

SetUpdatedAt sets UpdatedAt to the provided value.

```go
func (*Session) SetUpdatedAt(stamp time.Time)
```

### SetUserAgent

SetUserAgent sets UserAgent to the provided value.

```go
func (*Session) SetUserAgent(val string)
```

### SetUserID

SetUserID sets UserID to the provided value.

```go
func (*Session) SetUserID(val string)
```

### Update

Update starts building a UPDATE query.

```go
func (*Session) Update(opts ...QueryOption) string
```

### Delete

Delete starts building a DELETE query.

```go
func (*Setting) Delete(opts ...QueryOption) string
```

### GetCreatedAt

GetCreatedAt will return the value of CreatedAt.

```go
func (*Setting) GetCreatedAt() *time.Time
```

### GetName

GetName will return the value of Name.

```go
func (*Setting) GetName() string
```

### GetUpdatedAt

GetUpdatedAt will return the value of UpdatedAt.

```go
func (*Setting) GetUpdatedAt() *time.Time
```

### GetUpdatedByUserID

GetUpdatedByUserID will return the value of UpdatedByUserID.

```go
func (*Setting) GetUpdatedByUserID() string
```

### GetValue

GetValue will return the value of Value.

```go
func (*Setting) GetValue() string
```

### Insert

Insert starts building an INSERT INTO query.

```go
func (*Setting) Insert(opts ...QueryOption) string
```

### Select

Select starts building a SELECT query.

```go
func (*Setting) Select(opts ...QueryOption) string
```

### SetCreatedAt

SetCreatedAt sets CreatedAt to the provided value.

```go
func (*Setting) SetCreatedAt(stamp time.Time)
```

### SetName

SetName sets Name to the provided value.

```go
func (*Setting) SetName(val string)
```

### SetUpdatedAt

SetUpdatedAt sets UpdatedAt to the provided value.

```go
func (*Setting) SetUpdatedAt(stamp time.Time)
```

### SetUpdatedByUserID

SetUpdatedByUserID sets UpdatedByUserID to the provided value.

```go
func (*Setting) SetUpdatedByUserID(val string)
```

### SetValue

SetValue sets Value to the provided value.

```go
func (*Setting) SetValue(val string)
```

### Update

Update starts building a UPDATE query.

```go
func (*Setting) Update(opts ...QueryOption) string
```

### Delete

Delete starts building a DELETE query.

```go
func (*User) Delete(opts ...QueryOption) string
```

### GetCreatedAt

GetCreatedAt will return the value of CreatedAt.

```go
func (*User) GetCreatedAt() *time.Time
```

### GetDeletedAt

GetDeletedAt will return the value of DeletedAt.

```go
func (*User) GetDeletedAt() *time.Time
```

### GetEmail

GetEmail will return the value of Email.

```go
func (*User) GetEmail() string
```

### GetFullName

GetFullName will return the value of FullName.

```go
func (*User) GetFullName() string
```

### GetID

GetID will return the value of ID.

```go
func (*User) GetID() string
```

### GetIsActive

GetIsActive will return the value of IsActive.

```go
func (*User) GetIsActive() bool
```

### GetIsAdmin

GetIsAdmin will return the value of IsAdmin.

```go
func (*User) GetIsAdmin() bool
```

### GetIsAgent

GetIsAgent will return the value of IsAgent.

```go
func (*User) GetIsAgent() bool
```

### GetPassword

GetPassword will return the value of Password.

```go
func (*User) GetPassword() string
```

### GetUpdatedAt

GetUpdatedAt will return the value of UpdatedAt.

```go
func (*User) GetUpdatedAt() *time.Time
```

### GetUsername

GetUsername will return the value of Username.

```go
func (*User) GetUsername() string
```

### Insert

Insert starts building an INSERT INTO query.

```go
func (*User) Insert(opts ...QueryOption) string
```

### Select

Select starts building a SELECT query.

```go
func (*User) Select(opts ...QueryOption) string
```

### SetCreatedAt

SetCreatedAt sets CreatedAt to the provided value.

```go
func (*User) SetCreatedAt(stamp time.Time)
```

### SetDeletedAt

SetDeletedAt sets DeletedAt to the provided value.

```go
func (*User) SetDeletedAt(stamp time.Time)
```

### SetEmail

SetEmail sets Email to the provided value.

```go
func (*User) SetEmail(val string)
```

### SetFullName

SetFullName sets FullName to the provided value.

```go
func (*User) SetFullName(val string)
```

### SetID

SetID sets ID to the provided value.

```go
func (*User) SetID(val string)
```

### SetIsActive

SetIsActive sets IsActive to the provided value.

```go
func (*User) SetIsActive(val bool)
```

### SetIsAdmin

SetIsAdmin sets IsAdmin to the provided value.

```go
func (*User) SetIsAdmin(val bool)
```

### SetIsAgent

SetIsAgent sets IsAgent to the provided value.

```go
func (*User) SetIsAgent(val bool)
```

### SetPassword

SetPassword sets Password to the provided value.

```go
func (*User) SetPassword(val string)
```

### SetUpdatedAt

SetUpdatedAt sets UpdatedAt to the provided value.

```go
func (*User) SetUpdatedAt(stamp time.Time)
```

### SetUsername

SetUsername sets Username to the provided value.

```go
func (*User) SetUsername(val string)
```

### Update

Update starts building a UPDATE query.

```go
func (*User) Update(opts ...QueryOption) string
```

### ValidateSetting

ValidateSetting checks a value against its definition.

```go
func (SettingDefinition) ValidateSetting(value string) error
```

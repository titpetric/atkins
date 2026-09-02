# Package ./server/api

```go
import (
	"github.com/titpetric/atkins/server/api"
}
```

Package api implements the JSON endpoints of the atkins CI/CD server.

The endpoints fall into two groups. `/api/user/*` is the login flow the
`atkins --login https://domain` client drives. `/api/dispatch` and
`/api/job/*` are the queue: the CLI records a run, agents claim work
and report back.

Handlers decode, validate, map errors to status codes and encode. They
call storage rather than issuing SQL.

## Types

<details>
<summary><code>type AgentSSHKey</code></summary>

```go
// AgentSSHKey is a deploy key handed to an agent, private half and all.
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
<summary><code>type ArtefactView</code></summary>

```go
// ArtefactView is how an artefact is described over the API.
// It is a projection rather than the row: `storage_key` says where the
// bytes sit on the server, and that is the server's business. The rule
// is the same one SSHKeyView follows — the projection is the guard, not
// a `json:"-"` somebody has to remember to add to a new column.
type ArtefactView struct {
	ID          string `json:"id"`
	JobID       string `json:"job_id"`
	Path        string `json:"path"`
	Size        int64  `json:"size"`
	ContentType string `json:"content_type"`
	Checksum    string `json:"checksum"`
	AgentID     string `json:"agent_id,omitempty"`
	CreatedAt   string `json:"created_at,omitempty"`

	// URL is where the bytes are, so a caller that listed artefacts
	// does not have to know how to build the path.
	URL string `json:"url"`
}
```

</details>

<details>
<summary><code>type ClaimRequest</code></summary>

```go
// ClaimRequest is the body of /api/job/claim.
type ClaimRequest struct {
	// AgentID identifies the worker taking the lease.
	AgentID string `json:"agent_id"`

	// Labels are what the agent can offer. A job requiring labels only
	// lands on an agent advertising all of them.
	Labels []string `json:"labels"`
}
```

</details>

<details>
<summary><code>type ClaimResponse</code></summary>

```go
// ClaimResponse is returned when an agent leases a job.
// The repository is included rather than referenced: the agent's very
// next act is to clone it, and making it ask again for the remote URL
// would be a round trip that buys nothing.
type ClaimResponse struct {
	Job        *model.Job        `json:"job"`
	Repository *model.Repository `json:"repository"`
}
```

</details>

<details>
<summary><code>type DispatchRequest</code></summary>

```go
// DispatchRequest is the body of /api/dispatch.
// It carries what the server needs to reconstruct a run somewhere else:
// which repository, where inside its work tree, and which atkins command.
type DispatchRequest struct {
	// Repository identifies the git repository the client is working
	// on. The remote URL is normalized into a slug server-side, so ssh
	// and https clones of the same repository are one repository.
	Repository RepositoryPayload `json:"repository"`

	// WorkingDirectory is relative to the repository root. Empty means
	// the root itself.
	WorkingDirectory string `json:"working_directory"`

	// Command is the atkins invocation, e.g. "atkins test:build".
	Command string `json:"command"`

	// ParentID is the dispatching job when a running pipeline queues
	// further work. The client passes through ATKINS_JOB_ID.
	ParentID string `json:"parent_id"`

	// Labels constrain which agents may run the job. An empty list
	// runs anywhere.
	Labels []string `json:"labels"`

	// Params is an arbitrary JSON object handed to the job.
	Params map[string]any `json:"params"`

	// CloneDepth limits the history of the work tree the agent builds.
	// 0, the default, is the whole history.
	CloneDepth int64 `json:"clone_depth"`

	// Artefacts are glob patterns the agent collects after the command
	// exits, relative to the directory it ran in.
	Artefacts []string `json:"artefacts"`

	// Agent names the machine already running this command. Set, the
	// job is recorded as running there instead of queued for an agent
	// to claim, and the caller reports its log and its outcome.
	//
	// It is how a run on somebody's laptop is on the server at all: the
	// pipeline executes where it was started, and the history does not
	// depend on whether that machine could hand the work over.
	Agent string `json:"agent"`
}
```

</details>

<details>
<summary><code>type DispatchResponse</code></summary>

```go
// DispatchResponse is what /api/dispatch returns.
// The CLI puts JobID into ATKINS_JOB_ID for the run it is about to
// perform, and reports the outcome back to /api/job/{id}/status. The
// server records the job; atkins decides what to do with it.
type DispatchResponse struct {
	JobID          string `json:"job_id"`
	ParentID       string `json:"parent_id,omitempty"`
	RootID         string `json:"root_id"`
	Depth          int64  `json:"depth"`
	RepositoryID   string `json:"repository_id"`
	RepositorySlug string `json:"repository_slug"`
	Status         string `json:"status"`

	// ViewToken opens the job page in a browser without a session. It
	// is present only while the instance keeps jobs private; a public
	// one needs no token, and printing one there would put a secret in
	// a URL that guards nothing.
	ViewToken string `json:"view_token,omitempty"`
}
```

</details>

<details>
<summary><code>type EnrolRequest</code></summary>

```go
// EnrolRequest is the body of POST /api/agent/enrol.
type EnrolRequest struct {
	// Token is the shared enrolment secret configured on the server.
	Token string `json:"token"`

	// AgentID names the agent. Enrolling twice with the same ID
	// returns the same account, so a restarted agent keeps its
	// identity and its job history.
	AgentID string `json:"agent_id"`

	// Labels are advertised for information; the claim call is what
	// actually filters the queue.
	Labels []string `json:"labels"`
}
```

</details>

<details>
<summary><code>type Handlers</code></summary>

```go
// Handlers serves the atkins server API.
type Handlers struct {
	jwt      *auth.JWT
	tokenTTL time.Duration

	allowRegistration bool

	// agentToken enrols agents. Empty disables enrolment entirely.
	agentToken string

	users        *storage.UserStorage
	sessions     *storage.SessionStorage
	repositories *storage.RepositoryStorage
	jobs         *storage.JobStorage
	jobLogs      *storage.JobLogStorage
	artefacts    *storage.JobArtefactStorage
	rules        *storage.RepositoryRuleStorage
	settings     *storage.SettingStorage
	sshKeys      *storage.SSHKeyStorage
}
```

</details>

<details>
<summary><code>type JobCheckoutRequest</code></summary>

```go
// JobCheckoutRequest is the body of /api/job/{jobID}/checkout.
// It is the agent answering the question the job asked. A job may name a
// tag, and a tag moves; without the commit that tag pointed at, a run
// cannot be reproduced once it has.
type JobCheckoutRequest struct {
	// Ref is the effective ref, which is the one the job named unless
	// it named none and the agent resolved the default branch.
	Ref string `json:"ref"`

	// CommitSHA is the commit the work tree was placed at.
	CommitSHA string `json:"commit_sha"`
}
```

</details>

<details>
<summary><code>type JobLogRequest</code></summary>

```go
// JobLogRequest is the body of POST /api/job/{jobID}/log.
type JobLogRequest struct {
	// Stream is "output" (the command's combined output) or "error"
	// (the agent's own commentary). Anything else is read as output.
	Stream string `json:"stream"`

	Content string `json:"content"`
}
```

</details>

<details>
<summary><code>type JobStatusRequest</code></summary>

```go
// JobStatusRequest is the body of /api/job/{jobID}/status.
type JobStatusRequest struct {
	Status   string `json:"status"`
	ExitCode int64  `json:"exit_code"`
	Error    string `json:"error"`
}
```

</details>

<details>
<summary><code>type JobView</code></summary>

```go
// JobView is a job as the API returns it: the stored row, plus how to
// open it in a browser.
//
// The token is derived from the job id and the signing key rather than
// stored, so returning it to a caller who has just been allowed to read
// the job withholds nothing: they can already read the output and the
// artefacts the page shows. Leaving it out only meant that whoever lost
// the URL atkins printed could no longer open a job they own.
type JobView struct {
	*model.Job

	// ViewToken opens the job page without a session. Empty on a public
	// instance, which needs none.
	ViewToken string `json:"view_token,omitempty"`

	// URL is the page for this job, token and all. It is relative to
	// this server, the way an artefact's URL is.
	URL string `json:"url"`
}
```

</details>

<details>
<summary><code>type LoginRequest</code></summary>

```go
// LoginRequest is the body of /api/user/login.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`

	// Hostname identifies the machine logging in, so a user can tell
	// their sessions apart. Optional.
	Hostname string `json:"hostname"`
}
```

</details>

<details>
<summary><code>type LogoutRequest</code></summary>

```go
// LogoutRequest is the body of /api/user/logout. The refresh token is
// optional: an authenticated request already names its session.
type LogoutRequest struct {
	RefreshToken string `json:"refresh_token"`
}
```

</details>

<details>
<summary><code>type Options</code></summary>

```go
// Options is passed from the server module scope. Every field has a
// usable zero value except the storages, which the module wires after
// it has a database handle.
type Options struct {
	// SigningKey signs access tokens.
	SigningKey string

	// TokenTTL is the lifetime of an issued access token. Zero selects
	// DefaultTokenTTL.
	TokenTTL time.Duration

	// AllowRegistration opens /api/user/register to anyone. When
	// false, registration still succeeds while the instance has no
	// users at all, so a fresh server can be bootstrapped without
	// shell access to the database.
	AllowRegistration bool

	// AgentToken is the shared secret agents present to enrol. Empty
	// means no agent can enrol, and the queue is reachable only by an
	// account an admin flagged by hand.
	AgentToken string

	UserStorage           *storage.UserStorage
	SessionStorage        *storage.SessionStorage
	RepositoryStorage     *storage.RepositoryStorage
	JobStorage            *storage.JobStorage
	JobLogStorage         *storage.JobLogStorage
	JobArtefactStorage    *storage.JobArtefactStorage
	RepositoryRuleStorage *storage.RepositoryRuleStorage
	SettingStorage        *storage.SettingStorage
	SSHKeyStorage         *storage.SSHKeyStorage
}
```

</details>

<details>
<summary><code>type PolicyResponse</code></summary>

```go
// PolicyResponse tells an agent what it is allowed to run.
// The agent enforces this itself before cloning anything. The server
// already refuses a disallowed dispatch, but a job can outlive the rule
// that admitted it: a queued job whose repository was removed from the
// allowlist must not run just because it was queued first.
type PolicyResponse struct {
	// Policy is "open" or "allowlist".
	Policy string `json:"policy"`

	// Patterns are the active allowlist rules. Empty under an open
	// policy, and meaningful only under "allowlist".
	Patterns []string `json:"patterns"`
}
```

</details>

<details>
<summary><code>type RefreshRequest</code></summary>

```go
// RefreshRequest is the body of /api/user/refreshToken.
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}
```

</details>

<details>
<summary><code>type RegisterRequest</code></summary>

```go
// RegisterRequest is the body of /api/user/register.
type RegisterRequest struct {
	Email    string `json:"email"`
	Username string `json:"username"`
	FullName string `json:"full_name"`
	Password string `json:"password"`
}
```

</details>

<details>
<summary><code>type RepositoryPayload</code></summary>

```go
// RepositoryPayload is the git detail a client reports about its checkout.
type RepositoryPayload struct {
	RemoteURL string `json:"remote_url"`

	// Ref is what to check out: a branch, a tag, a commit sha or a
	// fully qualified refname. The atkins client sends the commit it is
	// on, so a dispatched run builds the code in front of whoever
	// started it.
	Ref string `json:"ref"`

	// DefaultBranch is what the agent falls back to when Ref is empty.
	DefaultBranch string `json:"default_branch"`

	// Branch and Revision are the pre-ref spelling of Ref, kept so a
	// client or a script written against the earlier API keeps working.
	// Deprecated: send Ref.
	Branch   string `json:"branch,omitempty"`
	Revision string `json:"revision,omitempty"`
}
```

</details>

<details>
<summary><code>type RequestError</code></summary>

```go
// RequestError is an error carrying the HTTP status to report it with.
type RequestError struct {
	StatusCode int

	Err error
}
```

</details>

<details>
<summary><code>type SSHKeyView</code></summary>

```go
// SSHKeyView is how a key is described to an operator: everything
// needed to recognize and install it, and none of the private half.
type SSHKeyView struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Host        string `json:"host"`
	PublicKey   string `json:"public_key"`
	Fingerprint string `json:"fingerprint"`
	IsActive    bool   `json:"is_active"`
	LastUsedAt  string `json:"last_used_at,omitempty"`
}
```

</details>

<details>
<summary><code>type TokenResponse</code></summary>

```go
// TokenResponse is what the login and refresh endpoints return. It is
// exactly what the CLI persists in ~/.atkins/credentials.json.
type TokenResponse struct {
	UserID       string `json:"user_id"`
	Username     string `json:"username"`
	Token        string `json:"token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresAt    int64  `json:"expires_at"`
}
```

</details>

<details>
<summary><code>type TriggerRequest</code></summary>

```go
// TriggerRequest is the body of POST /api/repository/{repositoryID}/trigger.
// It is the "POST a job name to a project endpoint" trigger: a cron, a
// webhook receiver or another job can queue work for a repository the
// server already knows, without a checkout of its own.
type TriggerRequest struct {
	// Job is the atkins job to run, e.g. "analyze". Either this or
	// Command is required; Job is turned into `atkins <job>`.
	Job string `json:"job"`

	// Command overrides the whole invocation when a bare job name is
	// not enough.
	Command string `json:"command"`

	// WorkingDirectory is relative to the repository root.
	WorkingDirectory string `json:"working_directory"`

	// Ref pins what gets checked out: a branch, a tag, a commit sha or
	// a fully qualified refname. Empty lets the agent resolve the
	// repository's default branch when the job runs, which is what a
	// "run this nightly" trigger wants.
	Ref string `json:"ref"`

	// CloneDepth limits the history of the work tree. 0 is all of it.
	// A fan-out over tags usually wants 1: each child builds one commit
	// and never looks behind it.
	CloneDepth int64 `json:"clone_depth"`

	// Branch and Revision are the pre-ref spelling of Ref.
	// Deprecated: send Ref.
	Branch   string `json:"branch"`
	Revision string `json:"revision"`

	// Params are handed to the job as ATKINS_JOB_PARAMS. This is how a
	// dispatching job passes a tag or a commit to each child.
	Params map[string]any `json:"params"`

	Labels []string `json:"labels"`

	// Artefacts are glob patterns the agent collects after the command
	// exits. This is what lets a cron collect `coverage/*.json` from a
	// pipeline that knows nothing about artefacts.
	Artefacts []string `json:"artefacts"`

	// ParentID records the job that triggered this one.
	ParentID string `json:"parent_id"`
}
```

</details>

<details>
<summary><code>type UserView</code></summary>

```go
// UserView is how a user is described over the API. It exists so a
// password hash can never reach a response by accident.
type UserView struct {
	ID       string `json:"id"`
	Email    string `json:"email"`
	Username string `json:"username"`
	FullName string `json:"full_name"`
	IsAdmin  bool   `json:"is_admin"`
	IsActive bool   `json:"is_active"`
	IsAgent  bool   `json:"is_agent"`
}
```

</details>

<details>
<summary><code>type WhoamiResponse</code></summary>

```go
// WhoamiResponse describes the authenticated user.
type WhoamiResponse struct {
	UserID   string `json:"user_id"`
	Email    string `json:"email"`
	Username string `json:"username"`
	FullName string `json:"full_name"`
	IsAdmin  bool   `json:"is_admin"`
}
```

</details>

## Consts

<details>
<summary><code>const DefaultTokenTTL</code></summary>

```go
// DefaultTokenTTL is the access token lifetime. It is short because the
// CLI holds a refresh token and renews silently; a leaked access token
// is only useful for the rest of the hour.
const DefaultTokenTTL = time.Hour
```

</details>

<details>
<summary><code>const HeaderArtefactChecksum</code></summary>

```go
// HeaderArtefactChecksum carries the SHA256 the agent computed while
// reading the file. It is optional; when present the server compares it
// against what actually arrived, so a truncated upload is a 400 rather
// than an artefact nobody can use.
const HeaderArtefactChecksum = "X-Atkins-Checksum"
```

</details>

## Function symbols

- `func NewHandlers (opts Options) *Handlers`
- `func NewSSHKeyView (key model.SSHKey) SSHKeyView`
- `func NewUserView (user *model.User) UserView`
- `func SSHKeyViews (keys []model.SSHKey) []SSHKeyView`
- `func WriteArtefact (w http.ResponseWriter, artefact *model.JobArtefact, contents io.Reader)`
- `func (*Handlers) AgentPolicy (w http.ResponseWriter, r *http.Request)`
- `func (*Handlers) AgentSSHKeys (w http.ResponseWriter, r *http.Request)`
- `func (*Handlers) AppendJobLog (w http.ResponseWriter, r *http.Request)`
- `func (*Handlers) CancelJob (w http.ResponseWriter, r *http.Request)`
- `func (*Handlers) ClaimJob (w http.ResponseWriter, r *http.Request)`
- `func (*Handlers) CreateRule (w http.ResponseWriter, r *http.Request)`
- `func (*Handlers) CreateSSHKey (w http.ResponseWriter, r *http.Request)`
- `func (*Handlers) DeleteRule (w http.ResponseWriter, r *http.Request)`
- `func (*Handlers) DeleteSSHKey (w http.ResponseWriter, r *http.Request)`
- `func (*Handlers) Dispatch (w http.ResponseWriter, r *http.Request)`
- `func (*Handlers) DownloadArtefact (w http.ResponseWriter, r *http.Request)`
- `func (*Handlers) Enrol (w http.ResponseWriter, r *http.Request)`
- `func (*Handlers) GetJob (w http.ResponseWriter, r *http.Request)`
- `func (*Handlers) GetJobLog (w http.ResponseWriter, r *http.Request)`
- `func (*Handlers) JobCheckout (w http.ResponseWriter, r *http.Request)`
- `func (*Handlers) JobHeartbeat (w http.ResponseWriter, r *http.Request)`
- `func (*Handlers) JobStatus (w http.ResponseWriter, r *http.Request)`
- `func (*Handlers) ListArtefacts (w http.ResponseWriter, r *http.Request)`
- `func (*Handlers) ListJobs (w http.ResponseWriter, r *http.Request)`
- `func (*Handlers) ListRepositories (w http.ResponseWriter, r *http.Request)`
- `func (*Handlers) ListRules (w http.ResponseWriter, r *http.Request)`
- `func (*Handlers) ListSSHKeys (w http.ResponseWriter, r *http.Request)`
- `func (*Handlers) ListSettings (w http.ResponseWriter, r *http.Request)`
- `func (*Handlers) ListUsers (w http.ResponseWriter, r *http.Request)`
- `func (*Handlers) Login (w http.ResponseWriter, r *http.Request)`
- `func (*Handlers) Logout (w http.ResponseWriter, r *http.Request)`
- `func (*Handlers) Mount (r platform.Router)`
- `func (*Handlers) MountAdmin (r platform.Router)`
- `func (*Handlers) MountAgent (r platform.Router)`
- `func (*Handlers) RefreshToken (w http.ResponseWriter, r *http.Request)`
- `func (*Handlers) Register (w http.ResponseWriter, r *http.Request)`
- `func (*Handlers) ResetSetting (w http.ResponseWriter, r *http.Request)`
- `func (*Handlers) RetryJob (w http.ResponseWriter, r *http.Request)`
- `func (*Handlers) Trigger (w http.ResponseWriter, r *http.Request)`
- `func (*Handlers) UpdateRule (w http.ResponseWriter, r *http.Request)`
- `func (*Handlers) UpdateSSHKey (w http.ResponseWriter, r *http.Request)`
- `func (*Handlers) UpdateSetting (w http.ResponseWriter, r *http.Request)`
- `func (*Handlers) UpdateUser (w http.ResponseWriter, r *http.Request)`
- `func (*Handlers) UploadArtefact (w http.ResponseWriter, r *http.Request)`
- `func (*Handlers) Whoami (w http.ResponseWriter, r *http.Request)`
- `func (*RequestError) Error () string`
- `func (*RequestError) Unwrap () error`
- `func (RepositoryPayload) CheckoutRef () string`

### NewHandlers

NewHandlers returns Handlers configured from opts.

```go
func NewHandlers(opts Options) *Handlers
```

### NewSSHKeyView

NewSSHKeyView projects a stored key onto the operator-facing view.

It is exported so the admin pages project keys the same way the API
does. The projection is the guard against disclosing private
material, so there must be exactly one of it.

```go
func NewSSHKeyView(key model.SSHKey) SSHKeyView
```

### NewUserView

NewUserView projects a stored user.

Exported so the admin pages describe an account the same way the API
does, and so a page can no more render a password hash than a
response can.

```go
func NewUserView(user *model.User) UserView
```

### SSHKeyViews

SSHKeyViews projects a listing.

```go
func SSHKeyViews(keys []model.SSHKey) []SSHKeyView
```

### WriteArtefact

WriteArtefact streams an artefact as a download.

The web package serves the same bytes from the job page, and a
difference between the two responses would only ever be a bug.

```go
func WriteArtefact(w http.ResponseWriter, artefact *model.JobArtefact, contents io.Reader)
```

### AgentPolicy

AgentPolicy returns the repository policy in force.

```go
func (*Handlers) AgentPolicy(w http.ResponseWriter, r *http.Request)
```

### AgentSSHKeys

AgentSSHKeys returns the deploy keys an agent should install.

```go
func (*Handlers) AgentSSHKeys(w http.ResponseWriter, r *http.Request)
```

### AppendJobLog

AppendJobLog records a chunk of output for a job.

```go
func (*Handlers) AppendJobLog(w http.ResponseWriter, r *http.Request)
```

### CancelJob

CancelJob settles an unfinished job as cancelled.

```go
func (*Handlers) CancelJob(w http.ResponseWriter, r *http.Request)
```

### ClaimJob

ClaimJob leases the oldest pending job the agent can run.

A 204 means the queue holds nothing for this agent right now, which is
the common case for a polling worker and not an error.

```go
func (*Handlers) ClaimJob(w http.ResponseWriter, r *http.Request)
```

### CreateRule

CreateRule adds a repository allowlist rule.

```go
func (*Handlers) CreateRule(w http.ResponseWriter, r *http.Request)
```

### CreateSSHKey

CreateSSHKey stores a deploy key.

```go
func (*Handlers) CreateSSHKey(w http.ResponseWriter, r *http.Request)
```

### DeleteRule

DeleteRule removes a rule.

```go
func (*Handlers) DeleteRule(w http.ResponseWriter, r *http.Request)
```

### DeleteSSHKey

DeleteSSHKey removes a key.

```go
func (*Handlers) DeleteSSHKey(w http.ResponseWriter, r *http.Request)
```

### Dispatch

Dispatch records a job for the caller's repository and returns its ID.

```go
func (*Handlers) Dispatch(w http.ResponseWriter, r *http.Request)
```

### DownloadArtefact

DownloadArtefact writes the bytes of one artefact.

This is the authenticated way to read an artefact, and the one a
script should use. The job page has an unauthenticated route of its
own, on the same terms as the output it shows.

```go
func (*Handlers) DownloadArtefact(w http.ResponseWriter, r *http.Request)
```

### Enrol

Enrol exchanges the shared enrolment token for agent credentials.

Agents don't have passwords. An operator puts one secret in the
agent's environment, and the agent trades it for the same rotating
token pair a logged-in human holds. That keeps one long-lived secret
in one place instead of a password per worker.

```go
func (*Handlers) Enrol(w http.ResponseWriter, r *http.Request)
```

### GetJob

GetJob returns a single job.

```go
func (*Handlers) GetJob(w http.ResponseWriter, r *http.Request)
```

### GetJobLog

GetJobLog returns the recorded output for a job.

```go
func (*Handlers) GetJobLog(w http.ResponseWriter, r *http.Request)
```

### JobCheckout

JobCheckout records what the machine running a job checked out.

```go
func (*Handlers) JobCheckout(w http.ResponseWriter, r *http.Request)
```

### JobHeartbeat

JobHeartbeat extends the lease held on a job by whoever is running it.

```go
func (*Handlers) JobHeartbeat(w http.ResponseWriter, r *http.Request)
```

### JobStatus

JobStatus settles a job into a terminal state.

```go
func (*Handlers) JobStatus(w http.ResponseWriter, r *http.Request)
```

### ListArtefacts

ListArtefacts returns the artefacts a job produced.

```go
func (*Handlers) ListArtefacts(w http.ResponseWriter, r *http.Request)
```

### ListJobs

ListJobs returns jobs, newest first, narrowed by query parameters.

```go
func (*Handlers) ListJobs(w http.ResponseWriter, r *http.Request)
```

### ListRepositories

ListRepositories returns the repositories the server knows about.

A trigger needs a repository ID, and this is where a cron or a
webhook receiver finds it without guessing.

```go
func (*Handlers) ListRepositories(w http.ResponseWriter, r *http.Request)
```

### ListRules

ListRules returns the repository allowlist.

```go
func (*Handlers) ListRules(w http.ResponseWriter, r *http.Request)
```

### ListSSHKeys

ListSSHKeys returns the deploy keys, without private material.

```go
func (*Handlers) ListSSHKeys(w http.ResponseWriter, r *http.Request)
```

### ListSettings

ListSettings returns every setting with its effective value.

```go
func (*Handlers) ListSettings(w http.ResponseWriter, r *http.Request)
```

### ListUsers

ListUsers returns every account.

```go
func (*Handlers) ListUsers(w http.ResponseWriter, r *http.Request)
```

### Login

Login exchanges email and password for an access and refresh token.

```go
func (*Handlers) Login(w http.ResponseWriter, r *http.Request)
```

### Logout

Logout revokes the session behind the presented credentials.

It accepts either an Authorization header or a refresh token in the
body, so a client whose access token has already expired can still
log out cleanly. Logging out twice is not an error.

```go
func (*Handlers) Logout(w http.ResponseWriter, r *http.Request)
```

### Mount

Mount registers the API routes on the given router.

```go
func (*Handlers) Mount(r platform.Router)
```

### MountAdmin

MountAdmin registers the administrative routes.

```go
func (*Handlers) MountAdmin(r platform.Router)
```

### MountAgent

MountAgent registers the routes only agents use.

```go
func (*Handlers) MountAgent(r platform.Router)
```

### RefreshToken

RefreshToken exchanges a refresh token for a new access token.

```go
func (*Handlers) RefreshToken(w http.ResponseWriter, r *http.Request)
```

### Register

Register creates a user.

```go
func (*Handlers) Register(w http.ResponseWriter, r *http.Request)
```

### ResetSetting

ResetSetting returns a setting to its default.

```go
func (*Handlers) ResetSetting(w http.ResponseWriter, r *http.Request)
```

### RetryJob

RetryJob queues a copy of a finished job.

A retry is a new job rather than a reset of the old one: the previous
attempt keeps its output and its outcome, which is the whole point of
looking at a failure after retrying it.

```go
func (*Handlers) RetryJob(w http.ResponseWriter, r *http.Request)
```

### Trigger

Trigger queues a job against a known repository.

```go
func (*Handlers) Trigger(w http.ResponseWriter, r *http.Request)
```

### UpdateRule

UpdateRule enables or disables a rule.

```go
func (*Handlers) UpdateRule(w http.ResponseWriter, r *http.Request)
```

### UpdateSSHKey

UpdateSSHKey enables or disables a key.

```go
func (*Handlers) UpdateSSHKey(w http.ResponseWriter, r *http.Request)
```

### UpdateSetting

UpdateSetting overrides one setting.

```go
func (*Handlers) UpdateSetting(w http.ResponseWriter, r *http.Request)
```

### UpdateUser

UpdateUser changes a user's administrative flags.

```go
func (*Handlers) UpdateUser(w http.ResponseWriter, r *http.Request)
```

### UploadArtefact

UploadArtefact stores a file a job produced.

The body is the file itself rather than a multipart form: an artefact
is bytes, and a raw body streams from the agent's disk to the
server's without either side building an envelope around it. The
name, the media type and the checksum are small enough to be
metadata, and travel as the `path` query parameter, `Content-Type`
and `X-Atkins-Checksum`.

```go
func (*Handlers) UploadArtefact(w http.ResponseWriter, r *http.Request)
```

### Whoami

Whoami reports the authenticated user. The CLI uses it to confirm a
stored credential still works.

```go
func (*Handlers) Whoami(w http.ResponseWriter, r *http.Request)
```

### Error

Error returns the underlying error message.

```go
func (*RequestError) Error() string
```

### Unwrap

Unwrap exposes the wrapped error to errors.Is/As.

```go
func (*RequestError) Unwrap() error
```

### CheckoutRef

CheckoutRef folds the deprecated branch and revision fields into the
single ref a job records. The more specific of the two wins: a
revision names one commit, a branch names wherever it has got to.

```go
func (RepositoryPayload) CheckoutRef() string
```

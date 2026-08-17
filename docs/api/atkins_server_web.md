# Package ./server/web

```go
import (
	"github.com/titpetric/atkins/server/web"
}
```

Package web serves the browser-facing pages of the atkins CI/CD
server.

Two surfaces live here, and they are deliberately different.

`atkins` prints one URL when it dispatches a job —
`<server>/job/{ULID}?t=<token>` — and that page is where the run is
watched: status, timing, the command, and the output the agent
captured. It takes no session, because the point of printing a URL in
a terminal is that pasting it into a browser just works; a private
instance checks the token in that URL instead.

`/admin/*` is the operator's face on `/api/admin/*`: repositories,
the allowlist, settings, users and deploy keys. Those pages need a
session and an admin account, and their forms carry a CSRF token.
See session.go.

## Types

```go
// Handlers serves the HTML pages.
type Handlers struct {
	jobs         *storage.JobStorage
	jobLogs      *storage.JobLogStorage
	artefacts    *storage.JobArtefactStorage
	repositories *storage.RepositoryStorage
	users        *storage.UserStorage
	sessions     *storage.SessionStorage
	rules        *storage.RepositoryRuleStorage
	settings     *storage.SettingStorage
	sshKeys      *storage.SSHKeyStorage

	// tokens mints and checks the per-job view tokens. It holds the
	// server's signing key, so a link stops working when the key is
	// rotated.
	tokens *auth.JWT

	// signingKey authenticates the session cookie and derives CSRF
	// tokens. It is the same secret tokens holds, kept raw because the
	// cookie and CSRF HMACs are computed directly rather than as JWTs.
	// Rotating it logs the browser out along with everything else.
	signingKey string

	// live is the running jobs' terminals: the browser reads output from
	// it and types back into it, and the agent is on the other side.
	live *stream.Hub
}
```

```go
// IndexPage is the view model for the front page.
type IndexPage struct {
	Jobs []model.Job

	// Refresh is the auto-reload interval in seconds. Zero for a page
	// that will not change on its own.
	Refresh int

	// Private is set when the listing is withheld for want of a session,
	// so the page can offer a way in rather than looking like an
	// instance nobody has ever used.
	Private bool

	// Scoped is set when the listing is the signed-in user's own runs
	// rather than everything on the instance.
	Scoped bool

	// SignedIn is set when the visitor has a session, which decides
	// whether the page offers a sign-in link or an admin one.
	SignedIn bool
}
```

```go
// JobPage is the view model for the job page.
type JobPage struct {
	Job        *model.Job
	Repository *model.Repository
	Children   []model.Job
	Log        []model.JobLog
	Artefacts  []model.JobArtefact

	// Refresh is the auto-reload interval in seconds. Zero for a
	// settled job, which never changes again.
	Refresh int
}
```

```go
// Links builds the URLs a page points at.
// It exists because a component is a package-level function and cannot
// reach the handler that rendered it, while whether a link needs a view
// token is a runtime setting. Passing this in keeps the decision in one
// place and lets a render test construct one directly.
type Links struct {
	tokens *auth.JWT
	public bool
}
```

```go
// LogSpan is a run of output that shares one appearance.
type LogSpan struct {
	Text string

	// Class is the space-separated class list for the run, empty for
	// output in the page's own colour.
	Class string
}
```

```go
// Options is passed from the server module scope.
type Options struct {
	JobStorage            *storage.JobStorage
	JobLogStorage         *storage.JobLogStorage
	JobArtefactStorage    *storage.JobArtefactStorage
	RepositoryStorage     *storage.RepositoryStorage
	UserStorage           *storage.UserStorage
	SessionStorage        *storage.SessionStorage
	RepositoryRuleStorage *storage.RepositoryRuleStorage
	SettingStorage        *storage.SettingStorage
	SSHKeyStorage         *storage.SSHKeyStorage

	// SigningKey is the server's access token signing key. It derives
	// the per-job view tokens, and without it the admin pages refuse
	// every session rather than trusting an unauthenticated cookie.
	SigningKey string

	// Stream is the live half of a job: the terminal page reads output
	// from it as it arrives and posts keystrokes back into it.
	Stream *stream.Hub
}
```

```go
// Page is the state every admin template needs whatever it is showing:
// who is signed in, the token its forms have to echo, which nav entry
// is current, and whatever the last redirect had to say.
//
// It is exported because a template cannot reach a field promoted
// through an unexported embedded one.
type Page struct {
	User    api.UserView
	CSRF    string
	Section string
	Notice  string
	Error   string
}
```

## Consts

```go
// ViewTokenParam is the query parameter carrying a job's view token.
// The API builds the same links, so the name lives with the model both
// sides share.
const ViewTokenParam = model.ViewTokenParam
```

## Function symbols

- `func NewHandlers (opts Options) (*Handlers, error)`
- `func (*Handlers) Allowlist (w http.ResponseWriter, r *http.Request, current *session)`
- `func (*Handlers) Artefact (w http.ResponseWriter, r *http.Request)`
- `func (*Handlers) Asset (w http.ResponseWriter, r *http.Request)`
- `func (*Handlers) CreateProject (w http.ResponseWriter, r *http.Request, current *session)`
- `func (*Handlers) CreateRule (w http.ResponseWriter, r *http.Request, current *session)`
- `func (*Handlers) CreateSSHKey (w http.ResponseWriter, r *http.Request, current *session)`
- `func (*Handlers) Index (w http.ResponseWriter, r *http.Request)`
- `func (*Handlers) InputJob (w http.ResponseWriter, r *http.Request)`
- `func (*Handlers) Job (w http.ResponseWriter, r *http.Request)`
- `func (*Handlers) Login (w http.ResponseWriter, r *http.Request)`
- `func (*Handlers) LoginForm (w http.ResponseWriter, r *http.Request)`
- `func (*Handlers) Logout (w http.ResponseWriter, r *http.Request)`
- `func (*Handlers) Mount (r platform.Router)`
- `func (*Handlers) Project (w http.ResponseWriter, r *http.Request, current *session)`
- `func (*Handlers) Projects (w http.ResponseWriter, r *http.Request, current *session)`
- `func (*Handlers) RefreshProject (w http.ResponseWriter, r *http.Request, current *session)`
- `func (*Handlers) Repositories (w http.ResponseWriter, r *http.Request, current *session)`
- `func (*Handlers) RunProject (w http.ResponseWriter, r *http.Request, current *session)`
- `func (*Handlers) SSHKeys (w http.ResponseWriter, r *http.Request, current *session)`
- `func (*Handlers) Settings (w http.ResponseWriter, r *http.Request, current *session)`
- `func (*Handlers) Setup (w http.ResponseWriter, r *http.Request)`
- `func (*Handlers) SetupForm (w http.ResponseWriter, r *http.Request)`
- `func (*Handlers) StreamJob (w http.ResponseWriter, r *http.Request)`
- `func (*Handlers) Terminal (w http.ResponseWriter, r *http.Request)`
- `func (*Handlers) TriggerRepository (w http.ResponseWriter, r *http.Request, current *session)`
- `func (*Handlers) UpdateProject (w http.ResponseWriter, r *http.Request, _ *session)`
- `func (*Handlers) UpdateRule (w http.ResponseWriter, r *http.Request, _ *session)`
- `func (*Handlers) UpdateSSHKey (w http.ResponseWriter, r *http.Request, _ *session)`
- `func (*Handlers) UpdateSetting (w http.ResponseWriter, r *http.Request, current *session)`
- `func (*Handlers) UpdateUser (w http.ResponseWriter, r *http.Request, _ *session)`
- `func (*Handlers) Users (w http.ResponseWriter, r *http.Request, current *session)`
- `func (Links) Artefact (jobID,artefactID string) string`
- `func (Links) Job (jobID string) string`
- `func (Links) Listing () bool`
- `func (Links) Terminal (jobID string) string`

### NewHandlers

NewHandlers returns Handlers for the page routes.

The error is always nil. It survives from when the templates were
parsed at construction: templ compiles them instead, so a broken page
is a build failure rather than a start-up one. The signature stays so
callers do not have to change when something here does need to fail.

```go
func NewHandlers(opts Options) (*Handlers, error)
```

### Allowlist

Allowlist lists the repository rules.

```go
func (*Handlers) Allowlist(w http.ResponseWriter, r *http.Request, current *session)
```

### Artefact

Artefact serves the bytes of one artefact, as a download.

```go
func (*Handlers) Artefact(w http.ResponseWriter, r *http.Request)
```

### Asset

Asset serves a vendored browser asset.

It is a hand-written handler rather than http.FileServer over the
embedded FS because the FS holds a README and a licence too, and a
server should serve what it meant to serve. The allowlist is three
entries long and says so.

```go
func (*Handlers) Asset(w http.ResponseWriter, r *http.Request)
```

### CreateProject

CreateProject adds a project and queues the job that reads its
pipeline.

```go
func (*Handlers) CreateProject(w http.ResponseWriter, r *http.Request, current *session)
```

### CreateRule

CreateRule adds an allowlist rule.

```go
func (*Handlers) CreateRule(w http.ResponseWriter, r *http.Request, current *session)
```

### CreateSSHKey

CreateSSHKey stores a deploy key.

```go
func (*Handlers) CreateSSHKey(w http.ResponseWriter, r *http.Request, current *session)
```

### Index

Index lists recent jobs.

A private instance scopes the listing to whoever is signed in, the
way /api/job does for a bearer token: an owner who lost the URL atkins
printed can find the job here and the link carries its view token. A
visitor with no session is told where to sign in — the page still
answers, because it is the server's front door and a health check
probing it should find a server rather than a refusal.

```go
func (*Handlers) Index(w http.ResponseWriter, r *http.Request)
```

### InputJob

InputJob queues what somebody typed at a running job.

It answers 204 whatever happens to the bytes. The queue is bounded and
the agent may have stopped collecting a moment ago, and a terminal
that popped up an error every time a keystroke arrived a moment too
late would be unusable; what a person sees is the same thing they see
in a real terminal, which is that the character did not echo.

```go
func (*Handlers) InputJob(w http.ResponseWriter, r *http.Request)
```

### Job

Job renders one job: its status, the command, and captured output.

```go
func (*Handlers) Job(w http.ResponseWriter, r *http.Request)
```

### Login

Login exchanges an email and password for a session cookie.

```go
func (*Handlers) Login(w http.ResponseWriter, r *http.Request)
```

### LoginForm

LoginForm renders the login page.

```go
func (*Handlers) LoginForm(w http.ResponseWriter, r *http.Request)
```

### Logout

Logout revokes the session the cookie names and clears it.

```go
func (*Handlers) Logout(w http.ResponseWriter, r *http.Request)
```

### Mount

Mount registers the page routes.

The job pages have no session to check: the browser reading them never
logged in. What a private instance checks instead is the per-job view
token in the URL atkins printed, which keeps the one thing that has
to stay true — a pasted URL opens the job — without also handing the
job to anyone who guesses a ULID.

The artefact download shares that trust model rather than a weaker or
a stronger one. A file a job produced is no more sensitive than the
output that produced it, so it is reachable on exactly the terms the
page is: same token, same setting. A download link on a page a browser
can open must not be a link that fails, and a browser has no bearer
token to offer. `GET /api/job/{id}/artefact/{id}` is the authenticated
door for scripts.

Everything under `/admin` is gated on an admin session instead. Those
pages are not reached from a printed URL, so there is nothing to
preserve by leaving them open.

```go
func (*Handlers) Mount(r platform.Router)
```

### Project

Project renders one project: what it is, what it can run, and what it
has run.

```go
func (*Handlers) Project(w http.ResponseWriter, r *http.Request, current *session)
```

### Projects

Projects lists what the instance has.

```go
func (*Handlers) Projects(w http.ResponseWriter, r *http.Request, current *session)
```

### RefreshProject

RefreshProject re-reads a project's pipeline.

It is a button because a pipeline changes when the code does, and the
server has no way to know that it has: nothing tells it when somebody
pushes. Pressing it is cheap — the agent has the repository cached and
the job does nothing but list.

```go
func (*Handlers) RefreshProject(w http.ResponseWriter, r *http.Request, current *session)
```

### Repositories

Repositories lists the known repositories with their last job.

```go
func (*Handlers) Repositories(w http.ResponseWriter, r *http.Request, current *session)
```

### RunProject

RunProject queues one of the project's jobs and opens its terminal.

What may be run is what the listing names. The form's job id is looked
up in the cached tree rather than pasted into a command, which is what
keeps this a menu of a project's own jobs instead of a box that runs
shell on an agent.

```go
func (*Handlers) RunProject(w http.ResponseWriter, r *http.Request, current *session)
```

### SSHKeys

SSHKeys lists the deploy keys.

```go
func (*Handlers) SSHKeys(w http.ResponseWriter, r *http.Request, current *session)
```

### Settings

Settings renders every registered setting with its effective value.

```go
func (*Handlers) Settings(w http.ResponseWriter, r *http.Request, current *session)
```

### Setup

Setup creates the first account and signs the browser in as it.

The emptiness of the user table is checked again here rather than
trusted from the form: two people opening /setup on a new instance
should produce one admin and one "somebody got there first", not two
admins because both pages were rendered before either was submitted.

```go
func (*Handlers) Setup(w http.ResponseWriter, r *http.Request)
```

### SetupForm

SetupForm renders the first-run form, or sends a visitor on when there
is nothing left to set up.

```go
func (*Handlers) SetupForm(w http.ResponseWriter, r *http.Request)
```

### StreamJob

StreamJob sends a job's output as it arrives.

The stored rows are replayed first, so a browser that connects late —
or reconnects — sees the run from the beginning rather than from
whenever it happened to arrive. The sequence numbers make the join
clean: the live feed is subscribed to before the table is read, and
anything it offers at or below the last replayed sequence is a chunk
the replay already covered.

```go
func (*Handlers) StreamJob(w http.ResponseWriter, r *http.Request)
```

### Terminal

Terminal renders the live view of one job.

```go
func (*Handlers) Terminal(w http.ResponseWriter, r *http.Request)
```

### TriggerRepository

TriggerRepository queues a job against a known repository.

This is the form behind `POST /api/repository/{id}/trigger`: copying
a repository ID into a curl invocation to run a nightly by hand is
exactly the sort of thing a page is for.

```go
func (*Handlers) TriggerRepository(w http.ResponseWriter, r *http.Request, current *session)
```

### UpdateProject

UpdateProject saves the project's details.

```go
func (*Handlers) UpdateProject(w http.ResponseWriter, r *http.Request, _ *session)
```

### UpdateRule

UpdateRule enables, disables or removes a rule.

```go
func (*Handlers) UpdateRule(w http.ResponseWriter, r *http.Request, _ *session)
```

### UpdateSSHKey

UpdateSSHKey activates, deactivates or removes a key.

```go
func (*Handlers) UpdateSSHKey(w http.ResponseWriter, r *http.Request, _ *session)
```

### UpdateSetting

UpdateSetting overrides or resets one setting.

```go
func (*Handlers) UpdateSetting(w http.ResponseWriter, r *http.Request, current *session)
```

### UpdateUser

UpdateUser toggles one flag on one account.

```go
func (*Handlers) UpdateUser(w http.ResponseWriter, r *http.Request, _ *session)
```

### Users

Users lists accounts.

```go
func (*Handlers) Users(w http.ResponseWriter, r *http.Request, current *session)
```

### Artefact

Artefact is where an artefact downloads from.

It carries the token of the job the artefact belongs to, so a
download is reachable exactly when the page listing it is.

```go
func (Links) Artefact(jobID, artefactID string) string
```

### Job

Job is where a job page lives, token and all.

Every link a page renders goes through here, so following a parent or
a child from a job you were given a link to keeps working: each hop
carries the token for the job it points at, and never the token for
the job it came from.

```go
func (Links) Job(jobID string) string
```

### Listing

Listing reports whether the front page lists anything to a visitor
with no session, which is what a job page opened from a printed URL
has. A signed-in visitor gets a listing either way, but the markup
cannot tell one from the other and a link to an empty page is worse
than no link.

```go
func (Links) Listing() bool
```

### Terminal

Terminal is where a job is watched as it runs, token and all.

It is a second page onto the same job rather than a replacement for
the first: the job page is a document, readable after the fact and
without scripting, and the terminal is a terminal. Both are reachable
on the same terms, because they show the same output.

```go
func (Links) Terminal(jobID string) string
```

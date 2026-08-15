# Package ./server/web

```go
import (
	"github.com/titpetric/atkins/server/web"
}
```

Package web serves the browser-facing pages of the atkins CI/CD
server.

There is deliberately very little here. `atkins` prints one URL when
it dispatches a job — `<server>/job/{ULID}?t=<token>` — and that page
is where the run is watched: status, timing, the command, and the
output the agent captured. Everything else lives behind the JSON API.

## Types

```go
// Handlers serves the HTML pages.
type Handlers struct {
	jobs         *storage.JobStorage
	jobLogs      *storage.JobLogStorage
	artefacts    *storage.JobArtefactStorage
	repositories *storage.RepositoryStorage
	settings     *storage.SettingStorage

	// tokens mints and checks the per-job view tokens. It holds the
	// server's signing key, so a link stops working when the key is
	// rotated.
	tokens *auth.JWT

	templates *template.Template
}
```

```go
// IndexPage is the view model for the front page.
type IndexPage struct {
	Jobs []model.Job

	// Refresh is the auto-reload interval in seconds. Zero for a page
	// that will not change on its own.
	Refresh int

	// Private is set when the listing is withheld, so the page can say
	// why rather than looking like an instance nobody has ever used.
	Private bool
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
// Options is passed from the server module scope.
type Options struct {
	JobStorage         *storage.JobStorage
	JobLogStorage      *storage.JobLogStorage
	JobArtefactStorage *storage.JobArtefactStorage
	RepositoryStorage  *storage.RepositoryStorage
	SettingStorage     *storage.SettingStorage

	// SigningKey derives the per-job view tokens.
	SigningKey string
}
```

## Consts

```go
// ViewTokenParam is the query parameter carrying a job's view token.
// One letter, because it rides along in a URL a human copies out of a
// terminal.
const ViewTokenParam = "t"
```

## Function symbols

- `func NewHandlers (opts Options) (*Handlers, error)`
- `func (*Handlers) Artefact (w http.ResponseWriter, r *http.Request)`
- `func (*Handlers) Index (w http.ResponseWriter, r *http.Request)`
- `func (*Handlers) Job (w http.ResponseWriter, r *http.Request)`
- `func (*Handlers) Mount (r platform.Router)`

### NewHandlers

NewHandlers returns Handlers with the page templates parsed.

```go
func NewHandlers(opts Options) (*Handlers, error)
```

### Artefact

Artefact serves the bytes of one artefact, as a download.

```go
func (*Handlers) Artefact(w http.ResponseWriter, r *http.Request)
```

### Index

Index lists recent jobs.

A private instance lists nothing: no per-job token can scope a
listing and there is no session to scope it by, so the listing lives
on /api/job where the caller is authenticated. The page still
answers, and says why — it is the server's front door, and a health
check probing it should find a server rather than a refusal.

```go
func (*Handlers) Index(w http.ResponseWriter, r *http.Request)
```

### Job

Job renders one job: its status, the command, and captured output.

```go
func (*Handlers) Job(w http.ResponseWriter, r *http.Request)
```

### Mount

Mount registers the page routes.

No page here has a session to check: the browser reading them never
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

```go
func (*Handlers) Mount(r platform.Router)
```

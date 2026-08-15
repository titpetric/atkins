# Package ./server/web

```go
import (
	"github.com/titpetric/atkins/server/web"
}
```

Package web serves the browser-facing pages of the atkins CI/CD
server.

There is deliberately very little here. `atkins` prints one URL when
it dispatches a job — `<server>/job/{ULID}` — and that page is where
the run is watched: status, timing, the command, and the output the
agent captured. Everything else lives behind the JSON API.

## Types

```go
// Handlers serves the HTML pages.
type Handlers struct {
	jobs         *storage.JobStorage
	jobLogs      *storage.JobLogStorage
	repositories *storage.RepositoryStorage

	templates *template.Template
}
```

```go
// JobPage is the view model for the job page.
type JobPage struct {
	Job        *model.Job
	Repository *model.Repository
	Children   []model.Job
	Log        []model.JobLog

	// Refresh is the auto-reload interval in seconds. Zero for a
	// settled job, which never changes again.
	Refresh int
}
```

```go
// Options is passed from the server module scope.
type Options struct {
	JobStorage        *storage.JobStorage
	JobLogStorage     *storage.JobLogStorage
	RepositoryStorage *storage.RepositoryStorage
}
```

## Function symbols

- `func NewHandlers (opts Options) (*Handlers, error)`
- `func (*Handlers) Index (w http.ResponseWriter, r *http.Request)`
- `func (*Handlers) Job (w http.ResponseWriter, r *http.Request)`
- `func (*Handlers) Mount (r platform.Router)`

### NewHandlers

NewHandlers returns Handlers with the page templates parsed.

```go
func NewHandlers(opts Options) (*Handlers, error)
```

### Index

Index lists recent jobs.

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

The pages are readable without a session. A job URL carries a ULID
that is not enumerable in practice, and the point of printing one in
a terminal is that pasting it into a browser just works. Run the
server behind your own auth if the output is sensitive.

```go
func (*Handlers) Mount(r platform.Router)
```

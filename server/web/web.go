// Package web serves the browser-facing pages of the atkins CI/CD
// server.
//
// There is deliberately very little here. `atkins` prints one URL when
// it dispatches a job — `<server>/job/{ULID}` — and that page is where
// the run is watched: status, timing, the command, and the output the
// agent captured. Everything else lives behind the JSON API.
package web

import (
	"database/sql"
	"errors"
	"html/template"
	"net/http"
	"time"

	"github.com/titpetric/platform"

	"github.com/titpetric/atkins/server/model"
	"github.com/titpetric/atkins/server/storage"
)

// Handlers serves the HTML pages.
type Handlers struct {
	jobs         *storage.JobStorage
	jobLogs      *storage.JobLogStorage
	repositories *storage.RepositoryStorage

	templates *template.Template
}

// Options is passed from the server module scope.
type Options struct {
	JobStorage        *storage.JobStorage
	JobLogStorage     *storage.JobLogStorage
	RepositoryStorage *storage.RepositoryStorage
}

// NewHandlers returns Handlers with the page templates parsed.
func NewHandlers(opts Options) (*Handlers, error) {
	templates, err := template.New("").Funcs(functions()).ParseFS(files, "templates/*.html")
	if err != nil {
		return nil, err
	}

	return &Handlers{
		jobs:         opts.JobStorage,
		jobLogs:      opts.JobLogStorage,
		repositories: opts.RepositoryStorage,
		templates:    templates,
	}, nil
}

// Mount registers the page routes.
//
// The pages are readable without a session. A job URL carries a ULID
// that is not enumerable in practice, and the point of printing one in
// a terminal is that pasting it into a browser just works. Run the
// server behind your own auth if the output is sensitive.
func (h *Handlers) Mount(r platform.Router) {
	r.Get("/", h.Index)
	r.Get("/job/{jobID}", h.Job)
}

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

// Index lists recent jobs.
func (h *Handlers) Index(w http.ResponseWriter, r *http.Request) {
	jobs, err := h.jobs.List(r.Context(), storage.ListFilter{Limit: 50})
	if err != nil {
		h.fail(w, r, http.StatusInternalServerError, err)
		return
	}

	h.render(w, r, "index.html", struct {
		Jobs    []model.Job
		Refresh int
	}{Jobs: jobs, Refresh: 5})
}

// Job renders one job: its status, the command, and captured output.
func (h *Handlers) Job(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	job, err := h.jobs.Get(ctx, platform.URLParam(r, "jobID"))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			h.fail(w, r, http.StatusNotFound, errors.New("no such job"))
			return
		}
		h.fail(w, r, http.StatusInternalServerError, err)
		return
	}

	page := &JobPage{Job: job}
	if !job.IsTerminal() {
		// Poll while there is still something to see.
		page.Refresh = 3
	}

	if repository, err := h.repositories.Get(ctx, job.RepositoryID); err == nil {
		page.Repository = repository
	}
	if entries, err := h.jobLogs.List(ctx, job.ID); err == nil {
		page.Log = entries
	}
	if children, err := h.jobs.List(ctx, storage.ListFilter{RootID: job.RootID}); err == nil {
		page.Children = childrenOf(children, job.ID)
	}

	h.render(w, r, "job.html", page)
}

// childrenOf filters the job tree down to direct children.
func childrenOf(jobs []model.Job, parentID string) []model.Job {
	children := make([]model.Job, 0, len(jobs))
	for _, job := range jobs {
		if job.ParentID == parentID {
			children = append(children, job)
		}
	}
	return children
}

// render writes a template, reporting failures as a 500 rather than a
// half-written page.
func (h *Handlers) render(w http.ResponseWriter, r *http.Request, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if err := h.templates.ExecuteTemplate(w, name, data); err != nil {
		h.fail(w, r, http.StatusInternalServerError, err)
	}
}

// fail renders the error page.
func (h *Handlers) fail(w http.ResponseWriter, r *http.Request, status int, err error) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)

	_ = h.templates.ExecuteTemplate(w, "error.html", struct {
		Status  int
		Message string
	}{Status: status, Message: err.Error()})
}

// functions are the template helpers.
func functions() template.FuncMap {
	return template.FuncMap{
		"stamp":    stamp,
		"duration": duration,
	}
}

// stamp formats a nullable timestamp.
func stamp(value *time.Time) string {
	if value == nil {
		return "—"
	}
	return value.UTC().Format("2006-01-02 15:04:05 MST")
}

// duration reports how long a job has been running, or how long it ran.
func duration(job *model.Job) string {
	if job.StartedAt == nil {
		return "—"
	}

	end := time.Now()
	if job.FinishedAt != nil {
		end = *job.FinishedAt
	}

	return end.Sub(*job.StartedAt).Truncate(time.Millisecond).String()
}

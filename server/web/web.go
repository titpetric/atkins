// Package web serves the browser-facing pages of the atkins CI/CD
// server.
//
// There is deliberately very little here. `atkins` prints one URL when
// it dispatches a job — `<server>/job/{ULID}?t=<token>` — and that page
// is where the run is watched: status, timing, the command, and the
// output the agent captured. Everything else lives behind the JSON API.
package web

import (
	"database/sql"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"time"

	"github.com/titpetric/platform"

	"github.com/titpetric/atkins/server/api"
	"github.com/titpetric/atkins/server/auth"
	"github.com/titpetric/atkins/server/model"
	"github.com/titpetric/atkins/server/storage"
)

// ViewTokenParam is the query parameter carrying a job's view token.
// One letter, because it rides along in a URL a human copies out of a
// terminal.
const ViewTokenParam = "t"

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

// NewHandlers returns Handlers with the page templates parsed.
func NewHandlers(opts Options) (*Handlers, error) {
	handlers := &Handlers{
		jobs:         opts.JobStorage,
		jobLogs:      opts.JobLogStorage,
		artefacts:    opts.JobArtefactStorage,
		repositories: opts.RepositoryStorage,
		settings:     opts.SettingStorage,
		tokens:       auth.NewJWT(opts.SigningKey),
	}

	// The link helper is bound to these handlers rather than to the
	// package: whether a link needs a token is a runtime setting, and
	// the closure reads it when the page renders.
	templates, err := template.New("").Funcs(handlers.functions()).ParseFS(files, "templates/*.html")
	if err != nil {
		return nil, err
	}
	handlers.templates = templates

	return handlers, nil
}

// Mount registers the page routes.
//
// No page here has a session to check: the browser reading them never
// logged in. What a private instance checks instead is the per-job view
// token in the URL atkins printed, which keeps the one thing that has
// to stay true — a pasted URL opens the job — without also handing the
// job to anyone who guesses a ULID.
//
// The artefact download shares that trust model rather than a weaker or
// a stronger one. A file a job produced is no more sensitive than the
// output that produced it, so it is reachable on exactly the terms the
// page is: same token, same setting. A download link on a page a browser
// can open must not be a link that fails, and a browser has no bearer
// token to offer. `GET /api/job/{id}/artefact/{id}` is the authenticated
// door for scripts.
func (h *Handlers) Mount(r platform.Router) {
	r.Get("/", h.Index)
	r.Get("/job/{jobID}", h.Job)
	r.Get("/job/{jobID}/artefact/{artefactID}", h.Artefact)
}

// public reports whether the pages are open to anyone holding a URL.
//
// A nil setting store means the module has not wired one, and the safe
// reading of "no configuration" is the private one.
func (h *Handlers) public() bool {
	if h.settings == nil {
		return false
	}
	return h.settings.Get(model.SettingJobVisibility) == model.VisibilityPublic
}

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

// Index lists recent jobs.
//
// A private instance lists nothing: no per-job token can scope a
// listing and there is no session to scope it by, so the listing lives
// on /api/job where the caller is authenticated. The page still
// answers, and says why — it is the server's front door, and a health
// check probing it should find a server rather than a refusal.
func (h *Handlers) Index(w http.ResponseWriter, r *http.Request) {
	if !h.public() {
		h.render(w, r, "index.html", IndexPage{Private: true})
		return
	}

	jobs, err := h.jobs.List(r.Context(), storage.ListFilter{Limit: 50})
	if err != nil {
		h.fail(w, r, http.StatusInternalServerError, err)
		return
	}

	h.render(w, r, "index.html", IndexPage{Jobs: jobs, Refresh: 5})
}

// Job renders one job: its status, the command, and captured output.
func (h *Handlers) Job(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	jobID := platform.URLParam(r, "jobID")

	// The token is checked before the job is loaded, so a wrong one
	// tells a caller nothing about whether the job exists.
	if !h.public() && !h.tokens.ValidViewToken(jobID, platform.QueryParam(r, ViewTokenParam)) {
		h.fail(w, r, http.StatusForbidden,
			errors.New("this job link is missing its access token; use the whole URL atkins printed"))
		return
	}

	job, err := h.jobs.Get(ctx, jobID)
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
	if artefacts, err := h.artefacts.List(ctx, job.ID); err == nil {
		page.Artefacts = artefacts
	}
	if children, err := h.jobs.List(ctx, storage.ListFilter{RootID: job.RootID}); err == nil {
		page.Children = childrenOf(children, job.ID)
	}

	h.render(w, r, "job.html", page)
}

// Artefact serves the bytes of one artefact, as a download.
func (h *Handlers) Artefact(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	jobID := platform.URLParam(r, "jobID")

	// The same gate as the page that links here. A download reachable
	// without the token would make the artefacts of a private job
	// public, which is a stranger place to leave them than the output
	// they came from.
	if !h.public() && !h.tokens.ValidViewToken(jobID, platform.QueryParam(r, ViewTokenParam)) {
		h.fail(w, r, http.StatusForbidden,
			errors.New("this download link is missing its access token; open the job page it belongs to"))
		return
	}

	artefact, err := h.artefacts.Get(ctx, jobID, platform.URLParam(r, "artefactID"))
	if err != nil {
		h.fail(w, r, http.StatusNotFound, errors.New("no such artefact"))
		return
	}

	contents, err := h.artefacts.Open(ctx, artefact)
	if err != nil {
		// The row outlives its bytes once retention has swept them.
		h.fail(w, r, http.StatusNotFound, errors.New("this artefact has been removed by retention"))
		return
	}
	defer contents.Close()

	api.WriteArtefact(w, artefact, contents)
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
func (h *Handlers) functions() template.FuncMap {
	return template.FuncMap{
		"stamp":        stamp,
		"duration":     duration,
		"filesize":     filesize,
		"joblink":      h.jobLink,
		"artefactlink": h.artefactLink,
		"listing":      h.public,
	}
}

// artefactLink is where an artefact downloads from, token and all.
//
// It carries the token of the job the artefact belongs to, so a
// download is reachable exactly when the page listing it is.
func (h *Handlers) artefactLink(jobID, artefactID string) string {
	link := "/job/" + jobID + "/artefact/" + artefactID
	if h.public() {
		return link
	}

	token := h.tokens.ViewToken(jobID)
	if token == "" {
		return link
	}

	return link + "?" + ViewTokenParam + "=" + token
}

// filesize renders a byte count the way a person reads one.
func filesize(size int64) string {
	const unit = 1024

	if size < unit {
		return fmt.Sprintf("%d B", size)
	}

	value := float64(size)
	for _, suffix := range []string{"KB", "MB", "GB", "TB"} {
		value /= unit
		if value < unit {
			return fmt.Sprintf("%.1f %s", value, suffix)
		}
	}

	return fmt.Sprintf("%.1f PB", value/unit)
}

// jobLink is where a job page lives, token and all.
//
// Every link a page renders goes through here, so following a parent or
// a child from a job you were given a link to keeps working: each hop
// carries the token for the job it points at, and never the token for
// the job it came from.
func (h *Handlers) jobLink(jobID string) string {
	if h.public() {
		return "/job/" + jobID
	}

	token := h.tokens.ViewToken(jobID)
	if token == "" {
		return "/job/" + jobID
	}

	return "/job/" + jobID + "?" + ViewTokenParam + "=" + token
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

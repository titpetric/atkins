// Package web serves the browser-facing pages of the atkins CI/CD
// server.
//
// Two surfaces live here, and they are deliberately different.
//
// `atkins` prints one URL when it dispatches a job —
// `<server>/job/{ULID}?t=<token>` — and that page is where the run is
// watched: status, timing, the command, and the output the agent
// captured. It takes no session, because the point of printing a URL in
// a terminal is that pasting it into a browser just works; a private
// instance checks the token in that URL instead.
//
// `/admin/*` is the operator's face on `/api/admin/*`: repositories,
// the allowlist, settings, users and deploy keys. Those pages need a
// session and an admin account, and their forms carry a CSRF token.
// See session.go.
package web

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/a-h/templ"
	"github.com/titpetric/platform"

	"github.com/titpetric/atkins/server/api"
	"github.com/titpetric/atkins/server/auth"
	"github.com/titpetric/atkins/server/model"
	"github.com/titpetric/atkins/server/storage"
)

// ViewTokenParam is the query parameter carrying a job's view token.
// The API builds the same links, so the name lives with the model both
// sides share.
const ViewTokenParam = model.ViewTokenParam

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
}

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
}

// NewHandlers returns Handlers for the page routes.
//
// The error is always nil. It survives from when the templates were
// parsed at construction: templ compiles them instead, so a broken page
// is a build failure rather than a start-up one. The signature stays so
// callers do not have to change when something here does need to fail.
func NewHandlers(opts Options) (*Handlers, error) {
	handlers := &Handlers{
		jobs:         opts.JobStorage,
		jobLogs:      opts.JobLogStorage,
		artefacts:    opts.JobArtefactStorage,
		repositories: opts.RepositoryStorage,
		users:        opts.UserStorage,
		sessions:     opts.SessionStorage,
		rules:        opts.RepositoryRuleStorage,
		settings:     opts.SettingStorage,
		sshKeys:      opts.SSHKeyStorage,
		tokens:       auth.NewJWT(opts.SigningKey),
		signingKey:   opts.SigningKey,
	}

	return handlers, nil
}

// Mount registers the page routes.
//
// The job pages have no session to check: the browser reading them never
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
//
// Everything under `/admin` is gated on an admin session instead. Those
// pages are not reached from a printed URL, so there is nothing to
// preserve by leaving them open.
func (h *Handlers) Mount(r platform.Router) {
	r.Get("/", h.Index)
	r.Get("/job/{jobID}", h.Job)
	r.Get("/job/{jobID}/artefact/{artefactID}", h.Artefact)

	r.Get("/login", h.LoginForm)
	r.Post("/login", h.Login)
	r.Post("/logout", h.Logout)

	r.Get("/admin", h.admin(h.Repositories))
	r.Get("/admin/repository", h.admin(h.Repositories))
	r.Post("/admin/repository/{repositoryID}/trigger", h.submit(h.TriggerRepository))

	r.Get("/admin/allowlist", h.admin(h.Allowlist))
	r.Post("/admin/allowlist", h.submit(h.CreateRule))
	r.Post("/admin/allowlist/{ruleID}", h.submit(h.UpdateRule))

	r.Get("/admin/setting", h.admin(h.Settings))
	r.Post("/admin/setting", h.submit(h.UpdateSetting))

	r.Get("/admin/user", h.admin(h.Users))
	r.Post("/admin/user/{userID}", h.submit(h.UpdateUser))

	r.Get("/admin/ssh-key", h.admin(h.SSHKeys))
	r.Post("/admin/ssh-key", h.submit(h.CreateSSHKey))
	r.Post("/admin/ssh-key/{keyID}", h.submit(h.UpdateSSHKey))
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

// Index lists recent jobs.
//
// A private instance scopes the listing to whoever is signed in, the
// way /api/job does for a bearer token: an owner who lost the URL atkins
// printed can find the job here and the link carries its view token. A
// visitor with no session is told where to sign in — the page still
// answers, because it is the server's front door and a health check
// probing it should find a server rather than a refusal.
func (h *Handlers) Index(w http.ResponseWriter, r *http.Request) {
	page := IndexPage{Refresh: 5}
	filter := storage.ListFilter{Limit: 50}

	if !h.public() {
		current, err := h.authenticate(r)
		if err != nil {
			h.render(w, r, indexView(IndexPage{Private: true}, h.links()))
			return
		}

		page.SignedIn = true
		// An admin sees the instance and an agent works the whole queue,
		// which is the same rule the API scopes jobs by.
		if !current.User.IsAdmin && !current.User.IsAgent {
			filter.ViewerID = current.User.ID
			page.Scoped = true
		}
	}

	jobs, err := h.jobs.List(r.Context(), filter)
	if err != nil {
		h.fail(w, r, http.StatusInternalServerError, err)
		return
	}
	page.Jobs = jobs

	h.render(w, r, indexView(page, h.links()))
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

	h.render(w, r, jobView(page, h.links()))
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

// render writes a component, reporting failures as a 500 rather than a
// half-written page.
func (h *Handlers) render(w http.ResponseWriter, r *http.Request, page templ.Component) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if err := page.Render(r.Context(), w); err != nil {
		h.fail(w, r, http.StatusInternalServerError, err)
	}
}

// fail renders the error page.
func (h *Handlers) fail(w http.ResponseWriter, r *http.Request, status int, err error) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)

	_ = errorView(status, err.Error()).Render(r.Context(), w)
}

// Links builds the URLs a page points at.
//
// It exists because a component is a package-level function and cannot
// reach the handler that rendered it, while whether a link needs a view
// token is a runtime setting. Passing this in keeps the decision in one
// place and lets a render test construct one directly.
type Links struct {
	tokens *auth.JWT
	public bool
}

// links returns the link builder for the current setting.
func (h *Handlers) links() Links {
	return Links{tokens: h.tokens, public: h.public()}
}

// Listing reports whether the front page lists anything to a visitor
// with no session, which is what a job page opened from a printed URL
// has. A signed-in visitor gets a listing either way, but the markup
// cannot tell one from the other and a link to an empty page is worse
// than no link.
func (l Links) Listing() bool {
	return l.public
}

// Job is where a job page lives, token and all.
//
// Every link a page renders goes through here, so following a parent or
// a child from a job you were given a link to keeps working: each hop
// carries the token for the job it points at, and never the token for
// the job it came from.
func (l Links) Job(jobID string) string {
	return model.JobLink(jobID, l.token(jobID))
}

// Artefact is where an artefact downloads from.
//
// It carries the token of the job the artefact belongs to, so a
// download is reachable exactly when the page listing it is.
func (l Links) Artefact(jobID, artefactID string) string {
	return l.withToken("/job/"+jobID+"/artefact/"+artefactID, jobID)
}

// withToken appends a job's view token to a link, unless the instance
// is public or has no signing key to derive one from.
func (l Links) withToken(link, jobID string) string {
	token := l.token(jobID)
	if token == "" {
		return link
	}

	return link + "?" + ViewTokenParam + "=" + token
}

// token is the view token a link should carry, or empty when the
// instance is public or has no signing key to derive one from.
func (l Links) token(jobID string) string {
	if l.public || l.tokens == nil {
		return ""
	}
	return l.tokens.ViewToken(jobID)
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

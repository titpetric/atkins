package web

import (
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/titpetric/platform"

	"github.com/titpetric/atkins/server/model"
	"github.com/titpetric/atkins/server/pipeline"
	"github.com/titpetric/atkins/server/storage"
)

// A project is a repository somebody named. The rest of the server
// discovers repositories — a slug derived from whatever remote a
// dispatch reported — and that is the right way round for a machine
// already building something. It is the wrong way round for a person
// with a clone URL and nothing running yet, which is what these pages
// are for: paste an address, name it, and the instance has something in
// it.
//
// What a project adds beyond the row is the pipeline. Nothing here reads
// atkins.yml: the server has no checkout and giving it one would mean
// giving it the clone cache, the deploy keys and the allowlist a second
// time. It queues a job instead — `atkins --list --json`, run by an
// agent in a real checkout, uploaded as an artefact — and reads the tree
// out of that. The listing is therefore the listing of the code an agent
// would actually build, which is the only listing worth showing.

// pipelineArtefact is the file a listing job leaves behind.
const pipelineArtefact = "pipeline.json"

// pipelineCommand is the job queued to discover a project's pipeline.
//
// It redirects into the artefact directory rather than declaring a glob,
// so the file lands somewhere the agent already collects and nothing has
// to be cleaned out of the checkout afterwards.
//
// The failure branch is the part worth reading. `atkins --list` lists a
// pipeline it cannot fully resolve — a `task:` step naming a job that is
// not there, which is what a project depending on a skill the agent does
// not carry looks like — and then exits non-zero, having written the
// jobs that did resolve to stdout and the reason on stderr. Both halves
// are wanted: the file is the tree the picker renders, and the warning is
// in the job's log, which the project page links to.
//
// So a non-zero exit with a file in hand is not a failed listing, and
// this succeeds on it. A non-zero exit with nothing written is — there
// was no pipeline, no atkins, or no checkout — and the empty file is
// removed rather than uploaded, so it cannot be mistaken for a pipeline
// with no jobs in it.
const pipelineCommand = `atkins --list --json > "$ATKINS_ARTEFACTS/` + pipelineArtefact + `" || {
	if [ ! -s "$ATKINS_ARTEFACTS/` + pipelineArtefact + `" ]; then
		rm -f "$ATKINS_ARTEFACTS/` + pipelineArtefact + `"
		exit 1
	fi
	echo "atkins --list reported the errors above; listing what it did resolve."
}`

// projectsPage lists the projects and offers the form that adds one.
type projectsPage struct {
	Page

	Projects []projectRow

	// Policy is the repository policy in force. A project added under a
	// closed allowlist is a project whose runs will be refused, and the
	// page says so where somebody is about to add one.
	Policy string
}

// projectRow is one project with the state the listing shows.
type projectRow struct {
	model.Repository

	LastJob *model.Job

	// Allowed reports whether the policy in force admits this project.
	Allowed bool

	// Jobs is how many jobs the cached listing holds, so a row can say
	// whether its pipeline has been read yet without parsing it twice.
	Jobs int
}

// Name is what to call the project on a page: what somebody typed, or
// the tail of the slug for a repository that arrived from a dispatch and
// was never named.
func (r projectRow) Name() string {
	if r.Repository.Name != "" {
		return r.Repository.Name
	}
	return model.ProjectName(r.Slug)
}

// projectPage is one project: its details, its pipeline, and the form
// that runs a job from it.
type projectPage struct {
	Page

	Project model.Repository

	// Tree is the pipeline as a menu, nil when there is nothing to show
	// yet.
	Tree *pipeline.Tree

	// Listing is the job that produced (or is producing) the tree.
	Listing *model.Job

	// ListingLink is where to watch that job, token and all.
	ListingLink string

	// PipelineError is why there is no tree: the listing job failed, or
	// what it uploaded could not be read.
	PipelineError string

	// Jobs are the project's recent runs.
	Jobs []jobRow

	// Allowed reports whether the policy in force admits this project.
	Allowed bool

	// Refresh is the auto-reload interval in seconds, non-zero only
	// while a listing job is still running.
	Refresh int
}

// jobRow is one run with the link that opens it.
type jobRow struct {
	model.Job

	Link string
}

// Name is the project's display name; see projectRow.Name.
func (p projectPage) Name() string {
	if p.Project.Name != "" {
		return p.Project.Name
	}
	return model.ProjectName(p.Project.Slug)
}

// Projects lists what the instance has.
func (h *Handlers) Projects(w http.ResponseWriter, r *http.Request, current *session) {
	ctx := r.Context()

	repositories, err := h.repositories.List(ctx, 0)
	if err != nil {
		h.fail(w, r, http.StatusInternalServerError, err)
		return
	}

	policy := h.settings.Get(model.SettingRepositoryPolicy)

	rows := make([]projectRow, 0, len(repositories))
	for _, repository := range repositories {
		row := projectRow{Repository: repository, Allowed: true}

		// One query per project. This is an operator page loaded by
		// hand, not the dispatch path.
		if jobs, err := h.jobs.List(ctx, storage.ListFilter{RepositoryID: repository.ID, Limit: 1}); err == nil && len(jobs) > 0 {
			row.LastJob = &jobs[0]
		}
		if allowed, err := h.rules.AllowedUnderPolicy(ctx, policy, repository.Slug); err == nil {
			row.Allowed = allowed
		}
		if tree, err := pipeline.Parse([]byte(repository.Pipeline)); err == nil {
			row.Jobs = len(tree.Commands())
		}

		rows = append(rows, row)
	}

	h.render(w, r, projectsView(&projectsPage{
		Page:     h.page(r, current, sectionProjects),
		Projects: rows,
		Policy:   policy,
	}))
}

// CreateProject adds a project and queues the job that reads its
// pipeline.
func (h *Handlers) CreateProject(w http.ResponseWriter, r *http.Request, current *session) {
	const back = "/admin/project"

	remote := strings.TrimSpace(r.PostFormValue("remote_url"))
	if remote == "" {
		after(w, r, back, "", errors.New("a git clone address is required"))
		return
	}

	project, err := h.repositories.CreateProject(r.Context(), current.User.ID, storage.ProjectRequest{
		Name:             r.PostFormValue("name"),
		RemoteURL:        remote,
		DefaultBranch:    r.PostFormValue("default_branch"),
		Command:          r.PostFormValue("command"),
		Ref:              r.PostFormValue("ref"),
		WorkingDirectory: r.PostFormValue("working_directory"),
	})
	if err != nil {
		after(w, r, back, "", err)
		return
	}

	// Reading the pipeline is the next thing anybody wants and the one
	// thing they cannot do themselves, so it is queued rather than
	// offered as a button they would then have to press.
	if err := h.readPipeline(r, project, current.User.ID); err != nil {
		after(w, r, "/admin/project/"+project.ID, "", err)
		return
	}

	after(w, r, "/admin/project/"+project.ID, "Added. Reading the pipeline — an agent has to check the project out first.", nil)
}

// Project renders one project: what it is, what it can run, and what it
// has run.
func (h *Handlers) Project(w http.ResponseWriter, r *http.Request, current *session) {
	ctx := r.Context()

	project, err := h.repositories.Get(ctx, platform.URLParam(r, "repositoryID"))
	if err != nil {
		h.fail(w, r, http.StatusNotFound, errors.New("no such project"))
		return
	}

	page := &projectPage{
		Page:    h.page(r, current, sectionProjects),
		Project: *project,
		Allowed: true,
	}

	if allowed, err := h.rules.AllowedUnderPolicy(ctx, h.settings.Get(model.SettingRepositoryPolicy), project.Slug); err == nil {
		page.Allowed = allowed
	}

	h.resolvePipeline(r, project, page)

	if jobs, err := h.jobs.List(ctx, storage.ListFilter{RepositoryID: project.ID, Limit: 20}); err == nil {
		links := h.links()
		for _, job := range jobs {
			page.Jobs = append(page.Jobs, jobRow{Job: job, Link: links.Job(job.ID)})
		}
	}

	h.render(w, r, projectView(page))
}

// RefreshProject re-reads a project's pipeline.
//
// It is a button because a pipeline changes when the code does, and the
// server has no way to know that it has: nothing tells it when somebody
// pushes. Pressing it is cheap — the agent has the repository cached and
// the job does nothing but list.
func (h *Handlers) RefreshProject(w http.ResponseWriter, r *http.Request, current *session) {
	project, err := h.repositories.Get(r.Context(), platform.URLParam(r, "repositoryID"))
	if err != nil {
		after(w, r, "/admin/project", "", errors.New("no such project"))
		return
	}

	back := "/admin/project/" + project.ID

	if err := h.readPipeline(r, project, current.User.ID); err != nil {
		after(w, r, back, "", err)
		return
	}

	after(w, r, back, "Reading the pipeline.", nil)
}

// UpdateProject saves the project's details.
func (h *Handlers) UpdateProject(w http.ResponseWriter, r *http.Request, _ *session) {
	project, err := h.repositories.Get(r.Context(), platform.URLParam(r, "repositoryID"))
	if err != nil {
		after(w, r, "/admin/project", "", errors.New("no such project"))
		return
	}

	back := "/admin/project/" + project.ID

	if _, err := h.repositories.UpdateProject(r.Context(), project.ID, storage.ProjectRequest{
		Name:             r.PostFormValue("name"),
		DefaultBranch:    r.PostFormValue("default_branch"),
		Command:          r.PostFormValue("command"),
		Ref:              r.PostFormValue("ref"),
		WorkingDirectory: r.PostFormValue("working_directory"),
	}); err != nil {
		after(w, r, back, "", err)
		return
	}

	after(w, r, back, "Saved.", nil)
}

// RunProject queues one of the project's jobs and opens its terminal.
//
// What may be run is what the listing names. The form's job id is looked
// up in the cached tree rather than pasted into a command, which is what
// keeps this a menu of a project's own jobs instead of a box that runs
// shell on an agent.
func (h *Handlers) RunProject(w http.ResponseWriter, r *http.Request, current *session) {
	ctx := r.Context()

	project, err := h.repositories.Get(ctx, platform.URLParam(r, "repositoryID"))
	if err != nil {
		after(w, r, "/admin/project", "", errors.New("no such project"))
		return
	}

	back := "/admin/project/" + project.ID

	tree, err := pipeline.Parse([]byte(project.Pipeline))
	if err != nil {
		after(w, r, back, "", errors.New("this project's pipeline has not been read yet"))
		return
	}

	chosen, found := tree.Lookup(r.PostFormValue("job"))
	if !found {
		after(w, r, back, "", errors.New("that job is not in this project's pipeline"))
		return
	}

	// The allowlist governs a run started here exactly as it governs a
	// dispatch. The button is hidden for a project the policy refuses,
	// and a hidden button is not a check.
	allowed, err := h.rules.AllowedUnderPolicy(ctx, h.settings.Get(model.SettingRepositoryPolicy), project.Slug)
	if err != nil {
		after(w, r, back, "", err)
		return
	}
	if !allowed {
		after(w, r, back, "", model.ErrRepositoryNotAllowed)
		return
	}

	ref := strings.TrimSpace(r.PostFormValue("ref"))
	if ref == "" {
		ref = project.Ref
	}

	job, err := h.jobs.Create(ctx, storage.JobRequest{
		RepositoryID:     project.ID,
		UserID:           current.User.ID,
		Command:          chosen.Command,
		Ref:              ref,
		WorkingDirectory: project.WorkingDirectory,
		// The pipeline decides, not the form. A job that never reads
		// stdin does not get a keyboard because somebody posted a
		// checkbox.
		Interactive: chosen.Interactive,
	})
	if err != nil {
		after(w, r, back, "", err)
		return
	}

	// Straight to the terminal: the next thing anybody wants after
	// pressing run is to watch it.
	http.Redirect(w, r, h.links().Terminal(job.ID), http.StatusSeeOther)
}

// readPipeline queues the job that lists a project's jobs.
func (h *Handlers) readPipeline(r *http.Request, project *model.Repository, userID string) error {
	job, err := h.jobs.Create(r.Context(), storage.JobRequest{
		RepositoryID:     project.ID,
		UserID:           userID,
		Command:          pipelineCommand,
		Ref:              project.Ref,
		WorkingDirectory: project.WorkingDirectory,
		// Listing needs the pipeline file and nothing behind it.
		CloneDepth: 1,
	})
	if err != nil {
		return err
	}

	return h.repositories.SetPipelineJob(r.Context(), project.ID, job.ID)
}

// resolvePipeline fills in the page's pipeline: from the cache when
// there is one, from the listing job's artefact the first time, and with
// the reason there is neither when there is neither.
func (h *Handlers) resolvePipeline(r *http.Request, project *model.Repository, page *projectPage) {
	ctx := r.Context()

	if project.Pipeline != "" {
		tree, err := pipeline.Parse([]byte(project.Pipeline))
		if err != nil {
			page.PipelineError = err.Error()
			return
		}
		page.Tree = tree
		return
	}

	if project.PipelineJobID == "" {
		page.PipelineError = "This project's pipeline has not been read yet."
		return
	}

	listing, err := h.jobs.Get(ctx, project.PipelineJobID)
	if err != nil {
		page.PipelineError = "The job that was reading this pipeline is gone. Read it again."
		return
	}

	page.Listing = listing
	page.ListingLink = h.links().Job(listing.ID)

	if !listing.IsTerminal() {
		// Poll while there is still something to wait for.
		page.Refresh = 3
		return
	}
	if listing.Status != model.JobStatusPassed {
		// A job that got as far as listing anything passes, warnings and
		// all — see pipelineCommand. Reaching here means nothing was
		// listed at all: no checkout, no pipeline file, no atkins. The
		// job's output says which, so the page sends them there.
		page.PipelineError = "Reading the pipeline failed — its output says why."
		return
	}

	listed, err := h.pipelineArtefact(r, listing.ID)
	if err != nil {
		page.PipelineError = err.Error()
		return
	}

	tree, err := pipeline.Parse(listed)
	if err != nil {
		page.PipelineError = err.Error()
		return
	}

	// Cache it: the artefact is swept by retention and the tree is not.
	// A failure to write is not a failure to render — the page already
	// has what it needs, and the next load will try again.
	_ = h.repositories.SetPipeline(ctx, project.ID, string(listed))

	page.Tree = tree
}

// maxPipelineListing bounds what is read back out of an artefact.
//
// `atkins --list --json` on a large monorepo is measured in tens of
// kilobytes. This is the size at which the file is not that any more,
// and reading the rest of it into a page would be reading somebody
// else's mistake into memory.
const maxPipelineListing = 2 << 20

// pipelineArtefact reads the listing a job uploaded.
func (h *Handlers) pipelineArtefact(r *http.Request, jobID string) ([]byte, error) {
	ctx := r.Context()

	artefacts, err := h.artefacts.List(ctx, jobID)
	if err != nil {
		return nil, err
	}

	for _, artefact := range artefacts {
		if artefact.Path != pipelineArtefact {
			continue
		}

		contents, err := h.artefacts.Open(ctx, &artefact)
		if err != nil {
			return nil, errors.New("the listing this project produced has been removed by retention; read it again")
		}
		defer contents.Close()

		return io.ReadAll(io.LimitReader(contents, maxPipelineListing))
	}

	return nil, errors.New("the job that read this pipeline uploaded nothing; open it to see why")
}

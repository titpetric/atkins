package api

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/titpetric/platform"

	"github.com/titpetric/atkins/server/model"
	"github.com/titpetric/atkins/server/storage"
)

// DispatchRequest is the body of /api/dispatch.
//
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
}

// RepositoryPayload is the git detail a client reports about its checkout.
type RepositoryPayload struct {
	RemoteURL     string `json:"remote_url"`
	Branch        string `json:"branch"`
	Revision      string `json:"revision"`
	DefaultBranch string `json:"default_branch"`
}

// DispatchResponse is what /api/dispatch returns.
//
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
}

// JobStatusRequest is the body of /api/job/{jobID}/status.
type JobStatusRequest struct {
	Status   string `json:"status"`
	ExitCode int64  `json:"exit_code"`
	Error    string `json:"error"`
}

// ClaimRequest is the body of /api/job/claim.
type ClaimRequest struct {
	// AgentID identifies the worker taking the lease.
	AgentID string `json:"agent_id"`

	// Labels are what the agent can offer. A job requiring labels only
	// lands on an agent advertising all of them.
	Labels []string `json:"labels"`
}

// ClaimResponse is returned when an agent leases a job.
//
// The repository is included rather than referenced: the agent's very
// next act is to clone it, and making it ask again for the remote URL
// would be a round trip that buys nothing.
type ClaimResponse struct {
	Job        *model.Job        `json:"job"`
	Repository *model.Repository `json:"repository"`
}

// Dispatch records a job for the caller's repository and returns its ID.
func (s *Handlers) Dispatch(w http.ResponseWriter, r *http.Request) {
	s.respond(w, r, s.dispatch(w, r))
}

func (s *Handlers) dispatch(w http.ResponseWriter, r *http.Request) error {
	user, _, err := s.authenticateUser(r)
	if err != nil {
		return err
	}

	var req DispatchRequest
	if err := decode(r, &req); err != nil {
		return err
	}

	if strings.TrimSpace(req.Command) == "" {
		return requestError(http.StatusBadRequest, errors.New("command is required"))
	}

	slug := model.RepositorySlug(req.Repository.RemoteURL)
	if slug == "" {
		return requestError(http.StatusBadRequest, model.ErrInvalidRepository)
	}

	// Check the allowlist before the repository row is created: a
	// refused repository should leave no trace beyond the 403.
	allowed, err := s.allowedRepository(r, slug)
	if err != nil {
		return err
	}
	if !allowed {
		return requestError(http.StatusForbidden,
			fmt.Errorf("%w: %s", model.ErrRepositoryNotAllowed, slug))
	}

	repository, err := s.repositories.Ensure(r.Context(), user.ID, storage.RepositoryRequest{
		RemoteURL:     req.Repository.RemoteURL,
		DefaultBranch: req.Repository.DefaultBranch,
	})
	if err != nil {
		if errors.Is(err, model.ErrInvalidRepository) {
			return requestError(http.StatusBadRequest, err)
		}
		return err
	}

	params, err := encodeParams(req.Params)
	if err != nil {
		return requestError(http.StatusBadRequest, err)
	}

	job, err := s.jobs.Create(r.Context(), storage.JobRequest{
		ParentID:         req.ParentID,
		RepositoryID:     repository.ID,
		UserID:           user.ID,
		WorkingDirectory: cleanWorkingDirectory(req.WorkingDirectory),
		Command:          req.Command,
		Branch:           req.Repository.Branch,
		Revision:         req.Repository.Revision,
		Labels:           req.Labels,
		Params:           params,
	})
	if err != nil {
		switch {
		case errors.Is(err, model.ErrMaxDepthExceeded):
			return requestError(http.StatusUnprocessableEntity, err)
		case errors.Is(err, sql.ErrNoRows):
			return requestError(http.StatusNotFound, errors.New("parent job not found"))
		}
		return err
	}

	platform.JSON(w, r, http.StatusCreated, DispatchResponse{
		JobID:          job.ID,
		ParentID:       job.ParentID,
		RootID:         job.RootID,
		Depth:          job.Depth,
		RepositoryID:   repository.ID,
		RepositorySlug: repository.Slug,
		Status:         job.Status,
	})
	return nil
}

// ListRepositories returns the repositories the server knows about.
//
// A trigger needs a repository ID, and this is where a cron or a
// webhook receiver finds it without guessing.
func (s *Handlers) ListRepositories(w http.ResponseWriter, r *http.Request) {
	s.respond(w, r, s.listRepositories(w, r))
}

func (s *Handlers) listRepositories(w http.ResponseWriter, r *http.Request) error {
	if _, _, err := s.authenticateUser(r); err != nil {
		return err
	}

	limit, _ := strconv.Atoi(platform.QueryParam(r, "limit"))

	repositories, err := s.repositories.List(r.Context(), limit)
	if err != nil {
		return err
	}

	platform.JSON(w, r, http.StatusOK, repositories)
	return nil
}

// GetJob returns a single job.
func (s *Handlers) GetJob(w http.ResponseWriter, r *http.Request) {
	s.respond(w, r, s.getJob(w, r))
}

func (s *Handlers) getJob(w http.ResponseWriter, r *http.Request) error {
	if _, _, err := s.authenticateUser(r); err != nil {
		return err
	}

	job, err := s.jobs.Get(r.Context(), platform.URLParam(r, "jobID"))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return requestError(http.StatusNotFound, errors.New("job not found"))
		}
		return err
	}

	platform.JSON(w, r, http.StatusOK, job)
	return nil
}

// ListJobs returns jobs, newest first, narrowed by query parameters.
func (s *Handlers) ListJobs(w http.ResponseWriter, r *http.Request) {
	s.respond(w, r, s.listJobs(w, r))
}

func (s *Handlers) listJobs(w http.ResponseWriter, r *http.Request) error {
	if _, _, err := s.authenticateUser(r); err != nil {
		return err
	}

	filter := storage.ListFilter{
		RepositoryID: platform.QueryParam(r, "repository_id"),
		RootID:       platform.QueryParam(r, "root_id"),
		Status:       platform.QueryParam(r, "status"),
	}
	if limit, err := strconv.Atoi(platform.QueryParam(r, "limit")); err == nil {
		filter.Limit = limit
	}
	if filter.Status != "" && !model.ValidJobStatus(filter.Status) {
		return requestError(http.StatusBadRequest, errors.New("unknown job status"))
	}

	jobs, err := s.jobs.List(r.Context(), filter)
	if err != nil {
		return err
	}

	platform.JSON(w, r, http.StatusOK, jobs)
	return nil
}

// ClaimJob leases the oldest pending job the agent can run.
//
// A 204 means the queue holds nothing for this agent right now, which is
// the common case for a polling worker and not an error.
func (s *Handlers) ClaimJob(w http.ResponseWriter, r *http.Request) {
	s.respond(w, r, s.claimJob(w, r))
}

func (s *Handlers) claimJob(w http.ResponseWriter, r *http.Request) error {
	// Only agents take work off the queue. A logged-in human should
	// not be able to lease a job and sit on it.
	if _, err := s.requireAgent(r); err != nil {
		return err
	}

	var req ClaimRequest
	if err := decode(r, &req); err != nil {
		return err
	}
	if strings.TrimSpace(req.AgentID) == "" {
		return requestError(http.StatusBadRequest, errors.New("agent_id is required"))
	}

	job, err := s.jobs.Claim(r.Context(), req.AgentID, req.Labels)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			w.WriteHeader(http.StatusNoContent)
			return nil
		}
		return err
	}

	response := ClaimResponse{Job: job}
	if repository, err := s.repositories.Get(r.Context(), job.RepositoryID); err == nil {
		response.Repository = repository
	}

	platform.JSON(w, r, http.StatusOK, response)
	return nil
}

// JobStatus settles a job into a terminal state.
func (s *Handlers) JobStatus(w http.ResponseWriter, r *http.Request) {
	s.respond(w, r, s.jobStatus(w, r))
}

func (s *Handlers) jobStatus(w http.ResponseWriter, r *http.Request) error {
	// The agent that ran the job reports it; an admin may settle one
	// by hand to cancel it.
	if _, err := s.requireAgent(r); err != nil {
		return err
	}

	var req JobStatusRequest
	if err := decode(r, &req); err != nil {
		return err
	}
	if !model.TerminalJobStatus(req.Status) {
		return requestError(http.StatusBadRequest, errors.New("status must be one of: passed, failed, timeout, cancelled"))
	}

	job, err := s.jobs.Finish(r.Context(), platform.URLParam(r, "jobID"), storage.StatusRequest{
		Status:   req.Status,
		ExitCode: req.ExitCode,
		Error:    req.Error,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Either the job is unknown or it already settled. Both
			// mean "this report changes nothing".
			return requestError(http.StatusConflict, errors.New("job is not open"))
		}
		return err
	}

	platform.JSON(w, r, http.StatusOK, job)
	return nil
}

// JobHeartbeat extends the lease the calling agent holds on a job.
func (s *Handlers) JobHeartbeat(w http.ResponseWriter, r *http.Request) {
	s.respond(w, r, s.jobHeartbeat(w, r))
}

func (s *Handlers) jobHeartbeat(w http.ResponseWriter, r *http.Request) error {
	if _, err := s.requireAgent(r); err != nil {
		return err
	}

	var req ClaimRequest
	if err := decode(r, &req); err != nil {
		return err
	}
	if strings.TrimSpace(req.AgentID) == "" {
		return requestError(http.StatusBadRequest, errors.New("agent_id is required"))
	}

	if err := s.jobs.Heartbeat(r.Context(), platform.URLParam(r, "jobID"), req.AgentID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return requestError(http.StatusConflict, errors.New("job is not leased by this agent"))
		}
		return err
	}

	w.WriteHeader(http.StatusNoContent)
	return nil
}

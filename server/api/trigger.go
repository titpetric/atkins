package api

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/titpetric/platform"

	"github.com/titpetric/atkins/server/model"
	"github.com/titpetric/atkins/server/storage"
)

// TriggerRequest is the body of POST /api/repository/{repositoryID}/trigger.
//
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

	// Branch and Revision pin what gets checked out. Empty lets the
	// agent fall back to the repository's default branch, which is
	// what a "run this nightly" trigger wants.
	Branch   string `json:"branch"`
	Revision string `json:"revision"`

	// Params are handed to the job as ATKINS_JOB_PARAMS. This is how a
	// dispatching job passes a tag or a commit to each child.
	Params map[string]any `json:"params"`

	Labels []string `json:"labels"`

	// ParentID records the job that triggered this one.
	ParentID string `json:"parent_id"`
}

// Trigger queues a job against a known repository.
func (s *Handlers) Trigger(w http.ResponseWriter, r *http.Request) {
	s.respond(w, r, s.trigger(w, r))
}

func (s *Handlers) trigger(w http.ResponseWriter, r *http.Request) error {
	user, _, err := s.authenticateUser(r)
	if err != nil {
		return err
	}

	var req TriggerRequest
	if err := decode(r, &req); err != nil {
		return err
	}

	command := strings.TrimSpace(req.Command)
	if command == "" {
		job := strings.TrimSpace(req.Job)
		if job == "" {
			return requestError(http.StatusBadRequest, errors.New("job or command is required"))
		}
		// A job name is the common case, and turning it into the
		// invocation here means a trigger payload stays a name rather
		// than a shell string somebody has to quote.
		command = "atkins " + job
	}

	repository, err := s.repositories.Get(r.Context(), platform.URLParam(r, "repositoryID"))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return requestError(http.StatusNotFound, errors.New("repository not found"))
		}
		return err
	}

	// The allowlist governs triggers exactly as it governs dispatch: a
	// repository that may not be built may not be built on a schedule
	// either.
	allowed, err := s.allowedRepository(r, repository.Slug)
	if err != nil {
		return err
	}
	if !allowed {
		return requestError(http.StatusForbidden,
			fmt.Errorf("%w: %s", model.ErrRepositoryNotAllowed, repository.Slug))
	}

	params, err := encodeParams(req.Params)
	if err != nil {
		return requestError(http.StatusBadRequest, err)
	}

	branch := req.Branch
	if branch == "" && req.Revision == "" {
		branch = repository.DefaultBranch
	}

	job, err := s.jobs.Create(r.Context(), storage.JobRequest{
		ParentID:         req.ParentID,
		RepositoryID:     repository.ID,
		UserID:           user.ID,
		WorkingDirectory: cleanWorkingDirectory(req.WorkingDirectory),
		Command:          command,
		Branch:           branch,
		Revision:         req.Revision,
		Labels:           req.Labels,
		Params:           params,
	})
	if err != nil {
		return mapJobCreateError(err)
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

// RetryJob queues a copy of a finished job.
//
// A retry is a new job rather than a reset of the old one: the previous
// attempt keeps its output and its outcome, which is the whole point of
// looking at a failure after retrying it.
func (s *Handlers) RetryJob(w http.ResponseWriter, r *http.Request) {
	s.respond(w, r, s.retryJob(w, r))
}

func (s *Handlers) retryJob(w http.ResponseWriter, r *http.Request) error {
	user, _, err := s.authenticateUser(r)
	if err != nil {
		return err
	}

	previous, err := s.jobs.Get(r.Context(), platform.URLParam(r, "jobID"))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return requestError(http.StatusNotFound, errors.New("job not found"))
		}
		return err
	}

	if !previous.IsTerminal() {
		return requestError(http.StatusConflict,
			errors.New("job has not finished; cancel it before retrying"))
	}

	repository, err := s.repositories.Get(r.Context(), previous.RepositoryID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return requestError(http.StatusNotFound, errors.New("repository not found"))
		}
		return err
	}

	allowed, err := s.allowedRepository(r, repository.Slug)
	if err != nil {
		return err
	}
	if !allowed {
		return requestError(http.StatusForbidden,
			fmt.Errorf("%w: %s", model.ErrRepositoryNotAllowed, repository.Slug))
	}

	// The retry keeps the original's parent, so a re-run child stays
	// under the job that dispatched it rather than becoming a root of
	// its own.
	job, err := s.jobs.Create(r.Context(), storage.JobRequest{
		ParentID:         previous.ParentID,
		RepositoryID:     previous.RepositoryID,
		UserID:           user.ID,
		WorkingDirectory: previous.WorkingDirectory,
		Command:          previous.Command,
		Branch:           previous.Branch,
		Revision:         previous.Revision,
		Labels:           splitLabels(previous.Labels),
		Params:           previous.Params,
	})
	if err != nil {
		return mapJobCreateError(err)
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

// CancelJob settles an unfinished job as cancelled.
func (s *Handlers) CancelJob(w http.ResponseWriter, r *http.Request) {
	s.respond(w, r, s.cancelJob(w, r))
}

func (s *Handlers) cancelJob(w http.ResponseWriter, r *http.Request) error {
	if _, _, err := s.authenticateUser(r); err != nil {
		return err
	}

	job, err := s.jobs.Finish(r.Context(), platform.URLParam(r, "jobID"), storage.StatusRequest{
		Status:   model.JobStatusCancelled,
		ExitCode: 1,
		Error:    "cancelled",
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return requestError(http.StatusConflict, errors.New("job is not open"))
		}
		return err
	}

	platform.JSON(w, r, http.StatusOK, job)
	return nil
}

// splitLabels parses the stored comma separated labels column.
func splitLabels(labels string) []string {
	parts := strings.Split(labels, ",")

	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if part = strings.TrimSpace(part); part != "" {
			result = append(result, part)
		}
	}

	if len(result) == 0 {
		return nil
	}
	return result
}

// mapJobCreateError turns a job creation failure into a status.
func mapJobCreateError(err error) error {
	switch {
	case errors.Is(err, model.ErrMaxDepthExceeded):
		return requestError(http.StatusUnprocessableEntity, err)
	case errors.Is(err, sql.ErrNoRows):
		return requestError(http.StatusNotFound, errors.New("parent job not found"))
	}
	return err
}

package api

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/titpetric/platform"
)

// JobLogRequest is the body of POST /api/job/{jobID}/log.
type JobLogRequest struct {
	// Stream is "output" (the command's combined output) or "error"
	// (the agent's own commentary). Anything else is read as output.
	Stream string `json:"stream"`

	Content string `json:"content"`
}

// AppendJobLog records a chunk of output for a job.
func (s *Handlers) AppendJobLog(w http.ResponseWriter, r *http.Request) {
	s.respond(w, r, s.appendJobLog(w, r))
}

func (s *Handlers) appendJobLog(w http.ResponseWriter, r *http.Request) error {
	// Output is written by the agent that ran the job.
	if _, err := s.requireAgent(r); err != nil {
		return err
	}

	var req JobLogRequest
	if err := decode(r, &req); err != nil {
		return err
	}
	if req.Content == "" {
		w.WriteHeader(http.StatusNoContent)
		return nil
	}

	jobID := platform.URLParam(r, "jobID")
	if _, err := s.jobs.Get(r.Context(), jobID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return requestError(http.StatusNotFound, errors.New("job not found"))
		}
		return err
	}

	if err := s.jobLogs.Append(r.Context(), jobID, req.Stream, req.Content); err != nil {
		return err
	}

	w.WriteHeader(http.StatusNoContent)
	return nil
}

// GetJobLog returns the recorded output for a job.
func (s *Handlers) GetJobLog(w http.ResponseWriter, r *http.Request) {
	s.respond(w, r, s.getJobLog(w, r))
}

func (s *Handlers) getJobLog(w http.ResponseWriter, r *http.Request) error {
	user, _, err := s.authenticateUser(r)
	if err != nil {
		return err
	}

	// Output is the sensitive half of a job, so the job is loaded and
	// checked before its lines are handed over — reading a log must not
	// be a way around the scoping on reading a job.
	job, err := s.readableJob(r, user, platform.URLParam(r, "jobID"))
	if err != nil {
		return err
	}

	entries, err := s.jobLogs.List(r.Context(), job.ID)
	if err != nil {
		return err
	}

	platform.JSON(w, r, http.StatusOK, entries)
	return nil
}

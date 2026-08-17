package api

import (
	"database/sql"
	"encoding/base64"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/titpetric/platform"

	"github.com/titpetric/atkins/server/model"
)

// JobInputResponse is what an agent collecting keystrokes gets back.
//
// The bytes are base64 because they are bytes: a terminal's input is
// arrow keys, control characters and whatever else the browser's
// keyboard produced, none of which survives being called a JSON string.
type JobInputResponse struct {
	// Input is base64 of what has been typed since the last collection,
	// empty when nothing has.
	Input string `json:"input,omitempty"`
}

// InputPollTimeout is how long a collection request is held open waiting
// for a keystroke.
//
// It is shorter than any sensible proxy's idle timeout and longer than
// the round trip, which is the whole of what it has to be: the agent
// calls again the moment it returns, so this decides how often an idle
// terminal costs a request rather than how quickly a keystroke arrives.
const InputPollTimeout = 25 * time.Second

// CollectJobInput hands the agent whatever has been typed at a job.
//
// It long-polls: with nothing queued the request waits, and returns
// empty when the wait is up. That is what makes a keystroke reach the
// process in the time it takes to cross the network, without the agent
// asking twenty times a second whether anybody has pressed a key.
//
// Only the agent holding the job may collect, and only while the job is
// running. Input is meant for a process, and the lease is what says
// which agent has one.
func (s *Handlers) CollectJobInput(w http.ResponseWriter, r *http.Request) {
	s.respond(w, r, s.collectJobInput(w, r))
}

func (s *Handlers) collectJobInput(w http.ResponseWriter, r *http.Request) error {
	if _, err := s.requireAgent(r); err != nil {
		return err
	}

	agentID := strings.TrimSpace(platform.QueryParam(r, "agent_id"))
	if agentID == "" {
		return requestError(http.StatusBadRequest, errors.New("agent_id is required"))
	}

	jobID := platform.URLParam(r, "jobID")

	job, err := s.jobs.Get(r.Context(), jobID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return requestError(http.StatusNotFound, errJobNotFound)
		}
		return err
	}

	// A settled job's keystrokes belong to nobody, and an agent that no
	// longer holds the lease is running a command the server has already
	// taken away from it. Both answer 409, the same as a heartbeat, so an
	// agent reads them the same way: stop.
	if job.Status != model.JobStatusRunning || job.AgentID != agentID {
		return requestError(http.StatusConflict, errors.New("job is not leased by this agent"))
	}

	if s.live == nil {
		platform.JSON(w, r, http.StatusOK, JobInputResponse{})
		return nil
	}

	input := s.live.Receive(r.Context(), jobID, InputPollTimeout)

	platform.JSON(w, r, http.StatusOK, JobInputResponse{
		Input: base64.StdEncoding.EncodeToString(input),
	})
	return nil
}

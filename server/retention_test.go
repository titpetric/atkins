package server_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/titpetric/atkins/server"
	"github.com/titpetric/atkins/server/model"
)

// finish settles a job as an agent would.
func finish(t *testing.T, url, agentToken, jobID string, status model.JobStatus) {
	t.Helper()

	code := call(t, http.MethodPost, url+"/api/job/"+jobID+"/status", agentToken, map[string]any{
		"status": status,
	}, nil)
	require.Equal(t, http.StatusOK, code)
}

// logCount reads how many output rows a job still has.
func logCount(t *testing.T, url, token, jobID string) int {
	t.Helper()

	var entries []map[string]any
	status := call(t, http.MethodGet, url+"/api/job/"+jobID+"/log", token, nil, &entries)
	require.Equal(t, http.StatusOK, status)

	return len(entries)
}

func TestRetentionSweepDropsOutputAndKeepsTheJob(t *testing.T) {
	// A sweep on a real ticker: this is the wiring the module owns —
	// the timer, the settings read on every pass, and the storage call.
	url := testServerWith(t, func(opts *server.Options) {
		opts.RetentionInterval = 20 * time.Millisecond
	})

	admin := register(t, url)
	agent := enrol(t, url, "agent-1")

	job := dispatch(t, url, admin.Token, "atkins build", "")

	status := call(t, http.MethodPost, url+"/api/job/"+job.JobID+"/log", agent.Token, map[string]string{
		"content": "hello\n",
	}, nil)
	require.Equal(t, http.StatusNoContent, status)

	// A job still running is never old enough to sweep, whatever the
	// window says.
	setSetting(t, url, admin.Token, model.SettingJobLogRetention, "1ns")

	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, 1, logCount(t, url, admin.Token, job.JobID))

	finish(t, url, agent.Token, job.JobID, model.JobStatusPassed)

	require.Eventually(t, func() bool {
		return logCount(t, url, admin.Token, job.JobID) == 0
	}, 5*time.Second, 20*time.Millisecond, "output was not swept")

	// The outcome outlives the output: that is the whole point of two
	// windows rather than one.
	var stored map[string]any
	status = call(t, http.MethodGet, url+"/api/job/"+job.JobID, admin.Token, nil, &stored)
	require.Equal(t, http.StatusOK, status)
	assert.Equal(t, model.JobStatusPassed, stored["status"])

	// Until job.retention says otherwise.
	setSetting(t, url, admin.Token, model.SettingJobRetention, "1ns")

	require.Eventually(t, func() bool {
		return call(t, http.MethodGet, url+"/api/job/"+job.JobID, admin.Token, nil, nil) == http.StatusNotFound
	}, 5*time.Second, 20*time.Millisecond, "job record was not swept")
}

func TestRetentionKeepsEverythingByDefault(t *testing.T) {
	url := testServerWith(t, func(opts *server.Options) {
		opts.RetentionInterval = 20 * time.Millisecond
	})

	admin := register(t, url)
	agent := enrol(t, url, "agent-1")

	job := dispatch(t, url, admin.Token, "atkins build", "")

	status := call(t, http.MethodPost, url+"/api/job/"+job.JobID+"/log", agent.Token, map[string]string{
		"content": "hello\n",
	}, nil)
	require.Equal(t, http.StatusNoContent, status)

	finish(t, url, agent.Token, job.JobID, model.JobStatusFailed)

	// job.retention defaults to keeping records forever, and
	// job.log_retention to a month. Neither touches a job that finished
	// a moment ago, however often the sweep runs.
	time.Sleep(100 * time.Millisecond)

	assert.Equal(t, 1, logCount(t, url, admin.Token, job.JobID))
	assert.Equal(t, http.StatusOK, call(t, http.MethodGet, url+"/api/job/"+job.JobID, admin.Token, nil, nil))
}

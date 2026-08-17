package server_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/titpetric/atkins/server/model"
)

// recordRun dispatches a job that names the machine already running it,
// which is what `atkins` does for a run happening on a laptop.
func recordRun(t *testing.T, url, token, command, agent string) dispatchResponse {
	t.Helper()

	var response dispatchResponse
	status := call(t, http.MethodPost, url+"/api/dispatch", token, map[string]any{
		"repository": map[string]string{
			"remote_url": "git@github.com:titpetric/atkins.git",
			"ref":        "0123456789abcdef",
		},
		"command": command,
		"agent":   agent,
	}, &response)
	require.Equal(t, http.StatusCreated, status)

	return response
}

// jobLog reads the recorded output of a job.
func jobLog(t *testing.T, url, token, jobID string) []map[string]any {
	t.Helper()

	var entries []map[string]any
	status := call(t, http.MethodGet, url+"/api/job/"+jobID+"/log", token, nil, &entries)
	require.Equal(t, http.StatusOK, status)

	return entries
}

func TestRecordedRunIsRunningAndNeverQueued(t *testing.T) {
	url := testServer(t)
	token := register(t, url)
	agent := enrol(t, url, "agent-1")

	job := recordRun(t, url, token.Token, "atkins test", "laptop")

	// The run is already happening, so the job starts running rather
	// than pending.
	assert.Equal(t, model.JobStatusRunning, job.Status)

	// And it is not work for an agent: claiming finds an empty queue.
	status := call(t, http.MethodPost, url+"/api/job/claim", agent.Token, map[string]any{
		"agent_id": "agent-1",
	}, nil)
	assert.Equal(t, http.StatusNoContent, status)

	var view map[string]any
	status = call(t, http.MethodGet, url+"/api/job/"+job.JobID, token.Token, nil, &view)
	require.Equal(t, http.StatusOK, status)

	// The machine that runs it holds the lease, so a run that dies
	// without reporting is reclaimed the way a lost agent is.
	assert.Equal(t, "laptop", view["agent_id"])
	assert.NotEmpty(t, view["started_at"])
	assert.NotEmpty(t, view["lease_expires_at"])
}

func TestRecordedRunIsReportedByItsOwner(t *testing.T) {
	url := testServer(t)
	token := register(t, url)

	job := recordRun(t, url, token.Token, "atkins test", "laptop")

	// Everything an agent reports about a job, the machine that ran it
	// locally reports with an ordinary account and no agent credentials.
	status := call(t, http.MethodPost, url+"/api/job/"+job.JobID+"/checkout", token.Token, map[string]any{
		"ref":        "main",
		"commit_sha": "0123456789abcdef0123456789abcdef01234567",
	}, nil)
	assert.Equal(t, http.StatusNoContent, status)

	status = call(t, http.MethodPost, url+"/api/job/"+job.JobID+"/heartbeat", token.Token, map[string]any{
		"agent_id": "laptop",
	}, nil)
	assert.Equal(t, http.StatusNoContent, status)

	status = call(t, http.MethodPost, url+"/api/job/"+job.JobID+"/log", token.Token, map[string]any{
		"stream":  "output",
		"content": "go:build ✓\n",
	}, nil)
	assert.Equal(t, http.StatusNoContent, status)

	status = call(t, http.MethodPost, url+"/api/job/"+job.JobID+"/status", token.Token, map[string]any{
		"status":    "failed",
		"exit_code": 2,
	}, nil)
	assert.Equal(t, http.StatusOK, status)

	entries := jobLog(t, url, token.Token, job.JobID)
	require.Len(t, entries, 1)
	assert.Equal(t, "go:build ✓\n", entries[0]["content"])

	var view map[string]any
	status = call(t, http.MethodGet, url+"/api/job/"+job.JobID, token.Token, nil, &view)
	require.Equal(t, http.StatusOK, status)

	// The exit code the shell saw is the exit code the job page shows.
	assert.Equal(t, model.JobStatusFailed, view["status"])
	assert.Equal(t, float64(2), view["exit_code"])
	assert.Equal(t, "0123456789abcdef0123456789abcdef01234567", view["commit_sha"])
}

func TestReportingRefusesAPendingJob(t *testing.T) {
	url := testServer(t)
	token := register(t, url)

	// Nobody has started this one, so a report on it would be fiction.
	// Cancelling is how its owner stops it.
	job := dispatch(t, url, token.Token, "atkins build", "")

	status := call(t, http.MethodPost, url+"/api/job/"+job.JobID+"/log", token.Token, map[string]any{
		"stream":  "output",
		"content": "output from a run that never happened\n",
	}, nil)
	assert.Equal(t, http.StatusForbidden, status)

	status = call(t, http.MethodPost, url+"/api/job/"+job.JobID+"/status", token.Token, map[string]any{
		"status":    "passed",
		"exit_code": 0,
	}, nil)
	assert.Equal(t, http.StatusForbidden, status)
}

func TestReportingRefusesAnotherUsersRun(t *testing.T) {
	url := testServer(t)
	admin := register(t, url)
	dev := registerUser(t, url, admin.Token, "dev@example.com", "dev")

	job := recordRun(t, url, admin.Token, "atkins test", "laptop")

	// Not their job, so it reads as one that isn't there — the same
	// answer reading it gives.
	status := call(t, http.MethodPost, url+"/api/job/"+job.JobID+"/log", dev.Token, map[string]any{
		"stream":  "output",
		"content": "output from somebody else's machine\n",
	}, nil)
	assert.Equal(t, http.StatusNotFound, status)

	status = call(t, http.MethodPost, url+"/api/job/"+job.JobID+"/status", dev.Token, map[string]any{
		"status":    "passed",
		"exit_code": 0,
	}, nil)
	assert.Equal(t, http.StatusNotFound, status)
}

func TestReportingOnAPublicInstanceStillRequiresOwnership(t *testing.T) {
	url := testServer(t)
	admin := register(t, url)
	dev := registerUser(t, url, admin.Token, "dev@example.com", "dev")

	// A public instance lets everyone read every job. Reporting is a
	// write, and ownership is checked outright rather than through
	// readability, or making an instance public would hand every user
	// everybody else's runs to settle.
	setSetting(t, url, admin.Token, model.SettingJobVisibility, model.VisibilityPublic)

	job := recordRun(t, url, admin.Token, "atkins test", "laptop")

	status := call(t, http.MethodGet, url+"/api/job/"+job.JobID, dev.Token, nil, nil)
	require.Equal(t, http.StatusOK, status)

	status = call(t, http.MethodPost, url+"/api/job/"+job.JobID+"/status", dev.Token, map[string]any{
		"status":    "passed",
		"exit_code": 0,
	}, nil)
	assert.Equal(t, http.StatusNotFound, status)
}

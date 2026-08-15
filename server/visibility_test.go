package server_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/titpetric/atkins/server/model"
)

// registerUser opens registration and creates one ordinary account.
func registerUser(t *testing.T, url, adminToken, email, username string) tokenResponse {
	t.Helper()

	setSetting(t, url, adminToken, model.SettingRegistrationOpen, "true")

	var user tokenResponse
	status := call(t, http.MethodPost, url+"/api/user/register", "", map[string]string{
		"email":    email,
		"username": username,
		"password": "correct-horse",
	}, &user)
	require.Equal(t, http.StatusCreated, status)

	return user
}

// jobIDs pulls the ids out of a job listing.
func jobIDs(jobs []map[string]any) []string {
	ids := make([]string, 0, len(jobs))
	for _, job := range jobs {
		id, _ := job["id"].(string)
		ids = append(ids, id)
	}
	return ids
}

// listJobs reads /api/job as one caller.
func listJobs(t *testing.T, url, token string) []map[string]any {
	t.Helper()

	var jobs []map[string]any
	status := call(t, http.MethodGet, url+"/api/job", token, nil, &jobs)
	require.Equal(t, http.StatusOK, status)

	return jobs
}

func TestJobsAreScopedToTheUserWhoDispatchedThem(t *testing.T) {
	url := testServer(t)
	admin := register(t, url)
	dev := registerUser(t, url, admin.Token, "dev@example.com", "dev")

	mine := dispatch(t, url, dev.Token, "atkins mine", "")
	theirs := dispatch(t, url, admin.Token, "atkins theirs", "")

	// A listing shows the caller their own runs and nothing else. Job
	// output routinely contains things nobody meant to publish.
	assert.Equal(t, []string{mine.JobID}, jobIDs(listJobs(t, url, dev.Token)))

	// Someone else's job is reported as missing rather than forbidden:
	// "not yours" and "not here" must look the same, or the endpoint
	// tells a stranger which jobs exist.
	for _, path := range []string{
		"/api/job/" + theirs.JobID,
		"/api/job/" + theirs.JobID + "/log",
	} {
		status := call(t, http.MethodGet, url+path, dev.Token, nil, nil)
		assert.Equal(t, http.StatusNotFound, status, path)
	}

	// Acting on it is refused the same way. Reading and writing are one
	// question here: a job you may not read is one you may not stop.
	status := call(t, http.MethodPost, url+"/api/job/"+theirs.JobID+"/cancel", dev.Token, nil, nil)
	assert.Equal(t, http.StatusNotFound, status)

	status = call(t, http.MethodPost, url+"/api/job/"+theirs.JobID+"/retry", dev.Token, nil, nil)
	assert.Equal(t, http.StatusNotFound, status)

	// An admin sees the instance, which is what the flag is for.
	assert.Len(t, listJobs(t, url, admin.Token), 2)

	status = call(t, http.MethodGet, url+"/api/job/"+mine.JobID, admin.Token, nil, nil)
	assert.Equal(t, http.StatusOK, status)
}

func TestAgentsSeeEveryJob(t *testing.T) {
	url := testServer(t)
	admin := register(t, url)
	dev := registerUser(t, url, admin.Token, "dev@example.com", "dev")

	dispatch(t, url, dev.Token, "atkins mine", "")
	dispatch(t, url, admin.Token, "atkins theirs", "")

	// An agent works the whole queue, so scoping it to "its own" jobs
	// would leave it unable to see the work it is there to run.
	agent := enrol(t, url, "agent-1")
	assert.Len(t, listJobs(t, url, agent.Token), 2)
}

func TestUserSeesTheJobsTheirOwnRunDispatched(t *testing.T) {
	url := testServer(t)
	admin := register(t, url)
	dev := registerUser(t, url, admin.Token, "dev@example.com", "dev")
	other := registerUser(t, url, admin.Token, "other@example.com", "other")

	parent := dispatch(t, url, dev.Token, "atkins analyze", "")

	// A pipeline that clears ATKINS_NO_DISPATCH queues children under
	// the agent's own credentials, so the child's user_id is not the
	// human's. Scoping on user_id alone would hide a fan-out from the
	// person who started it.
	agent := enrol(t, url, "agent-1")
	child := dispatch(t, url, agent.Token, "atkins analyzeTag", parent.JobID)
	require.Equal(t, parent.RootID, child.RootID)

	status := call(t, http.MethodGet, url+"/api/job/"+child.JobID, dev.Token, nil, nil)
	assert.Equal(t, http.StatusOK, status)

	assert.ElementsMatch(t, []string{parent.JobID, child.JobID}, jobIDs(listJobs(t, url, dev.Token)))

	// The tree is not visible to somebody outside it.
	status = call(t, http.MethodGet, url+"/api/job/"+child.JobID, other.Token, nil, nil)
	assert.Equal(t, http.StatusNotFound, status)

	assert.Empty(t, listJobs(t, url, other.Token))
}

func TestPublicVisibilityRestoresTheSharedView(t *testing.T) {
	url := testServer(t)
	admin := register(t, url)
	dev := registerUser(t, url, admin.Token, "dev@example.com", "dev")

	theirs := dispatch(t, url, admin.Token, "atkins theirs", "")
	dispatch(t, url, dev.Token, "atkins mine", "")

	// One team on one instance can have the shared view back, and it is
	// a decision an admin makes rather than the state a server starts
	// in.
	setSetting(t, url, admin.Token, model.SettingJobVisibility, model.VisibilityPublic)

	assert.Len(t, listJobs(t, url, dev.Token), 2)

	status := call(t, http.MethodGet, url+"/api/job/"+theirs.JobID, dev.Token, nil, nil)
	assert.Equal(t, http.StatusOK, status)

	// And it goes back without a restart, because it is a setting.
	status = call(t, http.MethodDelete, url+"/api/admin/setting/"+model.SettingJobVisibility, admin.Token, nil, nil)
	require.Equal(t, http.StatusOK, status)

	status = call(t, http.MethodGet, url+"/api/job/"+theirs.JobID, dev.Token, nil, nil)
	assert.Equal(t, http.StatusNotFound, status)
}

func TestJobLogIsScopedLikeTheJob(t *testing.T) {
	url := testServer(t)
	admin := register(t, url)
	dev := registerUser(t, url, admin.Token, "dev@example.com", "dev")
	agent := enrol(t, url, "agent-1")

	job := dispatch(t, url, admin.Token, "atkins build", "")

	status := call(t, http.MethodPost, url+"/api/job/"+job.JobID+"/log", agent.Token, map[string]string{
		"content": "AWS_SECRET_ACCESS_KEY=hunter2\n",
	}, nil)
	require.Equal(t, http.StatusNoContent, status)

	status = call(t, http.MethodGet, url+"/api/job/"+job.JobID+"/log", dev.Token, nil, nil)
	require.Equal(t, http.StatusNotFound, status)

	var entries []map[string]any
	status = call(t, http.MethodGet, url+"/api/job/"+job.JobID+"/log", admin.Token, nil, &entries)
	require.Equal(t, http.StatusOK, status)
	assert.Len(t, entries, 1)
}

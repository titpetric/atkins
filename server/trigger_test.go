package server_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/titpetric/atkins/server/model"
)

func TestTriggerQueuesAJobByName(t *testing.T) {
	url := testServer(t)
	admin := register(t, url)

	// A repository is known once something has been dispatched for it.
	seed := dispatch(t, url, admin.Token, "atkins", "")

	var triggered dispatchResponse
	status := call(t, http.MethodPost, url+"/api/repository/"+seed.RepositoryID+"/trigger", admin.Token, map[string]any{
		"job":    "analyze",
		"params": map[string]any{"tag": "v1.2.3"},
	}, &triggered)
	require.Equal(t, http.StatusCreated, status)
	assert.NotEmpty(t, triggered.JobID)
	assert.Equal(t, seed.RepositoryID, triggered.RepositoryID)

	var job map[string]any
	status = call(t, http.MethodGet, url+"/api/job/"+triggered.JobID, admin.Token, nil, &job)
	require.Equal(t, http.StatusOK, status)

	// A job name becomes the invocation, so a trigger payload stays a
	// name rather than a shell string.
	assert.Equal(t, "atkins analyze", job["command"])
	assert.JSONEq(t, `{"tag":"v1.2.3"}`, job["params"].(string))
	assert.Equal(t, model.JobStatusPending, job["status"])
}

func TestTriggerAcceptsAnExplicitCommand(t *testing.T) {
	url := testServer(t)
	admin := register(t, url)
	seed := dispatch(t, url, admin.Token, "atkins", "")

	var triggered dispatchResponse
	status := call(t, http.MethodPost, url+"/api/repository/"+seed.RepositoryID+"/trigger", admin.Token, map[string]any{
		"command":           "atkins -f ci/nightly.yml build",
		"working_directory": "server",
		"revision":          "0123456789abcdef",
	}, &triggered)
	require.Equal(t, http.StatusCreated, status)

	var job map[string]any
	status = call(t, http.MethodGet, url+"/api/job/"+triggered.JobID, admin.Token, nil, &job)
	require.Equal(t, http.StatusOK, status)
	assert.Equal(t, "atkins -f ci/nightly.yml build", job["command"])
	assert.Equal(t, "server", job["working_directory"])
	assert.Equal(t, "0123456789abcdef", job["revision"])
}

func TestTriggerNestsUnderADispatchingJob(t *testing.T) {
	url := testServer(t)
	admin := register(t, url)

	parent := dispatch(t, url, admin.Token, "atkins analyze", "")

	// This is the fan-out the issue describes: one job queues a child
	// per tag, each with its own parameters.
	for _, tag := range []string{"v1.0.0", "v1.1.0"} {
		var child dispatchResponse
		status := call(t, http.MethodPost, url+"/api/repository/"+parent.RepositoryID+"/trigger", admin.Token, map[string]any{
			"job":       "analyzeTag",
			"parent_id": parent.JobID,
			"params":    map[string]any{"tag": tag},
		}, &child)
		require.Equal(t, http.StatusCreated, status)
		assert.Equal(t, parent.JobID, child.ParentID)
		assert.Equal(t, parent.RootID, child.RootID)
		assert.Equal(t, int64(1), child.Depth)
	}

	var jobs []map[string]any
	status := call(t, http.MethodGet, url+"/api/job?root_id="+parent.RootID, admin.Token, nil, &jobs)
	require.Equal(t, http.StatusOK, status)
	assert.Len(t, jobs, 3)
}

func TestTriggerRequiresAJobOrCommand(t *testing.T) {
	url := testServer(t)
	admin := register(t, url)
	seed := dispatch(t, url, admin.Token, "atkins", "")

	status := call(t, http.MethodPost, url+"/api/repository/"+seed.RepositoryID+"/trigger", admin.Token, map[string]any{}, nil)
	assert.Equal(t, http.StatusBadRequest, status)
}

func TestTriggerRejectsAnUnknownRepository(t *testing.T) {
	url := testServer(t)
	admin := register(t, url)

	status := call(t, http.MethodPost, url+"/api/repository/01ARZ3NDEKTSV4RRFFQ69G5FAV/trigger", admin.Token, map[string]any{
		"job": "analyze",
	}, nil)
	assert.Equal(t, http.StatusNotFound, status)
}

func TestTriggerRespectsTheAllowlist(t *testing.T) {
	url := testServer(t)
	admin := register(t, url)
	seed := dispatch(t, url, admin.Token, "atkins", "")

	setSetting(t, url, admin.Token, model.SettingRepositoryPolicy, model.PolicyAllowlist)

	// A repository that may not be built may not be built on a
	// schedule either.
	status := call(t, http.MethodPost, url+"/api/repository/"+seed.RepositoryID+"/trigger", admin.Token, map[string]any{
		"job": "analyze",
	}, nil)
	assert.Equal(t, http.StatusForbidden, status)

	allowRepository(t, url, admin.Token, "github.com/titpetric/*")

	status = call(t, http.MethodPost, url+"/api/repository/"+seed.RepositoryID+"/trigger", admin.Token, map[string]any{
		"job": "analyze",
	}, nil)
	assert.Equal(t, http.StatusCreated, status)
}

func TestTriggerRequiresAuthentication(t *testing.T) {
	url := testServer(t)
	admin := register(t, url)
	seed := dispatch(t, url, admin.Token, "atkins", "")

	status := call(t, http.MethodPost, url+"/api/repository/"+seed.RepositoryID+"/trigger", "", map[string]any{
		"job": "analyze",
	}, nil)
	assert.Equal(t, http.StatusUnauthorized, status)
}

func TestListRepositories(t *testing.T) {
	url := testServer(t)
	admin := register(t, url)
	seed := dispatch(t, url, admin.Token, "atkins", "")

	var repositories []map[string]any
	status := call(t, http.MethodGet, url+"/api/repository", admin.Token, nil, &repositories)
	require.Equal(t, http.StatusOK, status)
	require.Len(t, repositories, 1)
	assert.Equal(t, seed.RepositoryID, repositories[0]["id"])
	assert.Equal(t, "github.com/titpetric/atkins", repositories[0]["slug"])
}

func TestRetryCopiesAFinishedJob(t *testing.T) {
	url := testServer(t)
	admin := register(t, url)
	agent := enrol(t, url, "agent-1")

	original := dispatch(t, url, admin.Token, "atkins build", "")

	status := call(t, http.MethodPost, url+"/api/job/"+original.JobID+"/status", agent.Token, map[string]any{
		"status":    model.JobStatusFailed,
		"exit_code": 2,
	}, nil)
	require.Equal(t, http.StatusOK, status)

	var retried dispatchResponse
	status = call(t, http.MethodPost, url+"/api/job/"+original.JobID+"/retry", admin.Token, nil, &retried)
	require.Equal(t, http.StatusCreated, status)
	assert.NotEqual(t, original.JobID, retried.JobID)
	assert.Equal(t, model.JobStatusPending, retried.Status)

	var copied map[string]any
	status = call(t, http.MethodGet, url+"/api/job/"+retried.JobID, admin.Token, nil, &copied)
	require.Equal(t, http.StatusOK, status)
	assert.Equal(t, "atkins build", copied["command"])
	assert.Equal(t, "docs", copied["working_directory"])

	// The original keeps its outcome: retrying is not forgetting.
	var previous map[string]any
	status = call(t, http.MethodGet, url+"/api/job/"+original.JobID, admin.Token, nil, &previous)
	require.Equal(t, http.StatusOK, status)
	assert.Equal(t, model.JobStatusFailed, previous["status"])
	assert.Equal(t, float64(2), previous["exit_code"])
}

func TestRetryRefusesAnUnfinishedJob(t *testing.T) {
	url := testServer(t)
	admin := register(t, url)

	job := dispatch(t, url, admin.Token, "atkins build", "")

	status := call(t, http.MethodPost, url+"/api/job/"+job.JobID+"/retry", admin.Token, nil, nil)
	assert.Equal(t, http.StatusConflict, status)
}

func TestRetryKeepsTheParent(t *testing.T) {
	url := testServer(t)
	admin := register(t, url)
	agent := enrol(t, url, "agent-1")

	parent := dispatch(t, url, admin.Token, "atkins analyze", "")
	child := dispatch(t, url, admin.Token, "atkins analyzeTag", parent.JobID)

	status := call(t, http.MethodPost, url+"/api/job/"+child.JobID+"/status", agent.Token, map[string]any{
		"status": model.JobStatusTimeout,
	}, nil)
	require.Equal(t, http.StatusOK, status)

	var retried dispatchResponse
	status = call(t, http.MethodPost, url+"/api/job/"+child.JobID+"/retry", admin.Token, nil, &retried)
	require.Equal(t, http.StatusCreated, status)

	// A re-run child stays under the job that dispatched it.
	assert.Equal(t, parent.JobID, retried.ParentID)
	assert.Equal(t, parent.RootID, retried.RootID)
}

func TestCancelSettlesAPendingJob(t *testing.T) {
	url := testServer(t)
	admin := register(t, url)

	job := dispatch(t, url, admin.Token, "atkins build", "")

	var cancelled map[string]any
	status := call(t, http.MethodPost, url+"/api/job/"+job.JobID+"/cancel", admin.Token, nil, &cancelled)
	require.Equal(t, http.StatusOK, status)
	assert.Equal(t, model.JobStatusCancelled, cancelled["status"])

	// A cancelled job is off the queue.
	agent := enrol(t, url, "agent-1")
	status = call(t, http.MethodPost, url+"/api/job/claim", agent.Token, map[string]any{
		"agent_id": "agent-1",
	}, nil)
	assert.Equal(t, http.StatusNoContent, status)

	// And cancelling twice changes nothing.
	status = call(t, http.MethodPost, url+"/api/job/"+job.JobID+"/cancel", admin.Token, nil, nil)
	assert.Equal(t, http.StatusConflict, status)
}

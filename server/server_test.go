package server_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"

	_ "github.com/titpetric/platform/pkg/drivers"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/titpetric/platform"

	"github.com/titpetric/atkins/server"
	"github.com/titpetric/atkins/server/model"
)

// connectionSeq hands each test a unique database connection name.
var connectionSeq atomic.Int64

// testAgentToken enrols agents in the test instances.
const testAgentToken = "test-agent-token"

// testServer starts a platform instance with the atkins module mounted
// against a throwaway sqlite database.
//
// The database is a file rather than :memory: because the platform gives
// sqlite a 10 connection pool, and every connection to an in-memory
// sqlite database is a separate, empty database.
func testServer(t *testing.T) string {
	return testServerWith(t, nil)
}

// testServerWith starts an instance whose options a test can adjust,
// for the cases that need a background sweep or a different root.
func testServerWith(t *testing.T, configure func(*server.Options)) string {
	t.Helper()

	// The platform caches connection pools by name for the life of the
	// process, so each test claims a name of its own and gets a fresh
	// database instead of the previous test's fixtures.
	connection := "atkins_test_" + strconv.Itoa(int(connectionSeq.Add(1)))
	dsn := "sqlite://file:" + filepath.Join(t.TempDir(), "atkins.db")
	t.Setenv("PLATFORM_DB_"+strings.ToUpper(connection), dsn)
	platform.SetupConnections(os.Environ())

	opts := server.NewOptions()
	opts.Connection = connection
	opts.SigningKey = "test-signing-key"
	opts.AgentToken = testAgentToken
	// Artefact bytes go somewhere the test owns, rather than into an
	// `artefacts` directory beside the package source.
	opts.ArtefactDir = filepath.Join(t.TempDir(), "artefacts")
	// Off by default: the bootstrap path (no users yet) is what the
	// first register call in each test exercises.
	opts.AllowRegistration = false
	opts.ReclaimInterval = 0
	opts.RetentionInterval = 0

	if configure != nil {
		configure(opts)
	}

	svc := platform.New(platform.NewTestOptions())
	svc.Register(server.NewModule(opts))

	require.NoError(t, svc.Start(context.Background()))
	t.Cleanup(svc.Stop)

	return svc.URL()
}

// call issues a JSON request and decodes the response into target.
func call(t *testing.T, method, url, token string, payload, target any) int {
	t.Helper()

	var body io.Reader
	if payload != nil {
		encoded, err := json.Marshal(payload)
		require.NoError(t, err)
		body = bytes.NewReader(encoded)
	}

	request, err := http.NewRequest(method, url, body)
	require.NoError(t, err)
	request.Header.Set("Content-Type", "application/json")
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}

	response, err := http.DefaultClient.Do(request)
	require.NoError(t, err)
	defer response.Body.Close()

	if target != nil && response.StatusCode != http.StatusNoContent {
		require.NoError(t, json.NewDecoder(response.Body).Decode(target))
	} else {
		_, _ = io.Copy(io.Discard, response.Body)
	}

	return response.StatusCode
}

// tokenResponse mirrors api.TokenResponse.
type tokenResponse struct {
	UserID       string `json:"user_id"`
	Username     string `json:"username"`
	Token        string `json:"token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresAt    int64  `json:"expires_at"`
}

// claimResponse mirrors api.ClaimResponse.
type claimResponse struct {
	Job        *model.Job        `json:"job"`
	Repository *model.Repository `json:"repository"`
}

// dispatchResponse mirrors api.DispatchResponse.
type dispatchResponse struct {
	JobID          string `json:"job_id"`
	ParentID       string `json:"parent_id"`
	RootID         string `json:"root_id"`
	Depth          int64  `json:"depth"`
	RepositoryID   string `json:"repository_id"`
	RepositorySlug string `json:"repository_slug"`
	Status         string `json:"status"`
	ViewToken      string `json:"view_token"`
}

// register creates the bootstrap user and returns its tokens.
func register(t *testing.T, url string) tokenResponse {
	t.Helper()

	var token tokenResponse
	status := call(t, http.MethodPost, url+"/api/user/register", "", map[string]string{
		"email":    "ci@example.com",
		"username": "ci",
		"password": "correct-horse",
	}, &token)
	require.Equal(t, http.StatusCreated, status)
	require.NotEmpty(t, token.Token)
	require.NotEmpty(t, token.RefreshToken)

	return token
}

// dispatch records a job and returns the response.
func dispatch(t *testing.T, url, token, command, parentID string) dispatchResponse {
	t.Helper()

	var response dispatchResponse
	status := call(t, http.MethodPost, url+"/api/dispatch", token, map[string]any{
		"repository": map[string]string{
			"remote_url": "git@github.com:titpetric/atkins.git",
			"ref":        "0123456789abcdef",
		},
		"working_directory": "docs",
		"command":           command,
		"parent_id":         parentID,
	}, &response)
	require.Equal(t, http.StatusCreated, status)

	return response
}

func TestRegisterBootstrapsFirstUserOnly(t *testing.T) {
	url := testServer(t)

	register(t, url)

	// Registration is closed once a user exists.
	status := call(t, http.MethodPost, url+"/api/user/register", "", map[string]string{
		"email":    "second@example.com",
		"username": "second",
		"password": "correct-horse",
	}, nil)
	assert.Equal(t, http.StatusForbidden, status)
}

func TestRegisterRejectsShortPassword(t *testing.T) {
	url := testServer(t)

	status := call(t, http.MethodPost, url+"/api/user/register", "", map[string]string{
		"email":    "ci@example.com",
		"username": "ci",
		"password": "short",
	}, nil)
	assert.Equal(t, http.StatusBadRequest, status)
}

func TestLogin(t *testing.T) {
	url := testServer(t)
	register(t, url)

	t.Run("wrong password", func(t *testing.T) {
		status := call(t, http.MethodPost, url+"/api/user/login", "", map[string]string{
			"email":    "ci@example.com",
			"password": "wrong",
		}, nil)
		assert.Equal(t, http.StatusUnauthorized, status)
	})

	t.Run("correct password", func(t *testing.T) {
		var token tokenResponse
		status := call(t, http.MethodPost, url+"/api/user/login", "", map[string]string{
			"email":    "ci@example.com",
			"password": "correct-horse",
			"hostname": "workstation",
		}, &token)
		require.Equal(t, http.StatusOK, status)
		assert.Equal(t, "ci", token.Username)
		assert.NotEmpty(t, token.Token)

		var whoami map[string]any
		status = call(t, http.MethodGet, url+"/api/user/whoami", token.Token, nil, &whoami)
		require.Equal(t, http.StatusOK, status)
		assert.Equal(t, "ci@example.com", whoami["email"])
		assert.Equal(t, true, whoami["is_admin"])
	})
}

func TestRefreshTokenRotates(t *testing.T) {
	url := testServer(t)
	token := register(t, url)

	var refreshed tokenResponse
	status := call(t, http.MethodPost, url+"/api/user/refreshToken", "", map[string]string{
		"refresh_token": token.RefreshToken,
	}, &refreshed)
	require.Equal(t, http.StatusOK, status)
	assert.NotEqual(t, token.RefreshToken, refreshed.RefreshToken)

	// The spent refresh token no longer works.
	status = call(t, http.MethodPost, url+"/api/user/refreshToken", "", map[string]string{
		"refresh_token": token.RefreshToken,
	}, nil)
	assert.Equal(t, http.StatusUnauthorized, status)
}

func TestLogoutRevokesAccessToken(t *testing.T) {
	url := testServer(t)
	token := register(t, url)

	status := call(t, http.MethodPost, url+"/api/user/logout", token.Token, map[string]string{}, nil)
	require.Equal(t, http.StatusNoContent, status)

	// The access token has not expired, but its session is revoked.
	status = call(t, http.MethodGet, url+"/api/user/whoami", token.Token, nil, nil)
	assert.Equal(t, http.StatusUnauthorized, status)
}

func TestDispatchRecordsJob(t *testing.T) {
	url := testServer(t)
	token := register(t, url)

	job := dispatch(t, url, token.Token, "atkins test:build", "")
	assert.NotEmpty(t, job.JobID)
	assert.Equal(t, job.JobID, job.RootID)
	assert.Equal(t, int64(0), job.Depth)
	assert.Equal(t, "github.com/titpetric/atkins", job.RepositorySlug)
	assert.Equal(t, model.JobStatusPending, job.Status)

	var stored map[string]any
	status := call(t, http.MethodGet, url+"/api/job/"+job.JobID, token.Token, nil, &stored)
	require.Equal(t, http.StatusOK, status)
	assert.Equal(t, "docs", stored["working_directory"])
	assert.Equal(t, "atkins test:build", stored["command"])
}

func TestDispatchDeduplicatesRepository(t *testing.T) {
	url := testServer(t)
	token := register(t, url)

	first := dispatch(t, url, token.Token, "atkins", "")

	// The same repository over https rather than ssh is one repository.
	var second dispatchResponse
	status := call(t, http.MethodPost, url+"/api/dispatch", token.Token, map[string]any{
		"repository": map[string]string{
			"remote_url": "https://github.com/titpetric/atkins",
		},
		"command": "atkins",
	}, &second)
	require.Equal(t, http.StatusCreated, status)

	assert.Equal(t, first.RepositoryID, second.RepositoryID)
}

func TestDispatchNestsChildJobs(t *testing.T) {
	url := testServer(t)
	token := register(t, url)

	parent := dispatch(t, url, token.Token, "atkins analyze", "")
	child := dispatch(t, url, token.Token, "atkins analyzeTag", parent.JobID)

	assert.Equal(t, parent.JobID, child.ParentID)
	assert.Equal(t, parent.RootID, child.RootID)
	assert.Equal(t, int64(1), child.Depth)
}

func TestDispatchStopsRunawayNesting(t *testing.T) {
	url := testServer(t)
	token := register(t, url)

	// MaxJobDepth defaults to 3, so the fourth generation is refused.
	parent := dispatch(t, url, token.Token, "atkins", "")
	for range 3 {
		parent = dispatch(t, url, token.Token, "atkins", parent.JobID)
	}

	status := call(t, http.MethodPost, url+"/api/dispatch", token.Token, map[string]any{
		"repository": map[string]string{"remote_url": "git@github.com:titpetric/atkins.git"},
		"command":    "atkins",
		"parent_id":  parent.JobID,
	}, nil)
	assert.Equal(t, http.StatusUnprocessableEntity, status)
}

func TestDispatchRequiresAuthentication(t *testing.T) {
	url := testServer(t)

	status := call(t, http.MethodPost, url+"/api/dispatch", "", map[string]any{
		"repository": map[string]string{"remote_url": "git@github.com:titpetric/atkins.git"},
		"command":    "atkins",
	}, nil)
	assert.Equal(t, http.StatusUnauthorized, status)
}

func TestJobStatusSettlesOnce(t *testing.T) {
	url := testServer(t)
	token := register(t, url)

	job := dispatch(t, url, token.Token, "atkins", "")

	var settled map[string]any
	status := call(t, http.MethodPost, url+"/api/job/"+job.JobID+"/status", token.Token, map[string]any{
		"status":    model.JobStatusFailed,
		"exit_code": 2,
		"error":     "step failed",
	}, &settled)
	require.Equal(t, http.StatusOK, status)
	assert.Equal(t, model.JobStatusFailed, settled["status"])
	assert.Equal(t, float64(2), settled["exit_code"])

	// A second report cannot overwrite the recorded outcome.
	status = call(t, http.MethodPost, url+"/api/job/"+job.JobID+"/status", token.Token, map[string]any{
		"status": model.JobStatusPassed,
	}, nil)
	assert.Equal(t, http.StatusConflict, status)
}

func TestJobStatusRejectsNonTerminalStatus(t *testing.T) {
	url := testServer(t)
	token := register(t, url)

	job := dispatch(t, url, token.Token, "atkins", "")

	status := call(t, http.MethodPost, url+"/api/job/"+job.JobID+"/status", token.Token, map[string]any{
		"status": model.JobStatusRunning,
	}, nil)
	assert.Equal(t, http.StatusBadRequest, status)
}

func TestClaimLeasesOnePendingJob(t *testing.T) {
	url := testServer(t)
	token := register(t, url)

	job := dispatch(t, url, token.Token, "atkins build", "")

	var claimed claimResponse
	status := call(t, http.MethodPost, url+"/api/job/claim", token.Token, map[string]any{
		"agent_id": "agent-1",
	}, &claimed)
	require.Equal(t, http.StatusOK, status)
	require.NotNil(t, claimed.Job)
	assert.Equal(t, job.JobID, claimed.Job.ID)
	assert.Equal(t, model.JobStatusRunning, claimed.Job.Status)
	assert.Equal(t, "agent-1", claimed.Job.AgentID)

	// The repository travels with the job so the agent can clone it.
	require.NotNil(t, claimed.Repository)
	assert.Equal(t, "github.com/titpetric/atkins", claimed.Repository.Slug)

	// The queue is now empty for the next agent.
	status = call(t, http.MethodPost, url+"/api/job/claim", token.Token, map[string]any{
		"agent_id": "agent-2",
	}, nil)
	assert.Equal(t, http.StatusNoContent, status)
}

func TestJobCheckoutRecordsTheResolvedCommit(t *testing.T) {
	url := testServer(t)
	token := register(t, url)

	var triggered dispatchResponse
	seed := dispatch(t, url, token.Token, "atkins", "")
	status := call(t, http.MethodPost, url+"/api/repository/"+seed.RepositoryID+"/trigger", token.Token, map[string]any{
		"job": "build",
		"ref": "v1.2.3",
	}, &triggered)
	require.Equal(t, http.StatusCreated, status)

	// A checkout can only be reported for a job an agent is running.
	status = call(t, http.MethodPost, url+"/api/job/"+triggered.JobID+"/checkout", token.Token, map[string]any{
		"ref":        "v1.2.3",
		"commit_sha": "0123456789abcdef0123456789abcdef01234567",
	}, nil)
	assert.Equal(t, http.StatusConflict, status)

	// The seed dispatch is ahead of it in the queue, so the agent takes
	// two jobs to reach the triggered one.
	var claimed claimResponse
	for range 2 {
		status = call(t, http.MethodPost, url+"/api/job/claim", token.Token, map[string]any{
			"agent_id": "agent-1",
		}, &claimed)
		require.Equal(t, http.StatusOK, status)
	}
	require.NotNil(t, claimed.Job)
	require.Equal(t, triggered.JobID, claimed.Job.ID)

	status = call(t, http.MethodPost, url+"/api/job/"+triggered.JobID+"/checkout", token.Token, map[string]any{
		"ref":        "v1.2.3",
		"commit_sha": "0123456789abcdef0123456789abcdef01234567",
	}, nil)
	require.Equal(t, http.StatusNoContent, status)

	var job map[string]any
	status = call(t, http.MethodGet, url+"/api/job/"+triggered.JobID, token.Token, nil, &job)
	require.Equal(t, http.StatusOK, status)

	// The tag says what was asked for; the sha says what ran, and only
	// one of the two survives the tag being moved.
	assert.Equal(t, "v1.2.3", job["ref"])
	assert.Equal(t, "0123456789abcdef0123456789abcdef01234567", job["commit_sha"])
}

func TestJobCheckoutRequiresACommit(t *testing.T) {
	url := testServer(t)
	token := register(t, url)

	job := dispatch(t, url, token.Token, "atkins build", "")

	status := call(t, http.MethodPost, url+"/api/job/"+job.JobID+"/checkout", token.Token, map[string]any{
		"ref": "main",
	}, nil)
	assert.Equal(t, http.StatusBadRequest, status)
}

func TestClaimRespectsLabels(t *testing.T) {
	url := testServer(t)
	token := register(t, url)

	var labelled dispatchResponse
	status := call(t, http.MethodPost, url+"/api/dispatch", token.Token, map[string]any{
		"repository": map[string]string{"remote_url": "git@github.com:titpetric/atkins.git"},
		"command":    "atkins build",
		"labels":     []string{"linux", "arm64"},
	}, &labelled)
	require.Equal(t, http.StatusCreated, status)

	// An agent missing one of the labels does not get the job.
	status = call(t, http.MethodPost, url+"/api/job/claim", token.Token, map[string]any{
		"agent_id": "amd64-agent",
		"labels":   []string{"linux", "amd64"},
	}, nil)
	assert.Equal(t, http.StatusNoContent, status)

	var claimed claimResponse
	status = call(t, http.MethodPost, url+"/api/job/claim", token.Token, map[string]any{
		"agent_id": "arm64-agent",
		"labels":   []string{"linux", "arm64", "docker"},
	}, &claimed)
	require.Equal(t, http.StatusOK, status)
	require.NotNil(t, claimed.Job)
	assert.Equal(t, labelled.JobID, claimed.Job.ID)
}

func TestHeartbeatRequiresTheLeaseHolder(t *testing.T) {
	url := testServer(t)
	token := register(t, url)

	job := dispatch(t, url, token.Token, "atkins build", "")

	status := call(t, http.MethodPost, url+"/api/job/claim", token.Token, map[string]any{
		"agent_id": "agent-1",
	}, nil)
	require.Equal(t, http.StatusOK, status)

	status = call(t, http.MethodPost, url+"/api/job/"+job.JobID+"/heartbeat", token.Token, map[string]any{
		"agent_id": "agent-1",
	}, nil)
	assert.Equal(t, http.StatusNoContent, status)

	status = call(t, http.MethodPost, url+"/api/job/"+job.JobID+"/heartbeat", token.Token, map[string]any{
		"agent_id": "impostor",
	}, nil)
	assert.Equal(t, http.StatusConflict, status)
}

func TestListJobsFiltersByRoot(t *testing.T) {
	url := testServer(t)
	token := register(t, url)

	parent := dispatch(t, url, token.Token, "atkins analyze", "")
	dispatch(t, url, token.Token, "atkins analyzeTag", parent.JobID)
	dispatch(t, url, token.Token, "atkins unrelated", "")

	var jobs []map[string]any
	status := call(t, http.MethodGet, url+"/api/job?root_id="+parent.RootID, token.Token, nil, &jobs)
	require.Equal(t, http.StatusOK, status)
	assert.Len(t, jobs, 2)
}

func TestModuleRequiresSigningKey(t *testing.T) {
	svc := platform.New(platform.NewTestOptions())
	svc.Register(server.NewModule(&server.Options{}))

	assert.Error(t, svc.Start(context.Background()))
}

package server_test

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"

	"github.com/titpetric/atkins/server/model"
)

// enrol trades the test agent token for agent credentials.
func enrol(t *testing.T, url, agentID string) tokenResponse {
	t.Helper()

	var token tokenResponse
	status := call(t, http.MethodPost, url+"/api/agent/enrol", "", map[string]any{
		"token":    testAgentToken,
		"agent_id": agentID,
	}, &token)
	require.Equal(t, http.StatusOK, status)
	require.NotEmpty(t, token.Token)

	return token
}

// setSetting overrides one server setting as an admin.
func setSetting(t *testing.T, url, token, name, value string) {
	t.Helper()

	status := call(t, http.MethodPost, url+"/api/admin/setting/"+name, token, map[string]string{
		"value": value,
	}, nil)
	require.Equal(t, http.StatusOK, status)
}

// allowRepository adds an allowlist rule.
func allowRepository(t *testing.T, url, token, pattern string) map[string]any {
	t.Helper()

	var rule map[string]any
	status := call(t, http.MethodPost, url+"/api/admin/repository", token, map[string]string{
		"pattern": pattern,
	}, &rule)
	require.Equal(t, http.StatusCreated, status)

	return rule
}

func TestEnrolRequiresTheSharedToken(t *testing.T) {
	url := testServer(t)

	status := call(t, http.MethodPost, url+"/api/agent/enrol", "", map[string]any{
		"token":    "wrong",
		"agent_id": "agent-1",
	}, nil)
	assert.Equal(t, http.StatusUnauthorized, status)

	status = call(t, http.MethodPost, url+"/api/agent/enrol", "", map[string]any{
		"token": testAgentToken,
	}, nil)
	assert.Equal(t, http.StatusBadRequest, status)
}

func TestEnrolIsRepeatable(t *testing.T) {
	url := testServer(t)

	first := enrol(t, url, "agent-1")
	second := enrol(t, url, "agent-1")

	// A restarted agent keeps its identity and its job history.
	assert.Equal(t, first.UserID, second.UserID)
	assert.NotEqual(t, first.RefreshToken, second.RefreshToken)
}

func TestEnrolledAgentIsNotAnAdmin(t *testing.T) {
	url := testServer(t)

	// Enrol before any human registers: the agent must not take the
	// bootstrap admin slot.
	agent := enrol(t, url, "agent-1")

	var whoami map[string]any
	status := call(t, http.MethodGet, url+"/api/user/whoami", agent.Token, nil, &whoami)
	require.Equal(t, http.StatusOK, status)
	assert.Equal(t, false, whoami["is_admin"])

	status = call(t, http.MethodGet, url+"/api/admin/user", agent.Token, nil, nil)
	assert.Equal(t, http.StatusForbidden, status)

	// And the human who registers afterwards still gets admin.
	human := register(t, url)
	status = call(t, http.MethodGet, url+"/api/user/whoami", human.Token, nil, &whoami)
	require.Equal(t, http.StatusOK, status)
	assert.Equal(t, true, whoami["is_admin"])
}

func TestOnlyAgentsClaimJobs(t *testing.T) {
	url := testServer(t)
	admin := register(t, url)

	// A second, ordinary human cannot take work off the queue.
	setSetting(t, url, admin.Token, model.SettingRegistrationOpen, "true")

	var human tokenResponse
	status := call(t, http.MethodPost, url+"/api/user/register", "", map[string]string{
		"email":    "dev@example.com",
		"username": "dev",
		"password": "correct-horse",
	}, &human)
	require.Equal(t, http.StatusCreated, status)

	dispatch(t, url, admin.Token, "atkins build", "")

	status = call(t, http.MethodPost, url+"/api/job/claim", human.Token, map[string]any{
		"agent_id": "impostor",
	}, nil)
	assert.Equal(t, http.StatusForbidden, status)

	agent := enrol(t, url, "agent-1")
	var claimed claimResponse
	status = call(t, http.MethodPost, url+"/api/job/claim", agent.Token, map[string]any{
		"agent_id": "agent-1",
	}, &claimed)
	require.Equal(t, http.StatusOK, status)
	assert.NotNil(t, claimed.Job)
}

func TestAgentRoutesRefuseAdmins(t *testing.T) {
	url := testServer(t)
	admin := register(t, url)

	job := dispatch(t, url, admin.Token, "atkins build", "")

	// An admin is not an agent: the flag is a role rather than a rank.
	// Admitting one here would hand an admin token the private half of
	// every deploy key, which the admin listing withholds on purpose,
	// and let it settle jobs under an agent_id that never ran them.
	for _, route := range []string{"/api/agent/ssh-key", "/api/agent/policy"} {
		status := call(t, http.MethodGet, url+route, admin.Token, nil, nil)
		assert.Equal(t, http.StatusForbidden, status, route)
	}

	status := call(t, http.MethodPost, url+"/api/job/claim", admin.Token, map[string]any{
		"agent_id": "impostor",
	}, nil)
	assert.Equal(t, http.StatusForbidden, status)

	status = call(t, http.MethodPost, url+"/api/job/"+job.JobID+"/status", admin.Token, map[string]any{
		"status": model.JobStatusPassed,
	}, nil)
	assert.Equal(t, http.StatusForbidden, status)

	// The queue is untouched: the job is still there for a real agent.
	agent := enrol(t, url, "agent-1")
	var claimed claimResponse
	status = call(t, http.MethodPost, url+"/api/job/claim", agent.Token, map[string]any{
		"agent_id": "agent-1",
	}, &claimed)
	require.Equal(t, http.StatusOK, status)
	require.NotNil(t, claimed.Job)
	assert.Equal(t, job.JobID, claimed.Job.ID)
}

func TestAdminRoutesRefuseNonAdmins(t *testing.T) {
	url := testServer(t)
	register(t, url)

	agent := enrol(t, url, "agent-1")

	for _, path := range []string{"/api/admin/user", "/api/admin/repository", "/api/admin/setting", "/api/admin/ssh-key"} {
		status := call(t, http.MethodGet, url+path, agent.Token, nil, nil)
		assert.Equal(t, http.StatusForbidden, status, path)
	}
}

func TestSettings(t *testing.T) {
	url := testServer(t)
	admin := register(t, url)

	var settings []map[string]any
	status := call(t, http.MethodGet, url+"/api/admin/setting", admin.Token, nil, &settings)
	require.Equal(t, http.StatusOK, status)
	require.NotEmpty(t, settings)

	// Everything reports its default until it is overridden.
	for _, setting := range settings {
		assert.Equal(t, true, setting["is_default"], setting["name"])
	}

	setSetting(t, url, admin.Token, model.SettingRepositoryPolicy, model.PolicyAllowlist)

	status = call(t, http.MethodGet, url+"/api/admin/setting", admin.Token, nil, &settings)
	require.Equal(t, http.StatusOK, status)
	for _, setting := range settings {
		if setting["name"] == model.SettingRepositoryPolicy {
			assert.Equal(t, model.PolicyAllowlist, setting["value"])
			assert.Equal(t, false, setting["is_default"])
		}
	}

	// Resetting returns it to the default.
	status = call(t, http.MethodDelete, url+"/api/admin/setting/"+model.SettingRepositoryPolicy, admin.Token, nil, nil)
	require.Equal(t, http.StatusOK, status)
}

func TestSettingRejectsUnknownNameAndBadValue(t *testing.T) {
	url := testServer(t)
	admin := register(t, url)

	status := call(t, http.MethodPost, url+"/api/admin/setting/nonsense", admin.Token, map[string]string{
		"value": "1",
	}, nil)
	assert.Equal(t, http.StatusBadRequest, status)

	status = call(t, http.MethodPost, url+"/api/admin/setting/"+model.SettingJobMaxDepth, admin.Token, map[string]string{
		"value": "many",
	}, nil)
	assert.Equal(t, http.StatusBadRequest, status)
}

func TestAllowlistBlocksDispatch(t *testing.T) {
	url := testServer(t)
	admin := register(t, url)

	setSetting(t, url, admin.Token, model.SettingRepositoryPolicy, model.PolicyAllowlist)

	// An allowlist with no rules admits nothing. That is the point of
	// turning it on.
	status := call(t, http.MethodPost, url+"/api/dispatch", admin.Token, map[string]any{
		"repository": map[string]string{"remote_url": "git@github.com:titpetric/atkins.git"},
		"command":    "atkins",
	}, nil)
	assert.Equal(t, http.StatusForbidden, status)

	allowRepository(t, url, admin.Token, "github.com/titpetric/*")

	job := dispatch(t, url, admin.Token, "atkins", "")
	assert.NotEmpty(t, job.JobID)

	// A repository outside the rule is still refused.
	status = call(t, http.MethodPost, url+"/api/dispatch", admin.Token, map[string]any{
		"repository": map[string]string{"remote_url": "git@github.com:someone/else.git"},
		"command":    "atkins",
	}, nil)
	assert.Equal(t, http.StatusForbidden, status)
}

func TestAllowlistRuleLifecycle(t *testing.T) {
	url := testServer(t)
	admin := register(t, url)

	setSetting(t, url, admin.Token, model.SettingRepositoryPolicy, model.PolicyAllowlist)
	rule := allowRepository(t, url, admin.Token, "github.com/titpetric/*")
	ruleID, _ := rule["id"].(string)
	require.NotEmpty(t, ruleID)

	// Duplicate patterns are a conflict, not a second rule.
	status := call(t, http.MethodPost, url+"/api/admin/repository", admin.Token, map[string]string{
		"pattern": "github.com/titpetric/*",
	}, nil)
	assert.Equal(t, http.StatusConflict, status)

	// Disabling the rule closes the door again.
	status = call(t, http.MethodPost, url+"/api/admin/repository/"+ruleID, admin.Token, map[string]any{
		"is_active": false,
	}, nil)
	require.Equal(t, http.StatusOK, status)

	status = call(t, http.MethodPost, url+"/api/dispatch", admin.Token, map[string]any{
		"repository": map[string]string{"remote_url": "git@github.com:titpetric/atkins.git"},
		"command":    "atkins",
	}, nil)
	assert.Equal(t, http.StatusForbidden, status)

	status = call(t, http.MethodDelete, url+"/api/admin/repository/"+ruleID, admin.Token, nil, nil)
	assert.Equal(t, http.StatusNoContent, status)

	status = call(t, http.MethodDelete, url+"/api/admin/repository/"+ruleID, admin.Token, nil, nil)
	assert.Equal(t, http.StatusNotFound, status)
}

func TestAgentPolicy(t *testing.T) {
	url := testServer(t)
	admin := register(t, url)
	agent := enrol(t, url, "agent-1")

	var policy map[string]any
	status := call(t, http.MethodGet, url+"/api/agent/policy", agent.Token, nil, &policy)
	require.Equal(t, http.StatusOK, status)
	assert.Equal(t, model.PolicyOpen, policy["policy"])

	setSetting(t, url, admin.Token, model.SettingRepositoryPolicy, model.PolicyAllowlist)
	allowRepository(t, url, admin.Token, "github.com/titpetric/*")

	status = call(t, http.MethodGet, url+"/api/agent/policy", agent.Token, nil, &policy)
	require.Equal(t, http.StatusOK, status)
	assert.Equal(t, model.PolicyAllowlist, policy["policy"])
	assert.Equal(t, []any{"github.com/titpetric/*"}, policy["patterns"])
}

func TestUserFlags(t *testing.T) {
	url := testServer(t)
	admin := register(t, url)

	setSetting(t, url, admin.Token, model.SettingRegistrationOpen, "true")

	var second tokenResponse
	status := call(t, http.MethodPost, url+"/api/user/register", "", map[string]string{
		"email":    "dev@example.com",
		"username": "dev",
		"password": "correct-horse",
	}, &second)
	require.Equal(t, http.StatusCreated, status)

	var updated map[string]any
	status = call(t, http.MethodPost, url+"/api/admin/user/"+second.UserID, admin.Token, map[string]any{
		"is_admin": true,
	}, &updated)
	require.Equal(t, http.StatusOK, status)
	assert.Equal(t, true, updated["is_admin"])

	// Deactivating stops the account at the door.
	status = call(t, http.MethodPost, url+"/api/admin/user/"+second.UserID, admin.Token, map[string]any{
		"is_active": false,
	}, nil)
	require.Equal(t, http.StatusOK, status)

	status = call(t, http.MethodGet, url+"/api/user/whoami", second.Token, nil, nil)
	assert.Equal(t, http.StatusForbidden, status)

	// A user listing never carries password material.
	var users []map[string]any
	status = call(t, http.MethodGet, url+"/api/admin/user", admin.Token, nil, &users)
	require.Equal(t, http.StatusOK, status)
	require.NotEmpty(t, users)
	for _, user := range users {
		assert.NotContains(t, user, "password")
	}
}

func TestRefusesToRemoveTheLastAdmin(t *testing.T) {
	url := testServer(t)
	admin := register(t, url)

	status := call(t, http.MethodPost, url+"/api/admin/user/"+admin.UserID, admin.Token, map[string]any{
		"is_admin": false,
	}, nil)
	assert.Equal(t, http.StatusConflict, status)

	status = call(t, http.MethodPost, url+"/api/admin/user/"+admin.UserID, admin.Token, map[string]any{
		"is_active": false,
	}, nil)
	assert.Equal(t, http.StatusConflict, status)
}

// testSSHKey generates a throwaway ed25519 key.
//
// Generated rather than checked in: a real private key in the
// repository is a bad habit even when it guards nothing.
func testSSHKey(t *testing.T) string {
	t.Helper()

	_, private, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	block, err := ssh.MarshalPrivateKey(private, "atkins test")
	require.NoError(t, err)

	return string(pem.EncodeToMemory(block))
}

func TestSSHKeys(t *testing.T) {
	url := testServer(t)
	admin := register(t, url)

	var created map[string]any
	status := call(t, http.MethodPost, url+"/api/admin/ssh-key", admin.Token, map[string]string{
		"name":        "github",
		"host":        "github.com",
		"private_key": testSSHKey(t),
	}, &created)
	require.Equal(t, http.StatusCreated, status, created)

	// The response describes the key without disclosing it.
	assert.NotContains(t, created, "private_key")
	assert.Contains(t, created["fingerprint"], "SHA256:")
	assert.Contains(t, created["public_key"], "ssh-ed25519")

	keyID, _ := created["id"].(string)
	require.NotEmpty(t, keyID)

	var listed []map[string]any
	status = call(t, http.MethodGet, url+"/api/admin/ssh-key", admin.Token, nil, &listed)
	require.Equal(t, http.StatusOK, status)
	require.Len(t, listed, 1)
	assert.NotContains(t, listed[0], "private_key")

	// An agent, and only an agent, receives the private half.
	agent := enrol(t, url, "agent-1")
	var agentKeys []map[string]any
	status = call(t, http.MethodGet, url+"/api/agent/ssh-key", agent.Token, nil, &agentKeys)
	require.Equal(t, http.StatusOK, status)
	require.Len(t, agentKeys, 1)
	assert.Contains(t, agentKeys[0]["private_key"], "OPENSSH PRIVATE KEY")

	// Deactivating stops it being handed out.
	status = call(t, http.MethodPost, url+"/api/admin/ssh-key/"+keyID, admin.Token, map[string]any{
		"is_active": false,
	}, nil)
	require.Equal(t, http.StatusNoContent, status)

	status = call(t, http.MethodGet, url+"/api/agent/ssh-key", agent.Token, nil, &agentKeys)
	require.Equal(t, http.StatusOK, status)
	assert.Empty(t, agentKeys)

	status = call(t, http.MethodDelete, url+"/api/admin/ssh-key/"+keyID, admin.Token, nil, nil)
	assert.Equal(t, http.StatusNoContent, status)
}

func TestSSHKeyRejectsGarbage(t *testing.T) {
	url := testServer(t)
	admin := register(t, url)

	status := call(t, http.MethodPost, url+"/api/admin/ssh-key", admin.Token, map[string]string{
		"name":        "broken",
		"private_key": "not a key at all",
	}, nil)
	assert.Equal(t, http.StatusBadRequest, status)
}

func TestJobLogRoundTrip(t *testing.T) {
	url := testServer(t)
	admin := register(t, url)
	agent := enrol(t, url, "agent-1")

	job := dispatch(t, url, admin.Token, "atkins build", "")

	for _, chunk := range []string{"first\n", "second\n"} {
		status := call(t, http.MethodPost, url+"/api/job/"+job.JobID+"/log", agent.Token, map[string]string{
			"stream":  "output",
			"content": chunk,
		}, nil)
		require.Equal(t, http.StatusNoContent, status)
	}

	var entries []map[string]any
	status := call(t, http.MethodGet, url+"/api/job/"+job.JobID+"/log", admin.Token, nil, &entries)
	require.Equal(t, http.StatusOK, status)
	require.Len(t, entries, 2)
	assert.Equal(t, "first\n", entries[0]["content"])
	assert.Equal(t, float64(0), entries[0]["seq"])
	assert.Equal(t, float64(1), entries[1]["seq"])
}

func TestJobPageRenders(t *testing.T) {
	url := testServer(t)
	admin := register(t, url)

	job := dispatch(t, url, admin.Token, "atkins test:build", "")
	require.NotEmpty(t, job.ViewToken)

	// The page atkins prints is readable without a session: pasting the
	// whole URL into a browser has to just work.
	response, err := http.Get(url + "/job/" + job.JobID + "?t=" + job.ViewToken)
	require.NoError(t, err)
	defer response.Body.Close()
	assert.Equal(t, http.StatusOK, response.StatusCode)

	// Without the token it does not, however good the ULID guess was.
	response, err = http.Get(url + "/job/" + job.JobID)
	require.NoError(t, err)
	defer response.Body.Close()
	assert.Equal(t, http.StatusForbidden, response.StatusCode)

	response, err = http.Get(url + "/job/01ARZ3NDEKTSV4RRFFQ69G5FAV?t=" + job.ViewToken)
	require.NoError(t, err)
	defer response.Body.Close()
	assert.Equal(t, http.StatusForbidden, response.StatusCode)
}

func TestJobPageOnAPublicInstance(t *testing.T) {
	url := testServer(t)
	admin := register(t, url)

	setSetting(t, url, admin.Token, model.SettingJobVisibility, model.VisibilityPublic)

	// A public instance issues no token: there is nothing for one to
	// guard, and a secret in a URL that guards nothing is a habit worth
	// not forming.
	job := dispatch(t, url, admin.Token, "atkins test:build", "")
	assert.Empty(t, job.ViewToken)

	response, err := http.Get(url + "/job/" + job.JobID)
	require.NoError(t, err)
	defer response.Body.Close()
	assert.Equal(t, http.StatusOK, response.StatusCode)

	response, err = http.Get(url + "/job/01ARZ3NDEKTSV4RRFFQ69G5FAV")
	require.NoError(t, err)
	defer response.Body.Close()
	assert.Equal(t, http.StatusNotFound, response.StatusCode)

	// The index is the listing a public instance is for.
	response, err = http.Get(url + "/")
	require.NoError(t, err)
	defer response.Body.Close()
	assert.Equal(t, http.StatusOK, response.StatusCode)
}

func TestJobListingPageIsPrivateByDefault(t *testing.T) {
	url := testServer(t)
	admin := register(t, url)

	dispatch(t, url, admin.Token, "atkins build", "")

	// There is no token that could scope a listing and no session to
	// scope it by, so a private instance lists nothing — while still
	// answering, because this is the front door a health check probes.
	response, err := http.Get(url + "/")
	require.NoError(t, err)
	defer response.Body.Close()
	require.Equal(t, http.StatusOK, response.StatusCode)

	body, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	assert.NotContains(t, string(body), "atkins build")
	assert.Contains(t, string(body), "private")
}

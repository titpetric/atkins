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

// claim takes one job off the queue as an enrolled agent.
func claim(t *testing.T, url, token, agentID string) claimResponse {
	t.Helper()

	var claimed claimResponse
	status := call(t, http.MethodPost, url+"/api/job/claim", token, map[string]any{
		"agent_id": agentID,
	}, &claimed)
	require.Equal(t, http.StatusOK, status)
	require.NotNil(t, claimed.Job)
	require.NotNil(t, claimed.Job.StartedAt)
	require.NotNil(t, claimed.Job.LeaseExpiresAt)

	return claimed
}

// leaseWindow is how long the claim on a job is good for.
func leaseWindow(job *model.Job) time.Duration {
	return job.LeaseExpiresAt.Sub(*job.StartedAt)
}

func TestLeaseTTLSettingTakesEffectWithoutRestart(t *testing.T) {
	url := testServer(t)
	admin := register(t, url)
	agent := enrol(t, url, "agent-1")

	dispatch(t, url, admin.Token, "atkins build", "")
	dispatch(t, url, admin.Token, "atkins build", "")

	// The registry default until an admin says otherwise.
	assert.Equal(t, 15*time.Minute, leaseWindow(claim(t, url, agent.Token, "agent-1").Job))

	setSetting(t, url, admin.Token, model.SettingJobLeaseTTL, "30s")

	// The next claim uses the new window. `job.lease_ttl` is documented
	// as configuration an admin changes without a restart, and reading
	// it once at start-up made that promise false.
	assert.Equal(t, 30*time.Second, leaseWindow(claim(t, url, agent.Token, "agent-1").Job))
}

func TestMaxDepthSettingTakesEffectWithoutRestart(t *testing.T) {
	url := testServer(t)
	admin := register(t, url)

	parent := dispatch(t, url, admin.Token, "atkins", "")
	child := dispatch(t, url, admin.Token, "atkins", parent.JobID)
	require.Equal(t, int64(1), child.Depth)

	setSetting(t, url, admin.Token, model.SettingJobMaxDepth, "1")

	// A grandchild is depth 2, one past the limit that was just set.
	status := call(t, http.MethodPost, url+"/api/dispatch", admin.Token, map[string]any{
		"repository": map[string]string{"remote_url": "git@github.com:titpetric/atkins.git"},
		"command":    "atkins",
		"parent_id":  child.JobID,
	}, nil)
	assert.Equal(t, http.StatusUnprocessableEntity, status)
}

func TestConfiguredLeaseTTLOutranksTheRegistryDefault(t *testing.T) {
	url := testServerWith(t, func(opts *server.Options) {
		opts.LeaseTTL = 90 * time.Second
	})
	admin := register(t, url)
	agent := enrol(t, url, "agent-1")

	dispatch(t, url, admin.Token, "atkins build", "")
	dispatch(t, url, admin.Token, "atkins build", "")

	// ATKINS_LEASE_TTL is a decision; the registry default is not. A
	// setting nobody has stored must not silently outrank the
	// configuration file.
	assert.Equal(t, 90*time.Second, leaseWindow(claim(t, url, agent.Token, "agent-1").Job))

	// And an admin still overrides it, without a restart.
	setSetting(t, url, admin.Token, model.SettingJobLeaseTTL, "45s")
	assert.Equal(t, 45*time.Second, leaseWindow(claim(t, url, agent.Token, "agent-1").Job))
}

func TestConfiguredMaxDepthOutranksTheRegistryDefault(t *testing.T) {
	url := testServerWith(t, func(opts *server.Options) {
		opts.MaxJobDepth = 1
	})
	admin := register(t, url)

	parent := dispatch(t, url, admin.Token, "atkins", "")
	child := dispatch(t, url, admin.Token, "atkins", parent.JobID)

	status := call(t, http.MethodPost, url+"/api/dispatch", admin.Token, map[string]any{
		"repository": map[string]string{"remote_url": "git@github.com:titpetric/atkins.git"},
		"command":    "atkins",
		"parent_id":  child.JobID,
	}, nil)
	assert.Equal(t, http.StatusUnprocessableEntity, status)

	// Resetting a setting that was never stored leaves the configured
	// value in charge rather than falling back to the default of 3.
	status = call(t, http.MethodDelete, url+"/api/admin/setting/"+model.SettingJobMaxDepth, admin.Token, nil, nil)
	require.Equal(t, http.StatusOK, status)

	status = call(t, http.MethodPost, url+"/api/dispatch", admin.Token, map[string]any{
		"repository": map[string]string{"remote_url": "git@github.com:titpetric/atkins.git"},
		"command":    "atkins",
		"parent_id":  child.JobID,
	}, nil)
	assert.Equal(t, http.StatusUnprocessableEntity, status)
}

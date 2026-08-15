package worker

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/titpetric/atkins/client"
	"github.com/titpetric/atkins/server/model"
)

// policyServer serves a fixed policy, counting how often it is asked.
func policyServer(t *testing.T, policy client.PolicyResponse, fail *bool) (*Worker, *int) {
	t.Helper()

	calls := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/agent/enrol":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(client.TokenResponse{
				UserID:       "agent",
				Username:     "agent-1",
				Token:        "access",
				RefreshToken: "refresh",
				ExpiresAt:    time.Now().Add(time.Hour).Unix(),
			})
		case "/api/agent/policy":
			calls++
			if fail != nil && *fail {
				http.Error(w, `{"error":{"code":503,"message":"down"}}`, http.StatusServiceUnavailable)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(policy)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)

	t.Setenv("ATKINS_CREDENTIALS", t.TempDir()+"/credentials.json")

	c := client.New(server.URL)
	_, err := c.Enrol(t.Context(), client.EnrolRequest{Token: "token", AgentID: "agent-1"})
	require.NoError(t, err)

	return &Worker{opts: &Options{AgentID: "agent-1"}, client: c}, &calls
}

func TestAllowedUnderOpenPolicy(t *testing.T) {
	worker, _ := policyServer(t, client.PolicyResponse{Policy: model.PolicyOpen}, nil)

	allowed, err := worker.allowed(t.Context(), "github.com/anyone/anything")
	require.NoError(t, err)
	assert.True(t, allowed)
}

func TestAllowedUnderAllowlist(t *testing.T) {
	worker, _ := policyServer(t, client.PolicyResponse{
		Policy:   model.PolicyAllowlist,
		Patterns: []string{"github.com/titpetric/*"},
	}, nil)

	allowed, err := worker.allowed(t.Context(), "github.com/titpetric/atkins")
	require.NoError(t, err)
	assert.True(t, allowed)

	// A job queued while its repository was listed must not run once
	// the rule is gone; this is the agent's own gate on that.
	allowed, err = worker.allowed(t.Context(), "github.com/someone/else")
	require.NoError(t, err)
	assert.False(t, allowed)
}

func TestAllowlistWithNoPatternsAdmitsNothing(t *testing.T) {
	worker, _ := policyServer(t, client.PolicyResponse{Policy: model.PolicyAllowlist}, nil)

	allowed, err := worker.allowed(t.Context(), "github.com/titpetric/atkins")
	require.NoError(t, err)
	assert.False(t, allowed)
}

func TestUnreadablePolicyRefuses(t *testing.T) {
	fail := true
	worker, _ := policyServer(t, client.PolicyResponse{Policy: model.PolicyOpen}, &fail)

	// An agent that cannot tell what it may run must run nothing: a
	// server outage must not turn an allowlist into an open door.
	allowed, err := worker.allowed(t.Context(), "github.com/titpetric/atkins")
	assert.Error(t, err)
	assert.False(t, allowed)
}

func TestPolicyIsCached(t *testing.T) {
	worker, calls := policyServer(t, client.PolicyResponse{Policy: model.PolicyOpen}, nil)

	for range 5 {
		_, err := worker.allowed(t.Context(), "github.com/titpetric/atkins")
		require.NoError(t, err)
	}

	// The policy sits on the claim path; asking once per job would be
	// a request per job for a value that changes rarely.
	assert.Equal(t, 1, *calls)
}

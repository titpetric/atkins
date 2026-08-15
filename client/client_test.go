package client_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/titpetric/atkins/client"
)

// recorded is one request the fake server saw.
type recorded struct {
	Method string
	Path   string
	Auth   string
	Body   map[string]any
}

// fakeServer is a stand-in for the atkins API.
type fakeServer struct {
	*httptest.Server

	requests []recorded

	// handler answers a request path. A missing path is a 404.
	handler map[string]func(w http.ResponseWriter, body map[string]any)
}

func newFakeServer(t *testing.T) *fakeServer {
	t.Helper()

	fake := &fakeServer{handler: map[string]func(http.ResponseWriter, map[string]any){}}

	fake.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body := map[string]any{}
		_ = json.NewDecoder(r.Body).Decode(&body)

		fake.requests = append(fake.requests, recorded{
			Method: r.Method,
			Path:   r.URL.Path,
			Auth:   r.Header.Get("Authorization"),
			Body:   body,
		})

		handler, ok := fake.handler[r.URL.Path]
		if !ok {
			http.Error(w, `{"error":{"code":404,"message":"no such route"}}`, http.StatusNotFound)
			return
		}
		handler(w, body)
	}))
	t.Cleanup(fake.Close)

	t.Setenv("ATKINS_CREDENTIALS", filepath.Join(t.TempDir(), "credentials.json"))

	return fake
}

// respond registers a JSON response for a path.
func (f *fakeServer) respond(path string, status int, payload any) {
	f.handler[path] = func(w http.ResponseWriter, _ map[string]any) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		if payload != nil {
			_ = json.NewEncoder(w).Encode(payload)
		}
	}
}

// tokens is a stock token response.
func tokens(token string) client.TokenResponse {
	return client.TokenResponse{
		UserID:       "user-1",
		Username:     "ci",
		Token:        token,
		RefreshToken: "refresh-" + token,
		ExpiresAt:    time.Now().Add(time.Hour).Unix(),
	}
}

func TestLoginStoresCredentials(t *testing.T) {
	fake := newFakeServer(t)
	fake.respond("/api/user/login", http.StatusOK, tokens("access"))

	credential, err := client.New(fake.URL).Login(t.Context(), client.LoginRequest{
		Email:    "ci@example.com",
		Password: "correct-horse",
		Hostname: "workstation",
	})
	require.NoError(t, err)
	assert.Equal(t, "ci", credential.Username)
	assert.Equal(t, "access", credential.Token)

	// The credential is on disk, so the next command finds it.
	reopened, err := client.Open("")
	require.NoError(t, err)
	assert.Equal(t, fake.URL, reopened.Server())
	assert.Equal(t, "ci", reopened.Username())
}

func TestRegisterStoresCredentials(t *testing.T) {
	fake := newFakeServer(t)
	fake.respond("/api/user/register", http.StatusCreated, tokens("access"))

	_, err := client.New(fake.URL).Register(t.Context(), client.RegisterRequest{
		Email:    "ci@example.com",
		Username: "ci",
		Password: "correct-horse",
	})
	require.NoError(t, err)

	_, err = client.Open("")
	assert.NoError(t, err)
}

func TestAPIErrorCarriesTheServerMessage(t *testing.T) {
	fake := newFakeServer(t)
	fake.handler["/api/user/login"] = func(w http.ResponseWriter, _ map[string]any) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"code":401,"message":"invalid credentials"}}`))
	}

	_, err := client.New(fake.URL).Login(t.Context(), client.LoginRequest{Email: "ci@example.com"})
	require.Error(t, err)

	var apiErr *client.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, http.StatusUnauthorized, apiErr.StatusCode)
	assert.Equal(t, "invalid credentials", apiErr.Error())
}

func TestAPIErrorFallsBackToTheRawBody(t *testing.T) {
	fake := newFakeServer(t)
	fake.handler["/api/user/login"] = func(w http.ResponseWriter, _ map[string]any) {
		http.Error(w, "upstream exploded", http.StatusBadGateway)
	}

	_, err := client.New(fake.URL).Login(t.Context(), client.LoginRequest{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "upstream exploded")
}

func TestDispatchAndJobURL(t *testing.T) {
	fake := newFakeServer(t)
	fake.respond("/api/user/login", http.StatusOK, tokens("access"))
	fake.respond("/api/dispatch", http.StatusCreated, client.DispatchResponse{
		JobID:          "01ARZ3NDEKTSV4RRFFQ69G5FAV",
		RootID:         "01ARZ3NDEKTSV4RRFFQ69G5FAV",
		RepositorySlug: "github.com/titpetric/atkins",
		Status:         "pending",
	})

	c := client.New(fake.URL)
	_, err := c.Login(t.Context(), client.LoginRequest{Email: "ci@example.com"})
	require.NoError(t, err)

	response, err := c.Dispatch(t.Context(), client.DispatchRequest{
		Repository: client.RepositoryPayload{RemoteURL: "git@github.com:titpetric/atkins.git"},
		Command:    "atkins build",
	})
	require.NoError(t, err)
	assert.Equal(t, "01ARZ3NDEKTSV4RRFFQ69G5FAV", response.JobID)

	assert.Equal(t, fake.URL+"/job/01ARZ3NDEKTSV4RRFFQ69G5FAV", c.JobURL(response.JobID))

	// The dispatch call carried the access token.
	last := fake.requests[len(fake.requests)-1]
	assert.Equal(t, "/api/dispatch", last.Path)
	assert.Equal(t, "Bearer access", last.Auth)
}

func TestAuthenticatedCallWithoutCredentials(t *testing.T) {
	fake := newFakeServer(t)

	_, err := client.New(fake.URL).Dispatch(t.Context(), client.DispatchRequest{})
	assert.ErrorIs(t, err, client.ErrNotLoggedIn)
}

func TestExpiredTokenIsRefreshedFirst(t *testing.T) {
	fake := newFakeServer(t)
	fake.respond("/api/user/login", http.StatusOK, client.TokenResponse{
		UserID:   "user-1",
		Username: "ci",
		Token:    "stale",
		// Already expired: the next call must refresh before it goes.
		RefreshToken: "refresh-stale",
		ExpiresAt:    time.Now().Add(-time.Hour).Unix(),
	})
	fake.respond("/api/user/refreshToken", http.StatusOK, tokens("fresh"))
	fake.respond("/api/dispatch", http.StatusCreated, client.DispatchResponse{JobID: "job-1"})

	c := client.New(fake.URL)
	_, err := c.Login(t.Context(), client.LoginRequest{Email: "ci@example.com"})
	require.NoError(t, err)

	_, err = c.Dispatch(t.Context(), client.DispatchRequest{Command: "atkins"})
	require.NoError(t, err)

	paths := make([]string, 0, len(fake.requests))
	for _, request := range fake.requests {
		paths = append(paths, request.Path)
	}
	assert.Equal(t, []string{"/api/user/login", "/api/user/refreshToken", "/api/dispatch"}, paths)

	// The rotated token was used, and persisted for next time.
	last := fake.requests[len(fake.requests)-1]
	assert.Equal(t, "Bearer fresh", last.Auth)

	reopened, err := client.Open("")
	require.NoError(t, err)
	assert.Equal(t, fake.URL, reopened.Server())
}

func TestClaimReturnsNothingOnAnEmptyQueue(t *testing.T) {
	fake := newFakeServer(t)
	fake.respond("/api/agent/enrol", http.StatusOK, tokens("access"))
	fake.respond("/api/job/claim", http.StatusNoContent, nil)

	c := client.New(fake.URL)
	_, err := c.Enrol(t.Context(), client.EnrolRequest{Token: "shared", AgentID: "agent-1"})
	require.NoError(t, err)

	claimed, err := c.Claim(t.Context(), "agent-1", nil)
	require.NoError(t, err)
	// An empty poll is the normal case, not an error.
	assert.Nil(t, claimed)
}

func TestClaimReturnsTheJobAndItsRepository(t *testing.T) {
	fake := newFakeServer(t)
	fake.respond("/api/agent/enrol", http.StatusOK, tokens("access"))
	fake.respond("/api/job/claim", http.StatusOK, client.ClaimResponse{
		Job: &client.Job{
			ID:      "job-1",
			Command: "atkins build",
			Status:  "running",
		},
		Repository: &client.Repository{
			Slug:      "github.com/titpetric/atkins",
			RemoteURL: "git@github.com:titpetric/atkins.git",
		},
	})

	c := client.New(fake.URL)
	_, err := c.Enrol(t.Context(), client.EnrolRequest{Token: "shared", AgentID: "agent-1"})
	require.NoError(t, err)

	claimed, err := c.Claim(t.Context(), "agent-1", []string{"linux"})
	require.NoError(t, err)
	require.NotNil(t, claimed)
	assert.Equal(t, "job-1", claimed.Job.ID)
	assert.Equal(t, "github.com/titpetric/atkins", claimed.Repository.Slug)

	last := fake.requests[len(fake.requests)-1]
	assert.Equal(t, "agent-1", last.Body["agent_id"])
	assert.Equal(t, []any{"linux"}, last.Body["labels"])
}

func TestAgentReporting(t *testing.T) {
	fake := newFakeServer(t)
	fake.respond("/api/agent/enrol", http.StatusOK, tokens("access"))
	fake.respond("/api/job/job-1/status", http.StatusOK, nil)
	fake.respond("/api/job/job-1/heartbeat", http.StatusNoContent, nil)
	fake.respond("/api/job/job-1/log", http.StatusNoContent, nil)

	c := client.New(fake.URL)
	_, err := c.Enrol(t.Context(), client.EnrolRequest{Token: "shared", AgentID: "agent-1"})
	require.NoError(t, err)

	require.NoError(t, c.Heartbeat(t.Context(), "job-1", "agent-1"))
	require.NoError(t, c.AppendLog(t.Context(), "job-1", client.StreamOutput, "hello\n"))
	require.NoError(t, c.ReportStatus(t.Context(), "job-1", client.JobStatusRequest{
		Status:   client.StatusFailed,
		ExitCode: 3,
		Error:    "step failed",
	}))

	last := fake.requests[len(fake.requests)-1]
	assert.Equal(t, client.StatusFailed, last.Body["status"])
	assert.Equal(t, float64(3), last.Body["exit_code"])
}

func TestPolicyAndSSHKeys(t *testing.T) {
	fake := newFakeServer(t)
	fake.respond("/api/agent/enrol", http.StatusOK, tokens("access"))
	fake.respond("/api/agent/policy", http.StatusOK, client.PolicyResponse{
		Policy:   client.PolicyAllowlist,
		Patterns: []string{"github.com/titpetric/*"},
	})
	fake.respond("/api/agent/ssh-key", http.StatusOK, []client.AgentSSHKey{
		{ID: "key-1", Name: "github", PrivateKey: "PRIVATE"},
	})

	c := client.New(fake.URL)
	_, err := c.Enrol(t.Context(), client.EnrolRequest{Token: "shared", AgentID: "agent-1"})
	require.NoError(t, err)

	policy, err := c.Policy(t.Context())
	require.NoError(t, err)
	assert.Equal(t, client.PolicyAllowlist, policy.Policy)
	assert.Equal(t, []string{"github.com/titpetric/*"}, policy.Patterns)

	keys, err := c.SSHKeys(t.Context())
	require.NoError(t, err)
	require.Len(t, keys, 1)
	assert.Equal(t, "PRIVATE", keys[0].PrivateKey)
}

func TestLogoutForgetsTheCredentialEvenWhenTheServerFails(t *testing.T) {
	fake := newFakeServer(t)
	fake.respond("/api/user/login", http.StatusOK, tokens("access"))
	fake.handler["/api/user/logout"] = func(w http.ResponseWriter, _ map[string]any) {
		http.Error(w, "gone", http.StatusInternalServerError)
	}

	c := client.New(fake.URL)
	_, err := c.Login(t.Context(), client.LoginRequest{Email: "ci@example.com"})
	require.NoError(t, err)

	// Asking to log out of an unreachable server still logs you out here.
	assert.Error(t, c.Logout(t.Context()))

	_, err = client.Open("")
	assert.ErrorIs(t, err, client.ErrNotLoggedIn)
}

func TestLogoutRevokesAndForgets(t *testing.T) {
	fake := newFakeServer(t)
	fake.respond("/api/user/login", http.StatusOK, tokens("access"))
	fake.respond("/api/user/logout", http.StatusNoContent, nil)

	c := client.New(fake.URL)
	_, err := c.Login(t.Context(), client.LoginRequest{Email: "ci@example.com"})
	require.NoError(t, err)

	require.NoError(t, c.Logout(t.Context()))

	logout := fake.requests[len(fake.requests)-1]
	assert.Equal(t, "refresh-access", logout.Body["refresh_token"])

	_, err = client.Open("")
	assert.ErrorIs(t, err, client.ErrNotLoggedIn)
}

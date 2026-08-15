package worker

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/titpetric/atkins/client"
	"github.com/titpetric/atkins/server/model"
)

// gitRepository builds a repository with one commit and returns its
// path, usable as a clone remote.
func gitRepository(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	run := func(args ...string) {
		t.Helper()

		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=atkins", "GIT_AUTHOR_EMAIL=atkins@example.com",
			"GIT_COMMITTER_NAME=atkins", "GIT_COMMITTER_EMAIL=atkins@example.com",
		)
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, string(out))
	}

	run("init", "--initial-branch=main")
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "sub"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "sub", "marker.txt"), []byte("in-sub\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "marker.txt"), []byte("at-root\n"), 0o644))
	run("add", ".")
	run("commit", "-m", "initial")

	return dir
}

// agentServer is a minimal stand-in for the queue endpoints an agent
// talks to while running one job.
type agentServer struct {
	*httptest.Server

	mu         sync.Mutex
	output     strings.Builder
	status     client.JobStatusRequest
	statusSeen bool
	heartbeats int
}

func newAgentServer(t *testing.T, policy client.PolicyResponse) *agentServer {
	t.Helper()

	fake := &agentServer{}

	fake.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fake.mu.Lock()
		defer fake.mu.Unlock()

		switch {
		case r.URL.Path == "/api/agent/enrol":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(client.TokenResponse{
				UserID:       "agent",
				Username:     "agent-1",
				Token:        "access",
				RefreshToken: "refresh",
				ExpiresAt:    time.Now().Add(time.Hour).Unix(),
			})

		case r.URL.Path == "/api/agent/policy":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(policy)

		case r.URL.Path == "/api/agent/ssh-key":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode([]client.AgentSSHKey{})

		case strings.HasSuffix(r.URL.Path, "/log"):
			var req client.JobLogRequest
			_ = json.NewDecoder(r.Body).Decode(&req)
			fake.output.WriteString(req.Content)
			w.WriteHeader(http.StatusNoContent)

		case strings.HasSuffix(r.URL.Path, "/heartbeat"):
			fake.heartbeats++
			w.WriteHeader(http.StatusNoContent)

		case strings.HasSuffix(r.URL.Path, "/status"):
			_ = json.NewDecoder(r.Body).Decode(&fake.status)
			fake.statusSeen = true
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{})

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(fake.Close)

	return fake
}

// result reports what the server was told.
func (f *agentServer) result() (string, client.JobStatusRequest) {
	f.mu.Lock()
	defer f.mu.Unlock()

	return f.output.String(), f.status
}

// testWorker enrols an agent against the fake server.
func testWorker(t *testing.T, fake *agentServer) *Worker {
	t.Helper()

	t.Setenv("ATKINS_CREDENTIALS", filepath.Join(t.TempDir(), "credentials.json"))

	worker, err := New(t.Context(), &Options{
		Server:            fake.URL,
		Token:             "shared",
		AgentID:           "agent-1",
		DataDir:           t.TempDir(),
		HeartbeatInterval: time.Hour,
		JobTimeout:        time.Minute,
		Shell:             "/bin/sh",
	})
	require.NoError(t, err)

	return worker
}

// job builds a jobContext for a repository path.
func job(remote, workingDirectory, command string) *jobContext {
	return &jobContext{
		Job: &client.Job{
			ID:               "01ARZ3NDEKTSV4RRFFQ69G5FAV",
			RootID:           "01ARZ3NDEKTSV4RRFFQ69G5FAV",
			WorkingDirectory: workingDirectory,
			Command:          command,
			Branch:           "main",
		},
		Repository: &client.Repository{
			ID:            "repo-1",
			Slug:          "local/demo",
			RemoteURL:     remote,
			DefaultBranch: "main",
		},
	}
}

func TestRunClonesAndExecutes(t *testing.T) {
	remote := gitRepository(t)
	fake := newAgentServer(t, client.PolicyResponse{Policy: model.PolicyOpen})
	worker := testWorker(t, fake)

	worker.run(t.Context(), job(remote, "", "cat marker.txt"))

	output, status := fake.result()
	assert.Equal(t, client.StatusPassed, status.Status)
	assert.Equal(t, int64(0), status.ExitCode)
	assert.Contains(t, output, "at-root")

	// The repository is cached for the next job rather than re-cloned.
	assert.DirExists(t, filepath.Join(worker.opts.DataDir, "repos", "local", "demo.git"))
}

func TestRunUsesTheWorkingDirectory(t *testing.T) {
	remote := gitRepository(t)
	fake := newAgentServer(t, client.PolicyResponse{Policy: model.PolicyOpen})
	worker := testWorker(t, fake)

	worker.run(t.Context(), job(remote, "sub", "cat marker.txt"))

	output, status := fake.result()
	assert.Equal(t, client.StatusPassed, status.Status)
	// The job ran inside sub/, not at the repository root.
	assert.Contains(t, output, "in-sub")
	assert.NotContains(t, output, "at-root")
}

func TestRunExportsTheJobEnvironment(t *testing.T) {
	remote := gitRepository(t)
	fake := newAgentServer(t, client.PolicyResponse{Policy: model.PolicyOpen})
	worker := testWorker(t, fake)

	worker.run(t.Context(), job(remote, "", `echo "job=$ATKINS_JOB_ID ci=$CI nodispatch=$ATKINS_NO_DISPATCH repo=$ATKINS_REPOSITORY"`))

	output, status := fake.result()
	require.Equal(t, client.StatusPassed, status.Status)
	assert.Contains(t, output, "job=01ARZ3NDEKTSV4RRFFQ69G5FAV")
	assert.Contains(t, output, "ci=true")
	// Without this the atkins an agent runs would dispatch straight
	// back to the server and nothing would execute.
	assert.Contains(t, output, "nodispatch=1")
	assert.Contains(t, output, "repo=local/demo")
}

func TestRunReportsAFailingCommand(t *testing.T) {
	remote := gitRepository(t)
	fake := newAgentServer(t, client.PolicyResponse{Policy: model.PolicyOpen})
	worker := testWorker(t, fake)

	worker.run(t.Context(), job(remote, "", "echo nope >&2; exit 3"))

	output, status := fake.result()
	assert.Equal(t, client.StatusFailed, status.Status)
	assert.Equal(t, int64(3), status.ExitCode)
	// stderr is captured too: a failure with no output is not a
	// diagnosis.
	assert.Contains(t, output, "nope")
}

func TestRunRefusesADisallowedRepository(t *testing.T) {
	remote := gitRepository(t)
	fake := newAgentServer(t, client.PolicyResponse{
		Policy:   model.PolicyAllowlist,
		Patterns: []string{"github.com/titpetric/*"},
	})
	worker := testWorker(t, fake)

	worker.run(t.Context(), job(remote, "", "echo should-not-run"))

	output, status := fake.result()
	assert.Equal(t, client.StatusFailed, status.Status)
	assert.Contains(t, output, "not on the repository allowlist")
	assert.NotContains(t, output, "should-not-run")

	// Nothing was cloned: the refusal happens before the network does.
	assert.NoDirExists(t, filepath.Join(worker.opts.DataDir, "repos", "local", "demo.git"))
}

func TestRunReportsAMissingWorkingDirectory(t *testing.T) {
	remote := gitRepository(t)
	fake := newAgentServer(t, client.PolicyResponse{Policy: model.PolicyOpen})
	worker := testWorker(t, fake)

	worker.run(t.Context(), job(remote, "nowhere", "echo hello"))

	output, status := fake.result()
	assert.Equal(t, client.StatusFailed, status.Status)
	assert.Contains(t, output, "does not exist in the repository")
}

func TestRunReportsAnUnreachableRemote(t *testing.T) {
	fake := newAgentServer(t, client.PolicyResponse{Policy: model.PolicyOpen})
	worker := testWorker(t, fake)

	worker.run(t.Context(), job(filepath.Join(t.TempDir(), "missing"), "", "echo hello"))

	output, status := fake.result()
	assert.Equal(t, client.StatusFailed, status.Status)
	assert.Contains(t, output, "clone")
}

func TestRunCleansUpTheWorkTree(t *testing.T) {
	remote := gitRepository(t)
	fake := newAgentServer(t, client.PolicyResponse{Policy: model.PolicyOpen})
	worker := testWorker(t, fake)

	worker.run(t.Context(), job(remote, "", "echo hello"))

	// A checkout per job would fill the disk if it outlived the job.
	entries, err := os.ReadDir(filepath.Join(worker.opts.DataDir, "work"))
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestRunTimesOut(t *testing.T) {
	remote := gitRepository(t)
	fake := newAgentServer(t, client.PolicyResponse{Policy: model.PolicyOpen})
	worker := testWorker(t, fake)
	worker.opts.JobTimeout = 300 * time.Millisecond

	worker.run(t.Context(), job(remote, "", "sleep 30"))

	_, status := fake.result()
	// A stuck job is reported as timed out rather than holding its
	// lease until the server reclaims it.
	assert.Equal(t, client.StatusTimeout, status.Status)
	assert.Contains(t, status.Error, "timeout")
}

func TestRunWithoutARepository(t *testing.T) {
	fake := newAgentServer(t, client.PolicyResponse{Policy: model.PolicyOpen})
	worker := testWorker(t, fake)

	worker.run(t.Context(), &jobContext{Job: &client.Job{ID: "job-1", Command: "echo hello"}})

	_, status := fake.result()
	assert.Equal(t, client.StatusFailed, status.Status)
	assert.Contains(t, status.Error, "no repository")
}

func TestSecondJobFetchesTheCachedRepository(t *testing.T) {
	remote := gitRepository(t)
	fake := newAgentServer(t, client.PolicyResponse{Policy: model.PolicyOpen})
	worker := testWorker(t, fake)

	worker.run(t.Context(), job(remote, "", "echo first"))

	cache := filepath.Join(worker.opts.DataDir, "repos", "local", "demo.git")
	require.DirExists(t, cache)

	// Drop a marker into the cache: a second job must reuse this
	// directory rather than clone over it.
	marker := filepath.Join(cache, "atkins-cache-marker")
	require.NoError(t, os.WriteFile(marker, []byte("kept"), 0o644))

	worker.run(t.Context(), job(remote, "", "echo second"))

	assert.FileExists(t, marker)

	output, status := fake.result()
	assert.Equal(t, client.StatusPassed, status.Status)
	assert.Contains(t, output, "second")
}

package worker

import (
	"encoding/json"
	"fmt"
	"io"
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

// testRepository is a git repository built for the checkout tests, and
// the shas of the things a job can ask for.
type testRepository struct {
	// Path is the work tree, usable as a clone remote.
	Path string

	// First, Tagged and Head are the three commits on main, oldest
	// first. Feature is the tip of the second branch.
	First   string
	Tagged  string
	Head    string
	Feature string
}

// The names a job can put in its ref field for this repository.
const (
	testTag    = "v1.0.0"
	testBranch = "feature"
)

// gitRepository builds a repository with three commits on main, a tag
// two commits back, and a second branch.
//
// Every commit writes a different marker.txt, so a test can tell which
// commit ended up in the work tree by reading a file rather than by
// trusting what the agent reported about itself.
func gitRepository(t *testing.T) *testRepository {
	t.Helper()

	dir := t.TempDir()
	run := func(args ...string) string {
		t.Helper()

		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=atkins", "GIT_AUTHOR_EMAIL=atkins@example.com",
			"GIT_COMMITTER_NAME=atkins", "GIT_COMMITTER_EMAIL=atkins@example.com",
		)
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, string(out))

		return strings.TrimSpace(string(out))
	}

	commit := func(marker, message string) string {
		t.Helper()

		require.NoError(t, os.WriteFile(filepath.Join(dir, "marker.txt"), []byte(marker+"\n"), 0o644))
		run("add", ".")
		run("commit", "-m", message)

		return run("rev-parse", "HEAD")
	}

	run("init", "--initial-branch=main")
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "sub"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "sub", "marker.txt"), []byte("in-sub\n"), 0o644))

	repository := &testRepository{Path: dir}
	repository.First = commit("first", "initial")
	repository.Tagged = commit("tagged", "release")
	run("tag", testTag)
	repository.Head = commit("at-root", "after the release")

	run("checkout", "-b", testBranch)
	repository.Feature = commit("on-feature", "feature work")
	run("checkout", "main")

	return repository
}

// agentServer is a minimal stand-in for the queue endpoints an agent
// talks to while running one job.
type agentServer struct {
	*httptest.Server

	mu         sync.Mutex
	output     strings.Builder
	status     client.JobStatusRequest
	statusSeen bool
	checkout   client.JobCheckoutRequest
	heartbeats int

	// artefacts records what was uploaded, keyed by the path the agent
	// declared.
	artefacts map[string][]byte

	// refuseArtefacts turns every upload into a 422, standing in for a
	// server that has hit its per-job limit.
	refuseArtefacts bool
}

func newAgentServer(t *testing.T, policy client.PolicyResponse) *agentServer {
	t.Helper()

	fake := &agentServer{artefacts: map[string][]byte{}}

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

		case strings.HasSuffix(r.URL.Path, "/artefact"):
			body, _ := io.ReadAll(r.Body)
			if fake.refuseArtefacts {
				http.Error(w, "job has reached artefact.max_count", http.StatusUnprocessableEntity)
				return
			}
			fake.artefacts[r.URL.Query().Get("path")] = body

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(client.Artefact{
				ID:          "artefact-1",
				Path:        r.URL.Query().Get("path"),
				Size:        int64(len(body)),
				ContentType: r.Header.Get("Content-Type"),
				Checksum:    r.Header.Get(client.HeaderArtefactChecksum),
			})

		case strings.HasSuffix(r.URL.Path, "/log"):
			var req client.JobLogRequest
			_ = json.NewDecoder(r.Body).Decode(&req)
			fake.output.WriteString(req.Content)
			w.WriteHeader(http.StatusNoContent)

		case strings.HasSuffix(r.URL.Path, "/checkout"):
			_ = json.NewDecoder(r.Body).Decode(&fake.checkout)
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

// checkedOut reports the ref and commit the agent recorded.
func (f *agentServer) checkedOut() client.JobCheckoutRequest {
	f.mu.Lock()
	defer f.mu.Unlock()

	return f.checkout
}

// stored reports the artefacts the server accepted.
func (f *agentServer) stored() map[string][]byte {
	f.mu.Lock()
	defer f.mu.Unlock()

	uploaded := make(map[string][]byte, len(f.artefacts))
	for path, content := range f.artefacts {
		uploaded[path] = content
	}
	return uploaded
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

// job builds a jobContext for a repository path. It names no ref, which
// is the "run the default branch" case; tests that want something else
// set Ref or CloneDepth on the result.
func job(remote, workingDirectory, command string) *jobContext {
	return &jobContext{
		Job: &client.Job{
			ID:               "01ARZ3NDEKTSV4RRFFQ69G5FAV",
			RootID:           "01ARZ3NDEKTSV4RRFFQ69G5FAV",
			WorkingDirectory: workingDirectory,
			Command:          command,
		},
		Repository: &client.Repository{
			ID:            "repo-1",
			Slug:          "local/demo",
			RemoteURL:     remote,
			DefaultBranch: "main",
		},
	}
}

// checkoutJob builds a job that names a ref, at an optional depth.
func checkoutJob(remote, ref string, depth int64, command string) *jobContext {
	job := job(remote, "", command)
	job.Job.Ref = ref
	job.Job.CloneDepth = depth

	return job
}

func TestRunClonesAndExecutes(t *testing.T) {
	remote := gitRepository(t).Path
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

func TestRunChecksOutWhatTheJobNamed(t *testing.T) {
	repository := gitRepository(t)

	tests := []struct {
		name string
		ref  string

		// marker is the file content only that commit has, so the test
		// reads the work tree rather than the agent's own account of it.
		marker string
		commit string
		branch string
	}{
		{
			name:   "no ref takes the default branch",
			marker: "at-root",
			commit: repository.Head,
			branch: "main",
		},
		{
			name:   "a tag",
			ref:    testTag,
			marker: "tagged",
			commit: repository.Tagged,
		},
		{
			name:   "a fully qualified tag ref",
			ref:    "refs/tags/" + testTag,
			marker: "tagged",
			commit: repository.Tagged,
		},
		{
			name:   "a commit sha",
			ref:    repository.First,
			marker: "first",
			commit: repository.First,
		},
		{
			name:   "an abbreviated commit sha",
			ref:    repository.First[:10],
			marker: "first",
			commit: repository.First,
		},
		{
			name:   "a branch",
			ref:    testBranch,
			marker: "on-feature",
			commit: repository.Feature,
			branch: testBranch,
		},
		{
			name:   "a fully qualified branch ref",
			ref:    "refs/heads/" + testBranch,
			marker: "on-feature",
			commit: repository.Feature,
			branch: testBranch,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fake := newAgentServer(t, client.PolicyResponse{Policy: model.PolicyOpen})
			worker := testWorker(t, fake)

			worker.run(t.Context(), checkoutJob(repository.Path, test.ref, 0, `cat marker.txt; echo "branch=$ATKINS_BRANCH"`))

			output, status := fake.result()
			require.Equal(t, client.StatusPassed, status.Status, output)
			assert.Contains(t, output, test.marker)
			assert.Contains(t, output, "branch="+test.branch+"\n")

			// The commit is recorded, not just the ref: a tag moves, and
			// "v1.0.0" alone does not say which code ran.
			checkout := fake.checkedOut()
			assert.Equal(t, test.commit, checkout.CommitSHA)
			if test.ref != "" {
				assert.Equal(t, test.ref, checkout.Ref)
			} else {
				assert.Equal(t, "main", checkout.Ref)
			}
		})
	}
}

func TestRunFailsOnAMissingRef(t *testing.T) {
	remote := gitRepository(t).Path
	fake := newAgentServer(t, client.PolicyResponse{Policy: model.PolicyOpen})
	worker := testWorker(t, fake)

	worker.run(t.Context(), checkoutJob(remote, "v9.9.9", 0, "echo should-not-run"))

	output, status := fake.result()
	assert.Equal(t, client.StatusFailed, status.Status)
	// Naming the ref is the whole point: a job that quietly built the
	// default branch instead would be a green run of the wrong code.
	assert.Contains(t, output, `ref "v9.9.9" not found in local/demo`)
	assert.NotContains(t, output, "should-not-run")
	assert.Empty(t, fake.checkedOut().CommitSHA)
}

func TestRunClonesShallowWhenAskedTo(t *testing.T) {
	repository := gitRepository(t)

	for _, depth := range []int64{1, 2} {
		t.Run(fmt.Sprintf("depth %d", depth), func(t *testing.T) {
			fake := newAgentServer(t, client.PolicyResponse{Policy: model.PolicyOpen})
			worker := testWorker(t, fake)

			worker.run(t.Context(), checkoutJob(repository.Path, testTag, depth,
				`cat marker.txt; git rev-list --count HEAD`))

			output, status := fake.result()
			require.Equal(t, client.StatusPassed, status.Status, output)
			assert.Contains(t, output, "tagged")
			assert.Contains(t, output, fmt.Sprintf("%d\n", depth))
			assert.Equal(t, repository.Tagged, fake.checkedOut().CommitSHA)
		})
	}
}

func TestShallowJobLeavesTheCacheComplete(t *testing.T) {
	repository := gitRepository(t)
	fake := newAgentServer(t, client.PolicyResponse{Policy: model.PolicyOpen})
	worker := testWorker(t, fake)

	worker.run(t.Context(), checkoutJob(repository.Path, testTag, 1, "true"))

	// Depth is a property of the work tree, never of the cache: the
	// cache is shared, and one job asking for a single commit must not
	// cost the next one the history it already has.
	cache := filepath.Join(worker.opts.DataDir, "repos", "local", "demo.git")
	count, err := worker.gitOutput(t.Context(), cache, "rev-list", "--count", "refs/heads/main")
	require.NoError(t, err)
	assert.Equal(t, "3", count)

	// And a full job after a shallow one still gets the whole history.
	worker.run(t.Context(), checkoutJob(repository.Path, "", 0, "git rev-list --count HEAD"))

	output, status := fake.result()
	require.Equal(t, client.StatusPassed, status.Status, output)
	assert.Contains(t, output, "3\n")
}

func TestRunExportsTheCheckoutEnvironment(t *testing.T) {
	repository := gitRepository(t)
	fake := newAgentServer(t, client.PolicyResponse{Policy: model.PolicyOpen})
	worker := testWorker(t, fake)

	worker.run(t.Context(), checkoutJob(repository.Path, testTag, 0,
		`echo "ref=$ATKINS_REF sha=$ATKINS_COMMIT_SHA revision=$ATKINS_REVISION branch=$ATKINS_BRANCH"`))

	output, status := fake.result()
	require.Equal(t, client.StatusPassed, status.Status, output)
	assert.Contains(t, output, "ref="+testTag)
	assert.Contains(t, output, "sha="+repository.Tagged)
	// ATKINS_REVISION keeps its name and now always holds a resolved
	// commit, so a pipeline can pin an artefact even under a moving tag.
	assert.Contains(t, output, "revision="+repository.Tagged)
	// A tag is not a branch, and saying it was would be a lie a pipeline
	// could act on.
	assert.Contains(t, output, "branch=\n")
}

func TestRunUsesTheWorkingDirectory(t *testing.T) {
	remote := gitRepository(t).Path
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
	remote := gitRepository(t).Path
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
	remote := gitRepository(t).Path
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
	remote := gitRepository(t).Path
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
	remote := gitRepository(t).Path
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
	remote := gitRepository(t).Path
	fake := newAgentServer(t, client.PolicyResponse{Policy: model.PolicyOpen})
	worker := testWorker(t, fake)

	worker.run(t.Context(), job(remote, "", "echo hello"))

	// A checkout per job would fill the disk if it outlived the job.
	entries, err := os.ReadDir(filepath.Join(worker.opts.DataDir, "work"))
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestRunTimesOut(t *testing.T) {
	remote := gitRepository(t).Path
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
	remote := gitRepository(t).Path
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

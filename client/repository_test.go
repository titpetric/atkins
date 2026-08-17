package client_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/titpetric/atkins/client"
)

// gitRun runs one git command in dir, failing the test if it does.
func gitRun(t *testing.T, dir string, args ...string) {
	t.Helper()

	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=atkins", "GIT_AUTHOR_EMAIL=atkins@example.com",
		"GIT_COMMITTER_NAME=atkins", "GIT_COMMITTER_EMAIL=atkins@example.com",
	)

	output, err := cmd.CombinedOutput()
	require.NoError(t, err, string(output))
}

// gitRepository creates a repository with one commit and an origin.
func gitRepository(t *testing.T, remote string) string {
	t.Helper()

	dir := t.TempDir()

	gitRun(t, dir, "init", "--initial-branch=main")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "atkins.yml"), []byte("jobs: {}\n"), 0o644))
	gitRun(t, dir, "add", ".")
	gitRun(t, dir, "commit", "-m", "initial")
	if remote != "" {
		gitRun(t, dir, "remote", "add", "origin", remote)
	}

	return dir
}

// pushedRepository creates a repository whose commit is on a remote that
// exists, which is what a dispatchable checkout looks like.
func pushedRepository(t *testing.T) string {
	t.Helper()

	remote := filepath.Join(t.TempDir(), "origin.git")
	gitRun(t, t.TempDir(), "init", "--bare", remote)

	dir := gitRepository(t, remote)
	gitRun(t, dir, "push", "-u", "origin", "main")

	return dir
}

func TestDetectCheckout(t *testing.T) {
	dir := gitRepository(t, "git@github.com:titpetric/atkins.git")

	checkout, err := client.DetectCheckout(dir)
	require.NoError(t, err)

	assert.Equal(t, "git@github.com:titpetric/atkins.git", checkout.RemoteURL)
	assert.Equal(t, "main", checkout.Branch)
	assert.Len(t, checkout.Revision, 40)
	assert.Empty(t, checkout.WorkingDirectory)

	// The dispatched ref is the commit, not the branch: the run belongs
	// to the code in front of whoever started it.
	payload := checkout.Payload()
	assert.Equal(t, checkout.RemoteURL, payload.RemoteURL)
	assert.Equal(t, checkout.Revision, payload.Ref)
}

func TestDetectCheckoutReportsSubdirectory(t *testing.T) {
	dir := gitRepository(t, "git@github.com:titpetric/atkins.git")

	nested := filepath.Join(dir, "server", "api")
	require.NoError(t, os.MkdirAll(nested, 0o755))

	checkout, err := client.DetectCheckout(nested)
	require.NoError(t, err)
	assert.Equal(t, "server/api", checkout.WorkingDirectory)
}

func TestDetectCheckoutWithoutRemote(t *testing.T) {
	// A repository nobody else can fetch is not dispatchable.
	dir := gitRepository(t, "")

	_, err := client.DetectCheckout(dir)
	assert.ErrorIs(t, err, client.ErrNotARepository)
}

func TestDetectCheckoutOutsideRepository(t *testing.T) {
	_, err := client.DetectCheckout(t.TempDir())
	assert.ErrorIs(t, err, client.ErrNotARepository)
}

func TestPublishable(t *testing.T) {
	t.Run("accepts a clean checkout a remote has", func(t *testing.T) {
		checkout, err := client.DetectCheckout(pushedRepository(t))
		require.NoError(t, err)

		assert.False(t, checkout.Dirty)
		assert.False(t, checkout.Unpushed)
		assert.NoError(t, checkout.Publishable())
	})

	t.Run("refuses uncommitted changes", func(t *testing.T) {
		dir := pushedRepository(t)
		require.NoError(t, os.WriteFile(filepath.Join(dir, "atkins.yml"), []byte("jobs: {build: {}}\n"), 0o644))

		checkout, err := client.DetectCheckout(dir)
		require.NoError(t, err)

		// An agent would check out the commit and build what is in it,
		// which is not what is in front of the person dispatching.
		assert.True(t, checkout.Dirty)
		assert.ErrorIs(t, checkout.Publishable(), client.ErrDirtyCheckout)
	})

	t.Run("refuses an untracked file", func(t *testing.T) {
		dir := pushedRepository(t)
		require.NoError(t, os.WriteFile(filepath.Join(dir, "notes.md"), []byte("wip\n"), 0o644))

		checkout, err := client.DetectCheckout(dir)
		require.NoError(t, err)
		assert.ErrorIs(t, checkout.Publishable(), client.ErrDirtyCheckout)
	})

	t.Run("refuses a commit no remote has", func(t *testing.T) {
		dir := pushedRepository(t)
		require.NoError(t, os.WriteFile(filepath.Join(dir, "atkins.yml"), []byte("jobs: {build: {}}\n"), 0o644))
		gitRun(t, dir, "commit", "-am", "local only")

		checkout, err := client.DetectCheckout(dir)
		require.NoError(t, err)

		assert.False(t, checkout.Dirty)
		assert.True(t, checkout.Unpushed)
		assert.ErrorIs(t, checkout.Publishable(), client.ErrUnpushedCheckout)
	})

	t.Run("accepts a commit on a remote branch nothing tracks", func(t *testing.T) {
		dir := pushedRepository(t)
		gitRun(t, dir, "checkout", "-b", "detached-work")

		// The question is whether the commit can be cloned, not whether
		// the local branch has an upstream.
		checkout, err := client.DetectCheckout(dir)
		require.NoError(t, err)
		assert.NoError(t, checkout.Publishable())
	})

	t.Run("refuses a repository nobody has pushed at all", func(t *testing.T) {
		checkout, err := client.DetectCheckout(gitRepository(t, "git@github.com:titpetric/atkins.git"))
		require.NoError(t, err)

		assert.ErrorIs(t, checkout.Publishable(), client.ErrUnpushedCheckout)
	})
}

func TestLabels(t *testing.T) {
	t.Setenv(client.EnvLabels, "")
	assert.Nil(t, client.Labels())

	t.Setenv(client.EnvLabels, " linux , arm64 ,, docker ")
	assert.Equal(t, []string{"linux", "arm64", "docker"}, client.Labels())
}

func TestCommand(t *testing.T) {
	assert.Equal(t, "atkins", client.Command(nil))
	assert.Equal(t, "atkins test:build", client.Command([]string{"/usr/local/bin/atkins", "test:build"}))

	// The agent runs `atkins`, so that is what the job records however
	// the local binary is named. A build under test, the `atkins.old`
	// the install job leaves behind or a distro's `atkins-cli` would
	// otherwise queue a command the agent cannot find, and the job comes
	// back as exit 127 with nothing in the log.
	assert.Equal(t, "atkins test:build", client.Command([]string{"./bin/atkins-linux-amd64", "test:build"}))
	assert.Equal(t, "atkins test:docs", client.Command([]string{"/tmp/atkins-main", "test:docs"}))
	assert.Equal(t, "atkins", client.Command([]string{"/usr/local/bin/atkins.old"}))
}

func TestDispatchFailsWithoutCredentials(t *testing.T) {
	t.Setenv("ATKINS_CREDENTIALS", filepath.Join(t.TempDir(), "credentials.json"))

	// Handing a run over is asked for, so being unable to is an error
	// rather than a silent local run.
	dispatched, err := client.Dispatch(t.Context(), client.DispatchOptions{})
	require.Error(t, err)
	assert.Nil(t, dispatched)
}

func TestDispatchRespectsNoDispatch(t *testing.T) {
	t.Setenv(client.EnvNoDispatch, "1")

	dispatched, err := client.Dispatch(t.Context(), client.DispatchOptions{})
	require.ErrorIs(t, err, client.ErrDispatchDisabled)
	assert.Nil(t, dispatched)
}

func TestRecordSkipsWithoutCredentials(t *testing.T) {
	t.Setenv("ATKINS_CREDENTIALS", filepath.Join(t.TempDir(), "credentials.json"))

	// Recording is bookkeeping: it drops out silently rather than
	// costing the run it was meant to describe.
	recorder := client.Record(t.Context(), client.RecordOptions{})
	assert.Nil(t, recorder)

	// And every method tolerates that, so a caller has no branch to
	// write around it.
	assert.Empty(t, recorder.URL())
	assert.Empty(t, recorder.JobID())

	written, err := recorder.Write([]byte("output nobody records\n"))
	require.NoError(t, err)
	assert.Equal(t, 22, written)

	recorder.Log(t.Context(), "and none of this either")
	recorder.Finish(t.Context(), 0, nil)
	recorder.Cancelled(t.Context())
}

func TestRecordSkipsInsideAJob(t *testing.T) {
	// A run an agent started is already a job, streamed by the agent
	// that owns it; recording it again would file the same work twice.
	t.Setenv(client.EnvJobID, "01JBJZ0000000000000000000")

	assert.Nil(t, client.Record(t.Context(), client.RecordOptions{}))
}

func TestDispatchDisabled(t *testing.T) {
	for value, expected := range map[string]bool{
		"":      false,
		"0":     false,
		"false": false,
		"no":    false,
		"1":     true,
		"true":  true,
	} {
		t.Setenv(client.EnvNoDispatch, value)
		assert.Equal(t, expected, client.DispatchDisabled(), "ATKINS_NO_DISPATCH=%q", value)
	}
}

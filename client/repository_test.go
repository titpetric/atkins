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

// gitRepository creates a repository with one commit and an origin.
func gitRepository(t *testing.T, remote string) string {
	t.Helper()

	dir := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=atkins", "GIT_AUTHOR_EMAIL=atkins@example.com",
			"GIT_COMMITTER_NAME=atkins", "GIT_COMMITTER_EMAIL=atkins@example.com",
		)
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, string(output))
	}

	run("init", "--initial-branch=main")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "atkins.yml"), []byte("jobs: {}\n"), 0o644))
	run("add", ".")
	run("commit", "-m", "initial")
	if remote != "" {
		run("remote", "add", "origin", remote)
	}

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

	payload := checkout.Payload()
	assert.Equal(t, checkout.RemoteURL, payload.RemoteURL)
	assert.Equal(t, checkout.Revision, payload.Revision)
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

func TestLabels(t *testing.T) {
	t.Setenv(client.EnvLabels, "")
	assert.Nil(t, client.Labels())

	t.Setenv(client.EnvLabels, " linux , arm64 ,, docker ")
	assert.Equal(t, []string{"linux", "arm64", "docker"}, client.Labels())
}

func TestCommand(t *testing.T) {
	assert.Equal(t, "atkins", client.Command(nil))
	assert.Equal(t, "atkins test:build", client.Command([]string{"/usr/local/bin/atkins", "test:build"}))
}

func TestDispatchSkipsWithoutCredentials(t *testing.T) {
	t.Setenv("ATKINS_CREDENTIALS", filepath.Join(t.TempDir(), "credentials.json"))

	assert.Nil(t, client.Dispatch(t.Context(), client.DispatchOptions{}))
}

func TestDispatchRespectsNoDispatch(t *testing.T) {
	t.Setenv(client.EnvNoDispatch, "1")

	assert.Nil(t, client.Dispatch(t.Context(), client.DispatchOptions{}))
}

func TestDispatchRespectsLocal(t *testing.T) {
	assert.Nil(t, client.Dispatch(t.Context(), client.DispatchOptions{Local: true}))
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

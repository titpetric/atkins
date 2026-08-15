package worker

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/titpetric/atkins/client"
)

func TestSanitizeName(t *testing.T) {
	assert.Equal(t, "github", sanitizeName("github"))
	assert.Equal(t, "deploy_key-1.pem", sanitizeName("deploy_key-1.pem"))

	// A name is turned into a file name, so separators and traversal
	// have to stop being separators and traversal.
	assert.Equal(t, "etc-passwd", sanitizeName("../etc/passwd"))
	assert.Equal(t, "a-b", sanitizeName("a/b"))
	assert.Equal(t, "key", sanitizeName(""))
	assert.Equal(t, "key", sanitizeName("..."))
}

func TestEnsureTrailingNewline(t *testing.T) {
	assert.Equal(t, "value\n", ensureTrailingNewline("value"))
	assert.Equal(t, "value\n", ensureTrailingNewline("value\n"))
	assert.Equal(t, "value\n", ensureTrailingNewline("value\n\n\n"))
}

func TestWriteSSHKeys(t *testing.T) {
	worker := &Worker{opts: &Options{DataDir: t.TempDir()}}

	command, err := worker.writeSSHKeys([]client.AgentSSHKey{
		{Name: "github", Host: "github.com", PrivateKey: "PRIVATE-A"},
		{Name: "gitlab", Host: "gitlab.com", PrivateKey: "PRIVATE-B"},
	})
	require.NoError(t, err)

	dir := filepath.Join(worker.opts.DataDir, "ssh")

	// The directory holds credentials, and only its owner may read it.
	info, err := os.Stat(dir)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o700), info.Mode().Perm())

	for name, expected := range map[string]string{"github": "PRIVATE-A\n", "gitlab": "PRIVATE-B\n"} {
		path := filepath.Join(dir, name)

		contents, err := os.ReadFile(path)
		require.NoError(t, err)
		assert.Equal(t, expected, string(contents))

		info, err := os.Stat(path)
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())
	}

	assert.Contains(t, command, "IdentitiesOnly=yes")
	assert.Contains(t, command, "BatchMode=yes")
	// Nothing pinned, so trust on first use rather than prompting.
	assert.Contains(t, command, "StrictHostKeyChecking=accept-new")
	assert.Contains(t, command, "-i "+filepath.Join(dir, "github"))
	assert.Contains(t, command, "-i "+filepath.Join(dir, "gitlab"))

	// Identities are offered in a stable order, so a failure is
	// reproducible.
	assert.Less(t, strings.Index(command, "github"), strings.Index(command, "gitlab"))
}

func TestWriteSSHKeysPinsHostKeys(t *testing.T) {
	worker := &Worker{opts: &Options{DataDir: t.TempDir()}}

	command, err := worker.writeSSHKeys([]client.AgentSSHKey{
		{Name: "github", PrivateKey: "PRIVATE", KnownHosts: "github.com ssh-ed25519 AAAA"},
	})
	require.NoError(t, err)

	assert.Contains(t, command, "StrictHostKeyChecking=yes")
	assert.NotContains(t, command, "accept-new")

	known, err := os.ReadFile(filepath.Join(worker.opts.DataDir, "ssh", "known_hosts"))
	require.NoError(t, err)
	assert.Equal(t, "github.com ssh-ed25519 AAAA\n", string(known))
}

func TestWriteSSHKeysRemovesRevokedKeys(t *testing.T) {
	worker := &Worker{opts: &Options{DataDir: t.TempDir()}}

	_, err := worker.writeSSHKeys([]client.AgentSSHKey{{Name: "github", PrivateKey: "PRIVATE"}})
	require.NoError(t, err)

	path := filepath.Join(worker.opts.DataDir, "ssh", "github")
	require.FileExists(t, path)

	// A key removed on the server has to stop working here too, so the
	// directory is rebuilt rather than added to.
	command, err := worker.writeSSHKeys(nil)
	require.NoError(t, err)

	assert.Empty(t, command)
	assert.NoFileExists(t, path)
}

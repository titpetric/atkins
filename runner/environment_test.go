package runner_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/titpetric/atkins/runner"
)

func TestDiscoverEnvironment_GoMod(t *testing.T) {
	tmpDir := t.TempDir()
	err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module test"), 0o644)
	require.NoError(t, err)

	env, err := runner.DiscoverEnvironment(tmpDir)
	require.NoError(t, err)
	assert.Equal(t, tmpDir, env.Root)
}

func TestDiscoverEnvironment_Dockerfile(t *testing.T) {
	tmpDir := t.TempDir()
	err := os.WriteFile(filepath.Join(tmpDir, "Dockerfile"), []byte("FROM alpine"), 0o644)
	require.NoError(t, err)

	env, err := runner.DiscoverEnvironment(tmpDir)
	require.NoError(t, err)
	assert.Equal(t, tmpDir, env.Root)
}

func TestDiscoverEnvironment_DockerSubfolder(t *testing.T) {
	tmpDir := t.TempDir()
	err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module test"), 0o644)
	require.NoError(t, err)
	err = os.MkdirAll(filepath.Join(tmpDir, "docker"), 0o755)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(tmpDir, "docker", "Dockerfile"), []byte("FROM alpine"), 0o644)
	require.NoError(t, err)

	env, err := runner.DiscoverEnvironment(tmpDir)
	require.NoError(t, err)
	assert.Equal(t, tmpDir, env.Root)
}

func TestDiscoverEnvironment_ComposeYml(t *testing.T) {
	tmpDir := t.TempDir()
	err := os.WriteFile(filepath.Join(tmpDir, "compose.yml"), []byte("services: {}"), 0o644)
	require.NoError(t, err)

	env, err := runner.DiscoverEnvironment(tmpDir)
	require.NoError(t, err)
	assert.Equal(t, tmpDir, env.Root)
}

func TestDiscoverEnvironment_DockerComposeYml(t *testing.T) {
	tmpDir := t.TempDir()
	err := os.WriteFile(filepath.Join(tmpDir, "docker-compose.yml"), []byte("services: {}"), 0o644)
	require.NoError(t, err)

	env, err := runner.DiscoverEnvironment(tmpDir)
	require.NoError(t, err)
	assert.Equal(t, tmpDir, env.Root)
}

func TestDiscoverEnvironment_GitHubDir(t *testing.T) {
	tmpDir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, ".github"), 0o755))

	env, err := runner.DiscoverEnvironment(tmpDir)
	require.NoError(t, err)
	assert.Equal(t, tmpDir, env.Root)
}

func TestDiscoverEnvironment_SchemaDir(t *testing.T) {
	tmpDir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "schema"), 0o755))

	env, err := runner.DiscoverEnvironment(tmpDir)
	require.NoError(t, err)
	assert.Equal(t, tmpDir, env.Root)
}

func TestDiscoverEnvironment_MultipleMarkers(t *testing.T) {
	tmpDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module test"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "Dockerfile"), []byte("FROM alpine"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "compose.yml"), []byte("services: {}"), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, ".github"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "schema"), 0o755))

	env, err := runner.DiscoverEnvironment(tmpDir)
	require.NoError(t, err)
	assert.Equal(t, tmpDir, env.Root)
}

func TestDiscoverEnvironment_ParentDir(t *testing.T) {
	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "sub", "folder")
	require.NoError(t, os.MkdirAll(subDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module test"), 0o644))

	env, err := runner.DiscoverEnvironment(subDir)
	require.NoError(t, err)
	assert.Equal(t, tmpDir, env.Root)
}

func TestDiscoverEnvironment_NoMarkers(t *testing.T) {
	tmpDir := t.TempDir()

	_, err := runner.DiscoverEnvironment(tmpDir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no project root found")
}

func TestDiscoverEnvironment_Root(t *testing.T) {
	tmpDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module test"), 0o644))

	env, err := runner.DiscoverEnvironment(tmpDir)
	require.NoError(t, err)
	assert.Equal(t, tmpDir, env.Root)
}

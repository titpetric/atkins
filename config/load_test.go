package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/titpetric/atkins/config"
	"github.com/titpetric/atkins/internal/assert"
)

// writeDocument writes a configuration document under dir/.atkins.
func writeDocument(t *testing.T, dir, contents string) string {
	t.Helper()

	path := filepath.Join(dir, ".atkins", "config.yml")
	assert.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	assert.NoError(t, os.WriteFile(path, []byte(contents), 0o600))
	return path
}

func TestLoad(t *testing.T) {
	home := t.TempDir()
	project := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("ATKINS_DB", "")
	t.Setenv("ATKINS_RECORD", "")

	writeDocument(t, home, "storage:\n  record: false\n")
	writeDocument(t, project, "storage:\n  database: /project.db\n")

	// The project document sets its fields on top of the user document.
	cfg, err := config.Load(project)
	assert.NoError(t, err)
	assert.Equal(t, "/project.db", cfg.Storage.Database)
	assert.False(t, cfg.Storage.Record)

	// The environment sets ATKINS_DB last.
	t.Setenv("ATKINS_DB", "/env.db")
	cfg, err = config.Load(project)
	assert.NoError(t, err)
	assert.Equal(t, "/env.db", cfg.Storage.Database)
}

func TestLoadRejectsABrokenDocument(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	project := t.TempDir()
	writeDocument(t, project, "storage: [\n")

	_, err := config.Load(project)
	assert.Error(t, err)
}

func TestPaths(t *testing.T) {
	home := t.TempDir()
	project := t.TempDir()
	t.Setenv("HOME", home)

	user := filepath.Join(home, ".atkins", "config.yml")
	assert.Equal(t, []string{user}, config.Paths(project))

	found := writeDocument(t, project, "version: 1\n")
	assert.Equal(t, []string{user, found}, config.Paths(project))

	// The user document is also the project document of $HOME, and
	// merges once.
	writeDocument(t, home, "version: 1\n")
	assert.Equal(t, []string{user}, config.Paths(home))
}

func TestPartialDocumentKeepsDefaults(t *testing.T) {
	t.Parallel()

	path := writeDocument(t, t.TempDir(), "storage:\n  database: /tmp/events.db\n")

	cfg, err := config.LoadFile(path)
	assert.NoError(t, err)

	assert.Equal(t, "/tmp/events.db", cfg.Storage.Database)
	// Untouched fields keep the embedded defaults and never become
	// zero values.
	assert.Equal(t, config.Version, cfg.Version)
}

func TestDiscoverWalksUp(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	nested := filepath.Join(root, "runner", "testdata")

	assert.NoError(t, os.MkdirAll(nested, 0o755))
	path := writeDocument(t, root, "version: 1\n")

	found := config.Discover(nested)
	assert.NotEmpty(t, found)
	assert.Equal(t, path, found)
}

func TestDiscoverWithoutDocument(t *testing.T) {
	t.Parallel()

	assert.Empty(t, config.Discover(t.TempDir()))
}

func TestProjectPathForNewDocument(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	assert.Equal(t, filepath.Join(dir, ".atkins", "config.yml"), config.ProjectPath(dir))
}

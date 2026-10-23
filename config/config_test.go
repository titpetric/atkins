package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/titpetric/atkins/config"
	"github.com/titpetric/atkins/internal/assert"
)

func TestDefaultIsComplete(t *testing.T) {
	// The embedded document is the source of every default, so a field
	// missing from it would silently default to a zero value.
	cfg, err := config.Default()
	assert.NoError(t, err)

	assert.Equal(t, config.Version, cfg.Version)
	// Recording is on: the point of the event database is that nobody
	// has to remember to turn it on before the run worth measuring.
	assert.True(t, cfg.Storage.Record)
	// The database has no default path. The run discovers it from the
	// directory it happens in, and a path here would send every
	// project's rows to one file.
	assert.Empty(t, cfg.Storage.Database)
}

func TestDefaultValidates(t *testing.T) {
	cfg, err := config.Default()
	assert.NoError(t, err)
	assert.NoError(t, cfg.Validate())
}

func TestRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".atkins", "config.yml")

	cfg, err := config.Default()
	assert.NoError(t, err)

	cfg.Storage.Database = "/tmp/events.db"
	cfg.Storage.Record = false
	assert.NoError(t, cfg.Save(path))

	info, err := os.Stat(path)
	assert.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())

	reloaded, err := config.LoadFile(path)
	assert.NoError(t, err)
	assert.Equal(t, "/tmp/events.db", reloaded.Storage.Database)
	assert.False(t, reloaded.Storage.Record)
}

func TestPartialDocumentKeepsDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".atkins", "config.yml")

	assert.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	assert.NoError(t, os.WriteFile(path, []byte("storage:\n  database: /tmp/events.db\n"), 0o600))

	cfg, err := config.LoadFile(path)
	assert.NoError(t, err)

	assert.Equal(t, "/tmp/events.db", cfg.Storage.Database)
	// Untouched fields keep the embedded defaults and never become
	// zero values.
	assert.Equal(t, config.Version, cfg.Version)
}

func TestEnvironmentOverlay(t *testing.T) {
	cfg, err := config.Default()
	assert.NoError(t, err)

	cfg.Storage.Database = "/configured.db"

	cfg.ApplyEnvironment([]string{
		"ATKINS_DB=/from-env.db",
		"ATKINS_RECORD=false",
	})

	assert.Equal(t, "/from-env.db", cfg.Storage.Database)
	assert.False(t, cfg.Storage.Record)
}

func TestEmptyEnvironmentLeavesConfigAlone(t *testing.T) {
	cfg, err := config.Default()
	assert.NoError(t, err)

	cfg.Storage.Database = "/configured.db"

	// An exported-but-empty variable means "not configured here", not
	// "clear the configured value".
	cfg.ApplyEnvironment([]string{"ATKINS_DB="})

	assert.Equal(t, "/configured.db", cfg.Storage.Database)
}

func TestUnparseableEnvironmentIsIgnored(t *testing.T) {
	cfg, err := config.Default()
	assert.NoError(t, err)

	before := cfg.Storage.Record
	cfg.ApplyEnvironment([]string{"ATKINS_RECORD=maybe"})

	assert.Equal(t, before, cfg.Storage.Record)
}

func TestValidateFillsEmptyValues(t *testing.T) {
	cfg := &config.Config{}
	assert.NoError(t, cfg.Validate())

	assert.Equal(t, config.Version, cfg.Version)
}

func TestValidateRejectsAVersionFromTheFuture(t *testing.T) {
	cfg, err := config.Default()
	assert.NoError(t, err)

	// A newer build may have written fields this one would ignore in
	// silence, which is worse than refusing to start.
	cfg.Version = config.Version + 1
	assert.Error(t, cfg.Validate())
}

func TestDiscoverWalksUp(t *testing.T) {
	root := t.TempDir()
	nested := filepath.Join(root, "runner", "testdata")

	assert.NoError(t, os.MkdirAll(nested, 0o755))
	assert.NoError(t, os.MkdirAll(filepath.Join(root, ".atkins"), 0o755))
	assert.NoError(t, os.WriteFile(filepath.Join(root, ".atkins", "config.yml"), []byte("version: 1\n"), 0o600))

	found := config.Discover(nested)
	assert.NotEmpty(t, found)
	assert.Equal(t, filepath.Join(root, ".atkins", "config.yml"), found)
}

func TestDiscoverWithoutDocument(t *testing.T) {
	assert.Empty(t, config.Discover(t.TempDir()))
}

func TestProjectPathForNewDocument(t *testing.T) {
	dir := t.TempDir()
	assert.Equal(t, filepath.Join(dir, ".atkins", "config.yml"), config.ProjectPath(dir))
}

func TestFields(t *testing.T) {
	cfg, err := config.Default()
	assert.NoError(t, err)

	fields := cfg.Fields()
	assert.NotEmpty(t, fields)

	paths := make([]string, 0, len(fields))
	for _, field := range fields {
		paths = append(paths, field.Path)
	}

	assert.Contains(t, paths, "storage.database")
	assert.Contains(t, paths, "storage.record")
	// How long a run is kept is not a setting; the menu must not offer
	// one.
	assert.NotContains(t, paths, "storage.retention")
	// Load derives the document version; the menu never edits it.
	assert.NotContains(t, paths, "version")
}

func TestFieldSet(t *testing.T) {
	cfg, err := config.Default()
	assert.NoError(t, err)

	record, ok := cfg.Field("storage.record")
	assert.True(t, ok)
	assert.NoError(t, record.Set("false"))
	assert.False(t, cfg.Storage.Record)
	assert.Error(t, record.Set("maybe"))

	database, ok := cfg.Field("storage.database")
	assert.True(t, ok)
	assert.NoError(t, database.Set("/tmp/events.db"))
	assert.Equal(t, "/tmp/events.db", cfg.Storage.Database)
}

func TestFieldDisplayMarksEmpty(t *testing.T) {
	cfg, err := config.Default()
	assert.NoError(t, err)

	field, ok := cfg.Field("storage.database")
	assert.True(t, ok)
	assert.Equal(t, "(not set)", field.Display())
}

func TestEnvironmentNames(t *testing.T) {
	names := config.EnvironmentNames()

	assert.Equal(t, "ATKINS_DB", names["storage.database"])
	assert.Equal(t, "ATKINS_RECORD", names["storage.record"])
}

func TestSplitList(t *testing.T) {
	assert.Nil(t, config.SplitList(""))
	assert.Nil(t, config.SplitList(" , , "))
	assert.Equal(t, []string{"a", "b"}, config.SplitList(" a , b "))
}

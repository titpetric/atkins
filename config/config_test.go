package config_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/titpetric/atkins/config"
	"github.com/titpetric/atkins/internal/assert"
)

func TestDefaultIsComplete(t *testing.T) {
	t.Parallel()

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

func TestStorageConfig(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{Storage: config.StorageConfig{Database: "/events.db", Record: true}}
	encoded, err := cfg.Encode()
	assert.NoError(t, err)

	assert.Contains(t, string(encoded), "storage:\n  database: /events.db\n  record: true\n")
}

func TestConfig_Encode(t *testing.T) {
	t.Parallel()

	cfg, err := config.Default()
	assert.NoError(t, err)

	encoded, err := cfg.Encode()
	assert.NoError(t, err)

	// Two-space indentation, and every field of the document.
	assert.True(t, strings.HasPrefix(string(encoded), "version: 1\nstorage:\n  database: \"\"\n"))
	assert.Contains(t, string(encoded), "  record: true\n")
}

func TestRoundTrip(t *testing.T) {
	t.Parallel()

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

func TestEnvironmentOverlay(t *testing.T) {
	t.Parallel()

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
	t.Parallel()

	cfg, err := config.Default()
	assert.NoError(t, err)

	cfg.Storage.Database = "/configured.db"

	// An exported-but-empty variable means "not configured here", not
	// "clear the configured value".
	cfg.ApplyEnvironment([]string{"ATKINS_DB="})

	assert.Equal(t, "/configured.db", cfg.Storage.Database)
}

func TestUnparseableEnvironmentIsIgnored(t *testing.T) {
	t.Parallel()

	cfg, err := config.Default()
	assert.NoError(t, err)

	before := cfg.Storage.Record
	cfg.ApplyEnvironment([]string{"ATKINS_RECORD=maybe"})

	assert.Equal(t, before, cfg.Storage.Record)
}

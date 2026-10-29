package config_test

import (
	"testing"

	"github.com/titpetric/atkins/config"
	"github.com/titpetric/atkins/internal/assert"
)

func TestFields(t *testing.T) {
	t.Parallel()

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
	// Retention is fixed, and the menu lists no field for it.
	assert.NotContains(t, paths, "storage.retention")
	// Load derives the document version; the menu never edits it.
	assert.NotContains(t, paths, "version")
}

func TestFieldSet(t *testing.T) {
	t.Parallel()

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

func TestField_String(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{Storage: config.StorageConfig{Database: "/events.db", Record: true}}

	database, ok := cfg.Field("storage.database")
	assert.True(t, ok)
	assert.Equal(t, "/events.db", database.String())

	record, ok := cfg.Field("storage.record")
	assert.True(t, ok)
	assert.Equal(t, "true", record.String())
}

func TestField_Kind(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{}

	database, ok := cfg.Field("storage.database")
	assert.True(t, ok)
	assert.Equal(t, "text", database.Kind())

	record, ok := cfg.Field("storage.record")
	assert.True(t, ok)
	assert.Equal(t, "true or false", record.Kind())
}

func TestFieldDisplayMarksEmpty(t *testing.T) {
	t.Parallel()

	cfg, err := config.Default()
	assert.NoError(t, err)

	field, ok := cfg.Field("storage.database")
	assert.True(t, ok)
	assert.Equal(t, "(not set)", field.Display())
}

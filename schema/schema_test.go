package schema_test

import (
	"io/fs"
	"testing"

	"github.com/titpetric/atkins/internal/assert"
	"github.com/titpetric/atkins/schema"
)

func TestMigrations(t *testing.T) {
	t.Parallel()

	files, err := fs.Glob(schema.Migrations(), "*.up.sql")
	assert.NoError(t, err)
	assert.Len(t, files, 2)
	assert.Contains(t, files, "run.up.sql")
	assert.Contains(t, files, "command.up.sql")

	// The two tables are the two questions: what a job cost, and what a
	// command cost.
	runs, err := fs.ReadFile(schema.Migrations(), "run.up.sql")
	assert.NoError(t, err)
	assert.Contains(t, string(runs), "CREATE TABLE IF NOT EXISTS run")
	assert.Contains(t, string(runs), "duration_ms")

	commands, err := fs.ReadFile(schema.Migrations(), "command.up.sql")
	assert.NoError(t, err)
	assert.Contains(t, string(commands), "CREATE TABLE IF NOT EXISTS command")
	assert.Contains(t, string(commands), "normalized")
}

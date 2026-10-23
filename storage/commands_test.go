package storage_test

import (
	"context"
	"testing"
	"time"

	"github.com/titpetric/atkins/internal/assert"
	"github.com/titpetric/atkins/storage"
)

func TestCommandsNarrow(t *testing.T) {
	ctx := context.Background()
	store := newStorage(t)

	_, err := store.Record(ctx, storage.RecordRequest{Log: commandRun(base, []commandFixture{
		{command: "go test -v ./...", seconds: 1},
		{command: "go build -o bin/x .", seconds: 2},
		{command: "npm run build", seconds: 3},
	})})
	assert.NoError(t, err)

	tests, err := store.Commands(ctx, storage.CommandFilter{Normalized: "go test -v"})
	assert.NoError(t, err)
	assert.Len(t, tests, 1)
	assert.Equal(t, "go test -v ./...", tests[0].Command)

	// The newest command comes first, and a limit keeps it.
	newest, err := store.Commands(ctx, storage.CommandFilter{Limit: 1})
	assert.NoError(t, err)
	assert.Len(t, newest, 1)
	assert.Equal(t, "npm", newest[0].Tool)

	none, err := store.Commands(ctx, storage.CommandFilter{Until: base.Add(-time.Hour)})
	assert.NoError(t, err)
	assert.Len(t, none, 0)

	window, err := store.Commands(ctx, storage.CommandFilter{Since: base, Until: base.Add(time.Minute)})
	assert.NoError(t, err)
	assert.Len(t, window, 3)
}

package storage_test

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/titpetric/atkins/internal/assert"
	"github.com/titpetric/atkins/storage"
)

// TestRecorders covers the double write: a run records to the project's
// database and to the workspace's, the same rows in each.
func TestRecorders(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()

	stores, err := storage.OpenAll(ctx,
		filepath.Join(dir, "project.db"),
		filepath.Join(dir, "workspace.db"))
	assert.NoError(t, err)
	assert.Len(t, stores, 2)
	t.Cleanup(func() {
		for _, store := range stores {
			assert.NoError(t, store.Close())
		}
	})

	assert.NoError(t, storage.NewRecorders(stores).Record(ctx, sleepRun(time.Now())))

	// The same tree and the same commands in each, so a report of
	// either asks the same questions of it.
	for _, store := range stores {
		runs, err := store.Runs(ctx, storage.RunFilter{})
		assert.NoError(t, err)
		assert.Len(t, runs, 4, store.Path())

		commands, err := store.Commands(ctx, storage.CommandFilter{})
		assert.NoError(t, err)
		assert.Len(t, commands, 2, store.Path())
	}
}

// A database that will not take the write does not cost the run its row
// in the other, and the failure is still reported.
func TestRecordersKeepGoing(t *testing.T) {
	ctx := context.Background()
	store := newStorage(t)

	closed := newStorage(t)
	assert.NoError(t, closed.Close())

	recorders := storage.NewRecorders([]*storage.Storage{closed, store})
	assert.Error(t, recorders.Record(ctx, sleepRun(time.Now())))

	runs, err := store.Runs(ctx, storage.RunFilter{Kind: "job"})
	assert.NoError(t, err)
	assert.Len(t, runs, 1)
}

// No databases is no recording, and no error: a run whose history
// nobody could open has already happened.
func TestRecordersWithoutADatabase(t *testing.T) {
	assert.NoError(t, storage.NewRecorders(nil).Record(context.Background(), sleepRun(time.Now())))

	var none storage.Recorders
	assert.NoError(t, none.Record(context.Background(), sleepRun(time.Now())))
	assert.True(t, errors.Is(nil, nil))
}

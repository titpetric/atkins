package storage_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/titpetric/atkins/internal/assert"
	"github.com/titpetric/atkins/storage"
)

// Count says what started before a time, everything for a zero time;
// Prune takes those rows out to the last one and releases the space,
// and a zero time removes nothing.
func TestPruneCountsAndRemovesToTheLastRow(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "atkins.db")
	store, err := storage.NewStorageAt(ctx, path)
	assert.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, store.Close()) })

	old := base.Add(-72 * time.Hour)
	for i := 0; i < 1200; i++ {
		_, err := store.Record(ctx, storage.RecordRequest{Log: commandRun(old.Add(time.Duration(i)*time.Second), []commandFixture{
			{command: "go test ./...", seconds: 1},
		})})
		assert.NoError(t, err)
	}
	_, err = store.Record(ctx, storage.RecordRequest{Log: commandRun(base, []commandFixture{
		{command: "go test ./...", seconds: 1},
	})})
	assert.NoError(t, err)

	all, err := store.Count(ctx, time.Time{})
	assert.NoError(t, err)
	assert.Equal(t, &storage.PurgeResult{Runs: 3 * 1201, Commands: 1201}, all)
	before, err := store.Count(ctx, base.Add(-time.Hour))
	assert.NoError(t, err)
	assert.Equal(t, &storage.PurgeResult{Runs: 3 * 1200, Commands: 1200}, before)
	assert.Positive(t, storage.Size(path))
	assert.Equal(t, int64(0), storage.Size(filepath.Join(t.TempDir(), "none.db")))

	nothing, err := store.Prune(ctx, time.Time{})
	assert.NoError(t, err)
	assert.Equal(t, &storage.PurgeResult{}, nothing)

	removed, err := store.Prune(ctx, base.Add(-time.Hour))
	assert.NoError(t, err)
	assert.Equal(t, before, removed, "more rows than one Purge takes go in one Prune")
	left, err := store.Count(ctx, time.Time{})
	assert.NoError(t, err)
	assert.Equal(t, &storage.PurgeResult{Runs: 3, Commands: 1}, left)
}

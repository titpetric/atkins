package storage_test

import (
	"context"
	"testing"
	"time"

	"github.com/titpetric/atkins/internal/assert"
	"github.com/titpetric/atkins/storage"
)

func TestPurgeDropsOldHistoryInBatches(t *testing.T) {
	ctx := context.Background()
	store := newStorage(t)

	old := base.Add(-72 * time.Hour)
	_, err := store.Record(ctx, storage.RecordRequest{Log: commandRun(old, []commandFixture{
		{command: "go test ./...", seconds: 1},
	})})
	assert.NoError(t, err)
	_, err = store.Record(ctx, storage.RecordRequest{Log: commandRun(base, []commandFixture{
		{command: "go test ./...", seconds: 1},
	})})
	assert.NoError(t, err)

	// A zero cutoff removes nothing: the caller that forgot to set a
	// retention window must not lose its history to the default.
	nothing, err := store.Purge(ctx, storage.PurgeRequest{})
	assert.NoError(t, err)
	assert.Equal(t, int64(0), nothing.Runs)

	result, err := store.Purge(ctx, storage.PurgeRequest{Before: base.Add(-time.Hour), Batch: 1, Batches: 10})
	assert.NoError(t, err)
	assert.Equal(t, &storage.PurgeResult{Runs: 3, Commands: 1}, result)

	left, err := store.Runs(ctx, storage.RunFilter{})
	assert.NoError(t, err)
	assert.Len(t, left, 3)

	stats, err := store.Stats(ctx, storage.StatsRequest{})
	assert.NoError(t, err)
	assert.Equal(t, int64(1), stats[0].Runs)
}

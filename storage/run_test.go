package storage_test

import (
	"context"
	"testing"
	"time"

	"github.com/titpetric/atkins/internal/assert"
	"github.com/titpetric/atkins/storage"
)

func TestRunByID(t *testing.T) {
	ctx := context.Background()
	store := newStorage(t)

	result, err := store.Record(ctx, storage.RecordRequest{Log: sleepRun(base)})
	assert.NoError(t, err)

	run, err := store.Run(ctx, result.InvocationID)
	assert.NoError(t, err)
	assert.Equal(t, "pipeline", run.Kind)

	_, err = store.Run(ctx, "missing")
	assert.ErrorIs(t, err, storage.ErrNotFound)
}

func TestRunsNarrows(t *testing.T) {
	ctx := context.Background()
	store := newStorage(t)

	for _, at := range []time.Time{base, base.Add(48 * time.Hour)} {
		_, err := store.Record(ctx, storage.RecordRequest{Log: sleepRun(at)})
		assert.NoError(t, err)
	}

	steps, err := store.Runs(ctx, storage.RunFilter{Name: "sleep 2"})
	assert.NoError(t, err)
	assert.Len(t, steps, 2)

	// The newest run comes first, and a limit keeps it.
	newest, err := store.Runs(ctx, storage.RunFilter{Kind: "pipeline", Limit: 1})
	assert.NoError(t, err)
	assert.Len(t, newest, 1)
	assert.True(t, newest[0].StartedAt.Equal(base.Add(48*time.Hour)))

	first, err := store.Runs(ctx, storage.RunFilter{Kind: "pipeline", Since: base.Add(-time.Hour), Until: base.Add(time.Hour)})
	assert.NoError(t, err)
	assert.Len(t, first, 1)
	assert.True(t, first[0].StartedAt.Equal(base))
}

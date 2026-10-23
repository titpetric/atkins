package storage_test

import (
	"context"
	"testing"
	"time"

	"github.com/titpetric/atkins/internal/assert"
	"github.com/titpetric/atkins/storage"
)

func TestCommandUsage(t *testing.T) {
	ctx := context.Background()
	store := newStorage(t)

	for _, at := range []time.Time{base, base.Add(26 * time.Hour)} {
		_, err := store.Record(ctx, storage.RecordRequest{Log: commandRun(at, []commandFixture{
			{command: "go test ./...", seconds: 3},
			{command: "go test -v ./pkg", seconds: 2},
			{command: "docker compose up -d", seconds: 7},
			{command: "set -eu", seconds: 1},
		})})
		assert.NoError(t, err)
	}

	usage, err := store.CommandUsage(ctx, storage.UsageRequest{})
	assert.NoError(t, err)
	assert.Len(t, usage, 4)

	// Each day lists its keys, the most expensive first.
	assert.Equal(t, "2026-03-01", usage[0].Bucket)
	assert.Equal(t, "docker compose up", usage[0].Command)
	assert.Equal(t, "docker", usage[0].Tool)
	assert.Equal(t, int64(7000), usage[0].TotalMs)
	assert.Equal(t, "go test", usage[1].Command)
	assert.Equal(t, int64(2), usage[1].Runs)
	assert.Equal(t, int64(5000), usage[1].TotalMs)
	assert.Equal(t, "2026-03-02", usage[2].Bucket)

	_, err = store.CommandUsage(ctx, storage.UsageRequest{Bucket: "fortnight"})
	assert.Error(t, err)
}

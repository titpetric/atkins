package storage_test

import (
	"context"
	"testing"
	"time"

	"github.com/titpetric/atkins/internal/assert"
	"github.com/titpetric/atkins/storage"
)

func TestUsageBuckets(t *testing.T) {
	ctx := context.Background()
	store := newStorage(t)

	// Two days of runs. A daily bucket has two points and an hourly
	// one keeps them apart by hour as well.
	for _, at := range []time.Time{base, base.Add(2 * time.Hour), base.Add(26 * time.Hour)} {
		_, err := store.Record(ctx, storage.RecordRequest{Log: commandRun(at, []commandFixture{
			{command: "go test ./...", seconds: 3},
		})})
		assert.NoError(t, err)
	}

	daily, err := store.Usage(ctx, storage.UsageRequest{Bucket: storage.BucketDay})
	assert.NoError(t, err)
	assert.Len(t, daily, 2)
	assert.Equal(t, storage.UsageEntry{Bucket: "2026-03-01", Tool: "go", Runs: 2, TotalMs: 6000}, daily[0])
	assert.Equal(t, "2026-03-02", daily[1].Bucket)
	assert.Equal(t, int64(1), daily[1].Runs)

	hourly, err := store.Usage(ctx, storage.UsageRequest{Bucket: storage.BucketHour})
	assert.NoError(t, err)
	assert.Len(t, hourly, 3)
	assert.Equal(t, "2026-03-01T12:00", hourly[0].Bucket)

	// A week carries the label of its Monday. The Sunday run and the
	// Monday run after it fall in the two weeks they belong to.
	weekly, err := store.Usage(ctx, storage.UsageRequest{Bucket: storage.BucketWeek})
	assert.NoError(t, err)
	assert.Len(t, weekly, 2)
	assert.Equal(t, "2026-02-23", weekly[0].Bucket)
	assert.Equal(t, "2026-03-02", weekly[1].Bucket)

	// An empty bucket is a day, which is the series a usage chart draws
	// by default.
	byDefault, err := store.Usage(ctx, storage.UsageRequest{})
	assert.NoError(t, err)
	assert.Equal(t, daily, byDefault)

	_, err = store.Usage(ctx, storage.UsageRequest{Bucket: storage.Bucket("fortnight")})
	assert.Error(t, err)
}

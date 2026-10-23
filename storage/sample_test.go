package storage_test

import (
	"context"
	"testing"
	"time"

	"github.com/titpetric/atkins/internal/assert"
	"github.com/titpetric/atkins/storage"
)

func TestSample(t *testing.T) {
	ctx := context.Background()
	store := newStorage(t)

	for _, at := range []time.Time{base, base.Add(26 * time.Hour)} {
		_, err := store.Record(ctx, storage.RecordRequest{Log: commandRun(at, []commandFixture{
			{command: "go test -v ./pkg", seconds: 2},
			{command: "set -eu", seconds: 1},
			{command: "docker login -p s3cret registry", seconds: 3, exit: 1},
		})})
		assert.NoError(t, err)
	}

	samples, err := store.Samples(ctx, storage.SampleRequest{})
	assert.NoError(t, err)
	assert.Len(t, samples, 4)

	// Oldest first, each command by its key and never its command line.
	first := samples[0]
	assert.True(t, base.Equal(first.At), first.At)
	assert.Equal(t, "go test", first.Command)
	assert.Equal(t, "go", first.Tool)
	assert.Equal(t, "step", first.Kind)
	assert.Equal(t, int64(2000), first.DurationMs)
	assert.Equal(t, "Demo", first.Pipeline)
	assert.Equal(t, "default", first.Job)
	assert.NotEqual(t, "", first.InvocationID)

	login := samples[1]
	assert.Equal(t, "docker login", login.Command)
	assert.Equal(t, int64(1), login.ExitCode)
	assert.Equal(t, first.InvocationID, login.InvocationID)
	assert.True(t, base.Add(26*time.Hour).Equal(samples[2].At), samples[2].At)

	// A limit keeps the newest, still oldest first.
	newest, err := store.Samples(ctx, storage.SampleRequest{Limit: 2})
	assert.NoError(t, err)
	assert.Len(t, newest, 2)
	assert.True(t, base.Add(26*time.Hour).Equal(newest[0].At), newest[0].At)
	assert.Equal(t, "docker login", newest[1].Command)

	// The window and the tool filter apply as they do to the stats.
	since, err := store.Samples(ctx, storage.SampleRequest{Since: base.Add(time.Hour), Tool: "go"})
	assert.NoError(t, err)
	assert.Len(t, since, 1)
	assert.Equal(t, "go test", since[0].Command)
}

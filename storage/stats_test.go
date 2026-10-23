package storage_test

import (
	"context"
	"testing"
	"time"

	"github.com/titpetric/atkins/eventlog"
	"github.com/titpetric/atkins/internal/assert"
	"github.com/titpetric/atkins/storage"
)

// commandRun is one job whose steps are the commands given, each with
// its own duration in seconds and exit code.
func commandRun(at time.Time, commands []commandFixture) *eventlog.Log {
	log := &eventlog.Log{
		Metadata: eventlog.RunMetadata{CreatedAt: at, Pipeline: "Demo"},
		State: &eventlog.StateNode{
			Name:     "Demo",
			Children: []*eventlog.StateNode{{Name: "default", ID: "jobs.default"}},
		},
		Summary: &eventlog.RunSummary{Result: eventlog.ResultPass},
	}

	offset := 0.0
	for index, command := range commands {
		id := "jobs.default.steps." + string(rune('0'+index))
		log.State.Children[0].Children = append(log.State.Children[0].Children, &eventlog.StateNode{
			Name: command.command, ID: id, Start: offset, Duration: command.seconds,
		})
		log.Events = append(log.Events, &eventlog.Event{
			ID: id, Type: eventlog.EventTypeStep, Command: command.command,
			Dir: command.dir, ExitCode: command.exit, Start: offset, Duration: command.seconds,
		})
		offset += command.seconds
	}

	return log
}

type commandFixture struct {
	command string
	seconds float64
	exit    int
	dir     string
}

func TestStatsRanksByTotalTime(t *testing.T) {
	ctx := context.Background()
	store := newStorage(t)

	// A four second test that runs three times costs more than a nine
	// second build that runs once, and the report has to say so: that is
	// the difference between the slowest command and the expensive one.
	_, err := store.Record(ctx, storage.RecordRequest{Log: commandRun(base, []commandFixture{
		{command: "go test ./...", seconds: 4},
		{command: "go test -v ./...", seconds: 4},
		{command: "go test ./pkg", seconds: 4, exit: 1},
		{command: "go build -o bin/x .", seconds: 9},
	})})
	assert.NoError(t, err)

	stats, err := store.Stats(ctx, storage.StatsRequest{})
	assert.NoError(t, err)
	assert.Len(t, stats, 3)

	// The build is one run of nine seconds, the two plain test runs are
	// eight between them, and the report ranks on that total and not on
	// the single slowest command.
	assert.Equal(t, "go build -o", stats[0].Normalized)
	assert.Equal(t, int64(9000), stats[0].TotalMs)
	assert.Equal(t, int64(1), stats[0].Runs)

	assert.Equal(t, "go test", stats[1].Normalized)
	assert.Equal(t, int64(2), stats[1].Runs)
	assert.Equal(t, int64(8000), stats[1].TotalMs)
	assert.Equal(t, int64(4000), stats[1].AvgMs)
	assert.Equal(t, int64(4000), stats[1].MaxMs)
	assert.Equal(t, int64(1), stats[1].Failures)
	assert.NotNil(t, stats[1].LastAt)
	assert.Equal(t, "go", stats[1].Tool)

	// `go test -v` is its own row: a different flag set is a different
	// thing to measure.
	assert.Equal(t, "go test -v", stats[2].Normalized)
	assert.Equal(t, int64(1), stats[2].Runs)
}

func TestStatsNarrows(t *testing.T) {
	ctx := context.Background()
	store := newStorage(t)

	_, err := store.Record(ctx, storage.RecordRequest{Log: commandRun(base, []commandFixture{
		{command: "go test ./...", seconds: 4, dir: "/w/one"},
		{command: "npm run build", seconds: 7, dir: "/w/one/web"},
		{command: "make all", seconds: 2, dir: "/w/two"},
	})})
	assert.NoError(t, err)

	byTool, err := store.Stats(ctx, storage.StatsRequest{Tool: "npm"})
	assert.NoError(t, err)
	assert.Len(t, byTool, 1)
	assert.Equal(t, "npm run build", byTool[0].Normalized)

	// A directory keeps what ran below it too, so a workspace reports
	// its projects.
	byDir, err := store.Stats(ctx, storage.StatsRequest{Dir: "/w/one"})
	assert.NoError(t, err)
	assert.Len(t, byDir, 2)

	byLimit, err := store.Stats(ctx, storage.StatsRequest{Limit: 1})
	assert.NoError(t, err)
	assert.Len(t, byLimit, 1)
	assert.Equal(t, "npm run build", byLimit[0].Normalized)

	// A window that ended before the run began holds nothing.
	empty, err := store.Stats(ctx, storage.StatsRequest{Until: base.Add(-time.Hour)})
	assert.NoError(t, err)
	assert.Len(t, empty, 0)

	window, err := store.Stats(ctx, storage.StatsRequest{Since: base.Add(-time.Hour), Until: base.Add(time.Hour)})
	assert.NoError(t, err)
	assert.Len(t, window, 3)
}

func TestStatsOnAnEmptyDatabase(t *testing.T) {
	stats, err := newStorage(t).Stats(context.Background(), storage.StatsRequest{})
	assert.NoError(t, err)
	assert.Len(t, stats, 0)
}

func TestUsageBuckets(t *testing.T) {
	ctx := context.Background()
	store := newStorage(t)

	// Two days of runs, so a daily bucket has two points and an hourly
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
	assert.Equal(t, "2026-03-01", daily[0].Bucket)
	assert.Equal(t, "go", daily[0].Tool)
	assert.Equal(t, int64(2), daily[0].Runs)
	assert.Equal(t, int64(6000), daily[0].TotalMs)
	assert.Equal(t, "2026-03-02", daily[1].Bucket)
	assert.Equal(t, int64(1), daily[1].Runs)

	hourly, err := store.Usage(ctx, storage.UsageRequest{Bucket: storage.BucketHour})
	assert.NoError(t, err)
	assert.Len(t, hourly, 3)
	assert.Equal(t, "2026-03-01T12:00", hourly[0].Bucket)

	// A week is labelled with its Monday, so the Sunday run and the
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

	_, err = store.Usage(ctx, storage.UsageRequest{Bucket: "fortnight"})
	assert.Error(t, err)
}

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
	assert.Equal(t, int64(1), result.Commands)
	assert.Equal(t, int64(3), result.Runs)

	left, err := store.Runs(ctx, storage.RunFilter{})
	assert.NoError(t, err)
	assert.Len(t, left, 3)

	stats, err := store.Stats(ctx, storage.StatsRequest{})
	assert.NoError(t, err)
	assert.Equal(t, int64(1), stats[0].Runs)
}

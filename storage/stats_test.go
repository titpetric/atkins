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
	assert.Len(t, stats, 2)

	// The three test runs are twelve seconds between them, across
	// their flags, and rank above the nine second build: the report
	// ranks on the total time of a key.
	assert.Equal(t, "go test", stats[0].Command)
	assert.Equal(t, int64(3), stats[0].Runs)
	assert.Equal(t, int64(12000), stats[0].TotalMs)
	assert.Equal(t, int64(4000), stats[0].AvgMs)
	assert.Equal(t, int64(4000), stats[0].MaxMs)
	assert.Equal(t, int64(1), stats[0].Failures)
	assert.NotNil(t, stats[0].LastAt)
	assert.Equal(t, "go", stats[0].Tool)

	assert.NotNil(t, stats[1].LastAt)
	assert.Equal(t, storage.CommandStat{
		Command: "go build",
		Tool:    "go",
		Runs:    1,
		TotalMs: 9000,
		AvgMs:   9000,
		MaxMs:   9000,
		LastAt:  stats[1].LastAt,
	}, stats[1])

	// Durations round at both ends. 0.6ms and 1.4ms each store as 1ms,
	// 0.4ms as none, and the three average to 1ms.
	_, err = store.Record(ctx, storage.RecordRequest{Log: commandRun(base, []commandFixture{
		{command: "pwd", seconds: 0.0006},
		{command: "pwd", seconds: 0.0014},
		{command: "pwd", seconds: 0.0004},
	})})
	assert.NoError(t, err)

	quick, err := store.Stats(ctx, storage.StatsRequest{Tool: "pwd"})
	assert.NoError(t, err)
	assert.Equal(t, int64(3), quick[0].Runs)
	assert.Equal(t, int64(2), quick[0].TotalMs)
	assert.Equal(t, int64(1), quick[0].AvgMs)

	// A builtin is neither recorded nor reported.
	_, err = store.Record(ctx, storage.RecordRequest{Log: commandRun(base, []commandFixture{
		{command: "set -eu", seconds: 1},
		{command: "cd /src", seconds: 1},
	})})
	assert.NoError(t, err)

	all, err := store.Stats(ctx, storage.StatsRequest{})
	assert.NoError(t, err)
	assert.Len(t, all, 3)
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
	assert.Equal(t, "npm run build", byTool[0].Command)

	// A directory keeps what ran below it too. A workspace reports
	// its projects.
	byDir, err := store.Stats(ctx, storage.StatsRequest{Dir: "/w/one"})
	assert.NoError(t, err)
	assert.Len(t, byDir, 2)

	byLimit, err := store.Stats(ctx, storage.StatsRequest{Limit: 1})
	assert.NoError(t, err)
	assert.Len(t, byLimit, 1)
	assert.Equal(t, "npm run build", byLimit[0].Command)

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

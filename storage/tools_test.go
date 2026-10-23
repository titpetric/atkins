package storage_test

import (
	"context"
	"testing"

	"github.com/titpetric/atkins/internal/assert"
	"github.com/titpetric/atkins/storage"
)

func TestToolsRollUpTheirCommands(t *testing.T) {
	ctx := context.Background()
	store := newStorage(t)

	// `go` is one tool whether it builds, tests or vets, and the share
	// of a week it takes is what a per-command table cannot show.
	_, err := store.Record(ctx, storage.RecordRequest{Log: commandRun(base, []commandFixture{
		{command: "go test ./...", seconds: 4},
		{command: "go test -v ./...", seconds: 4, exit: 1},
		{command: "go build -o bin/x .", seconds: 9},
		{command: "npm run build", seconds: 7},
	})})
	assert.NoError(t, err)

	tools, err := store.Tools(ctx, storage.StatsRequest{})
	assert.NoError(t, err)
	assert.Len(t, tools, 2)

	assert.Equal(t, "go", tools[0].Tool)
	assert.Equal(t, int64(2), tools[0].Commands)
	assert.Equal(t, int64(3), tools[0].Runs)
	assert.Equal(t, int64(17000), tools[0].TotalMs)
	assert.Equal(t, int64(1), tools[0].Failures)

	assert.Equal(t, storage.ToolStat{Tool: "npm", Commands: 1, Runs: 1, TotalMs: 7000}, tools[1])

	byLimit, err := store.Tools(ctx, storage.StatsRequest{Limit: 1})
	assert.NoError(t, err)
	assert.Len(t, byLimit, 1)
}

func TestToolsOnAnEmptyDatabase(t *testing.T) {
	tools, err := newStorage(t).Tools(context.Background(), storage.StatsRequest{})
	assert.NoError(t, err)
	assert.Len(t, tools, 0)
}

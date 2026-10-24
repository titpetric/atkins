package main

import (
	"bytes"
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	"github.com/titpetric/atkins/eventlog"
	"github.com/titpetric/atkins/internal/assert"
	"github.com/titpetric/atkins/storage"
)

// An age is days or a Go duration; nothing, zero, a negative age and a
// word are errors.
func TestParseAge(t *testing.T) {
	t.Parallel()
	for text, want := range map[string]time.Duration{"31d": 31 * 24 * time.Hour, " 7d ": 7 * 24 * time.Hour, "24h": 24 * time.Hour, "90m": 90 * time.Minute} {
		age, err := ParseAge(text)
		assert.NoError(t, err, text)
		assert.Equal(t, want, age, text)
	}
	for _, text := range []string{"", "0d", "-1d", "0", "lots", "1x", "-3h"} {
		_, err := ParseAge(text)
		assert.Error(t, err, text)
	}
}

// oneStep is a run of one job with one step, started at a time.
func oneStep(at time.Time) *eventlog.Log {
	return &eventlog.Log{
		Metadata: eventlog.RunMetadata{CreatedAt: at, Pipeline: "Demo"},
		State: &eventlog.StateNode{Name: "Demo", Children: []*eventlog.StateNode{{Name: "default", ID: "jobs.default",
			Children: []*eventlog.StateNode{{Name: "go test ./...", ID: "jobs.default.steps.0", Duration: 1}}}}},
		Events:  []*eventlog.Event{{ID: "jobs.default.steps.0", Type: eventlog.EventTypeStep, Command: "go test ./...", Duration: 1}},
		Summary: &eventlog.RunSummary{Result: eventlog.ResultPass},
	}
}

// --prune takes --db; a dry run counts the old rows and removes nothing,
// a prune removes them and says what is left; a database that is not
// there is an error and stays absent.
func TestPrune(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "atkins.db")
	store, err := storage.NewStorageAt(ctx, path)
	assert.NoError(t, err)
	old := time.Now().Add(-40 * 24 * time.Hour)
	for _, at := range []time.Time{old, old.Add(time.Minute), time.Now().Add(-time.Hour)} {
		_, err := store.Record(ctx, storage.RecordRequest{Log: oneStep(at)})
		assert.NoError(t, err)
	}
	assert.NoError(t, store.Close())

	var out bytes.Buffer
	assert.Error(t, runPrune(ctx, &out, &Options{OlderThan: "31d"}), "no --db")
	none := filepath.Join(t.TempDir(), "none.db")
	assert.Error(t, runPrune(ctx, &out, &Options{DB: none, OlderThan: "31d"}))
	assert.NoFileExists(t, none)
	assert.Error(t, runPrune(ctx, &out, &Options{DB: path, OlderThan: "soon"}))

	out.Reset()
	assert.NoError(t, runPrune(ctx, &out, &Options{DB: path, OlderThan: "31d", DryRun: true}))
	assert.True(t, strings.HasPrefix(out.String(), path+": would remove 6 runs, 2 commands older than 31d; 3 runs, 1 command left; "), out.String())
	store, err = storage.NewStorageAt(ctx, path)
	assert.NoError(t, err)
	all, err := store.Count(ctx, time.Time{})
	assert.NoError(t, err)
	assert.Equal(t, int64(9), all.Runs, "a dry run removes nothing")
	assert.NoError(t, store.Close())

	out.Reset()
	assert.NoError(t, runPrune(ctx, &out, &Options{DB: path, OlderThan: "7d"}))
	assert.True(t, strings.HasPrefix(out.String(), path+": removed 6 runs, 2 commands older than 7d; 3 runs, 1 command left; "), out.String())
	assert.Contains(t, out.String(), " before, ")
	store, err = storage.NewStorageAt(ctx, path)
	assert.NoError(t, err)
	all, err = store.Count(ctx, time.Time{})
	assert.NoError(t, err)
	assert.Equal(t, &storage.PurgeResult{Runs: 3, Commands: 1}, all)
	assert.NoError(t, store.Close())
}

package storage_test

import (
	"context"
	"testing"

	"github.com/titpetric/atkins/eventlog"
	"github.com/titpetric/atkins/internal/assert"
	"github.com/titpetric/atkins/storage"
)

func TestRecordKeepsTheJobAndItsCommands(t *testing.T) {
	ctx := context.Background()
	store := newStorage(t)

	result, err := store.Record(ctx, storage.RecordRequest{Log: sleepRun(base), Dir: "/w/demo"})
	assert.NoError(t, err)
	assert.NotEmpty(t, result.InvocationID)
	assert.Equal(t, storage.RecordResult{InvocationID: result.InvocationID, Runs: 4, Commands: 2}, *result)

	// The pipeline node carries the metadata of the whole invocation and
	// the time of everything under it.
	pipelines, err := store.Runs(ctx, storage.RunFilter{Kind: "pipeline"})
	assert.NoError(t, err)
	assert.Len(t, pipelines, 1)
	assert.Equal(t, "Demo", pipelines[0].Pipeline)
	assert.Equal(t, "atkins.yml", pipelines[0].File)
	assert.Equal(t, "/w/demo", pipelines[0].Dir)
	assert.Equal(t, "titpetric/atkins", pipelines[0].Repository)
	assert.Equal(t, "main", pipelines[0].Branch)
	assert.Equal(t, "abc1234", pipelines[0].CommitSha)
	assert.Equal(t, int64(3000), pipelines[0].DurationMs)
	assert.Equal(t, int64(2), pipelines[0].Steps)

	// The job costs the three seconds of both sleeps, which is the whole
	// point of recording the tree along with the commands.
	jobs, err := store.Runs(ctx, storage.RunFilter{Kind: "job"})
	assert.NoError(t, err)
	assert.Len(t, jobs, 1)
	assert.Equal(t, "default", jobs[0].Name)
	assert.Equal(t, int64(3000), jobs[0].DurationMs)
	assert.Equal(t, pipelines[0].ID, jobs[0].ParentID)

	// The steps cost one and two, and belong to the job.
	steps, err := store.Runs(ctx, storage.RunFilter{Kind: "step"})
	assert.NoError(t, err)
	assert.Len(t, steps, 2)
	for _, step := range steps {
		assert.Equal(t, jobs[0].ID, step.ParentID)
	}

	// Every row of the invocation shares its id.
	all, err := store.Runs(ctx, storage.RunFilter{Invocation: result.InvocationID})
	assert.NoError(t, err)
	assert.Len(t, all, 4)

	// The commands keep what ran, and point at the step that ran it.
	commands, err := store.Commands(ctx, storage.CommandFilter{Invocation: result.InvocationID})
	assert.NoError(t, err)
	assert.Len(t, commands, 2)
	for _, command := range commands {
		assert.Equal(t, "sleep", command.Normalized)
		assert.Equal(t, "sleep", command.Tool)
		assert.Equal(t, "/w/demo", command.Dir)
		assert.NotEmpty(t, command.ParentID)
	}
}

func TestRecordToleratesNothingToRecord(t *testing.T) {
	ctx := context.Background()
	store := newStorage(t)

	// A run that produced no document, and a document with no tree,
	// write nothing and return no error: the run has happened and its
	// exit code stands.
	result, err := store.Record(ctx, storage.RecordRequest{})
	assert.NoError(t, err)
	assert.Equal(t, 0, result.Runs)

	result, err = store.Record(ctx, storage.RecordRequest{Log: &eventlog.Log{}})
	assert.NoError(t, err)
	assert.Equal(t, 0, result.Runs)
}

func TestRecordKeepsFailuresAndSubstitutions(t *testing.T) {
	ctx := context.Background()
	store := newStorage(t)

	log := &eventlog.Log{
		Metadata: eventlog.RunMetadata{CreatedAt: base, Pipeline: "Demo"},
		State: &eventlog.StateNode{
			Name: "Demo", Result: eventlog.ResultFail, Duration: 2,
			Children: []*eventlog.StateNode{{
				Name: "test", ID: "jobs.test", Result: eventlog.ResultFail, Duration: 2,
				Children: []*eventlog.StateNode{
					{Name: "go test", ID: "jobs.test.steps.0", Result: eventlog.ResultFail, Duration: 2},
					{Name: "skipped", ID: "jobs.test.steps.1", Result: eventlog.ResultSkipped},
				},
			}},
		},
		Events: []*eventlog.Event{
			{ID: "jobs.test.steps.0", Type: eventlog.EventTypeStep, Command: "go test ./...", ExitCode: 1, Error: "FAIL", Duration: 2},
			{ID: "subst-1", ParentID: "jobs.test", Type: eventlog.EventTypeSubstitution, Command: "git rev-parse HEAD", Duration: 0.01},
			// An event with no command is a step outcome; the run rows
			// already carry it.
			{ID: "jobs.test.steps.1", Type: eventlog.EventTypeStep, Result: eventlog.ResultSkipped},
		},
		Summary: &eventlog.RunSummary{Duration: 2, TotalSteps: 2, FailedSteps: 1, Result: eventlog.ResultFail},
	}

	result, err := store.Record(ctx, storage.RecordRequest{Log: log})
	assert.NoError(t, err)
	assert.Equal(t, 2, result.Commands)

	pipelines, err := store.Runs(ctx, storage.RunFilter{Kind: "pipeline"})
	assert.NoError(t, err)
	assert.Equal(t, "fail", pipelines[0].Result)

	steps, err := store.Runs(ctx, storage.RunFilter{Kind: "step", Name: "skipped"})
	assert.NoError(t, err)
	assert.Equal(t, "skipped", steps[0].Result)

	// The substitution names its job as the parent, and the job's row is
	// what it points at.
	jobs, err := store.Runs(ctx, storage.RunFilter{Kind: "job"})
	assert.NoError(t, err)
	substitutions, err := store.Commands(ctx, storage.CommandFilter{Parent: jobs[0].ID})
	assert.NoError(t, err)
	assert.Len(t, substitutions, 1)
	assert.Equal(t, "substitution", substitutions[0].Kind)
	assert.Equal(t, "git rev-parse HEAD", substitutions[0].Normalized)

	failed, err := store.Commands(ctx, storage.CommandFilter{Tool: "go"})
	assert.NoError(t, err)
	assert.Len(t, failed, 1)
	assert.Equal(t, int64(1), failed[0].ExitCode)
	assert.Equal(t, "FAIL", failed[0].Error)
}

package storage_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "modernc.org/sqlite" // the driver storage opens

	"github.com/titpetric/atkins/eventlog"
	"github.com/titpetric/atkins/internal/assert"
	"github.com/titpetric/atkins/storage"
)

// base is the moment the fixture runs start, so a test can assert on a
// window without a clock of its own.
var base = time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)

// newStorage returns an empty in-memory database.
func newStorage(t *testing.T) *storage.Storage {
	t.Helper()

	store, err := storage.NewStorageAt(context.Background(), storage.Memory)
	assert.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, store.Close())
	})

	return store
}

// sleepRun is the run the feature was asked for: a job that runs two
// sleeps, so the commands cost a second and two, and the job that holds
// them costs three.
func sleepRun(at time.Time) *eventlog.Log {
	return &eventlog.Log{
		Metadata: eventlog.RunMetadata{
			RunID:      "01HRUN",
			CreatedAt:  at,
			Pipeline:   "Demo",
			File:       "atkins.yml",
			ModulePath: "github.com/titpetric/atkins",
			Git: &eventlog.GitInfo{
				Repository: "titpetric/atkins",
				Branch:     "main",
				Commit:     "abc1234",
			},
		},
		State: &eventlog.StateNode{
			Name: "Demo", Result: eventlog.ResultPass, Duration: 3,
			Children: []*eventlog.StateNode{{
				// The tree shows a job with its description; the row keeps
				// the canonical name out of the id, so a year of runs of
				// `default` stays one group when the description changes.
				Name: "default - Run everything", ID: "jobs.default", Result: eventlog.ResultPass, Duration: 3, Steps: 2,
				Children: []*eventlog.StateNode{
					{Name: "sleep 1", ID: "jobs.default.steps.0", Result: eventlog.ResultPass, Duration: 1},
					{Name: "sleep 2", ID: "jobs.default.steps.1", Result: eventlog.ResultPass, Start: 1, Duration: 2},
				},
			}},
		},
		Events: []*eventlog.Event{
			{ID: "jobs.default.steps.0", Type: eventlog.EventTypeStep, Command: "sleep 1", Dir: "/w/demo", Duration: 1},
			{ID: "jobs.default.steps.1", Type: eventlog.EventTypeStep, Command: "sleep 2", Dir: "/w/demo", Start: 1, Duration: 2},
		},
		Summary: &eventlog.RunSummary{Duration: 3, TotalSteps: 2, PassedSteps: 2, Result: eventlog.ResultPass},
	}
}

func TestRecordKeepsTheJobAndItsCommands(t *testing.T) {
	ctx := context.Background()
	store := newStorage(t)

	result, err := store.Record(ctx, storage.RecordRequest{Log: sleepRun(base), Dir: "/w/demo"})
	assert.NoError(t, err)
	assert.Equal(t, 4, result.Runs)
	assert.Equal(t, 2, result.Commands)
	assert.NotEmpty(t, result.InvocationID)

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
	// point of recording the tree and not only the commands.
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

	// A run that produced no document, and a document with no tree, are
	// both nothing to write rather than an error: the run has already
	// happened and its exit code is already decided.
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

func TestDiscoverWalksUpToTheProject(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	root := t.TempDir()
	nested := filepath.Join(root, "a", "b")
	assert.NoError(t, os.MkdirAll(nested, 0o755))
	assert.NoError(t, os.MkdirAll(filepath.Join(root, ".atkins"), 0o755))

	// Every run in a checkout records to the same file, whichever
	// directory inside it the run happens in, and to the workspace's
	// beside it.
	paths, err := storage.Discover(nested)
	assert.NoError(t, err)
	assert.Equal(t, filepath.Join(root, ".atkins", "atkins.db"), paths.Project)
	assert.Equal(t, filepath.Join(home, ".atkins", "atkins.db"), paths.Workspace)
	assert.Equal(t, []string{paths.Project, paths.Workspace}, paths.List())
}

func TestDiscoverFallsBackToHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	// A directory that belongs to no project still records, so a stray
	// command in /tmp is counted too. It has no project, so the
	// workspace's is the one database and the run records once.
	paths, err := storage.Discover(t.TempDir())
	assert.NoError(t, err)
	assert.Equal(t, "", paths.Project)
	assert.Equal(t, filepath.Join(home, ".atkins", "atkins.db"), paths.Workspace)
	assert.Equal(t, []string{paths.Workspace}, paths.List())
	assert.DirExists(t, filepath.Join(home, ".atkins"))
}

// A directory inside $HOME whose nearest .atkins is the workspace's own
// has no project: the run records once and not twice to one file.
func TestDiscoverInsideHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	assert.NoError(t, os.MkdirAll(filepath.Join(home, ".atkins"), 0o755))

	nested := filepath.Join(home, "scratch")
	assert.NoError(t, os.MkdirAll(nested, 0o755))

	paths, err := storage.Discover(nested)
	assert.NoError(t, err)
	assert.Equal(t, "", paths.Project)
	assert.Equal(t, filepath.Join(home, ".atkins", "atkins.db"), paths.Workspace)
}

// OpenAll keeps what opened and reports the first failure, so a run
// records to the database that works.
func TestOpenAll(t *testing.T) {
	ctx := context.Background()
	good := filepath.Join(t.TempDir(), "atkins.db")

	stores, err := storage.OpenAll(ctx, good)
	assert.NoError(t, err)
	assert.Len(t, stores, 1)
	assert.NoError(t, stores[0].Close())

	// A path inside a file is no directory to make a database in.
	blocked := filepath.Join(t.TempDir(), "file")
	assert.NoError(t, os.WriteFile(blocked, []byte("x"), 0o644))

	stores, err = storage.OpenAll(ctx, good, filepath.Join(blocked, "atkins.db"))
	assert.Error(t, err)
	assert.Len(t, stores, 1)
	assert.Equal(t, good, stores[0].Path())
	assert.NoError(t, stores[0].Close())
}

func TestStorageOnAFile(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "atkins.db")

	store, err := storage.NewStorageAt(ctx, path)
	assert.NoError(t, err)
	assert.Equal(t, path, store.Path())

	_, err = store.Record(ctx, storage.RecordRequest{Log: sleepRun(base)})
	assert.NoError(t, err)
	assert.NoError(t, store.Close())

	// Reopening applies no migration twice and reads what the first run
	// wrote.
	again, err := storage.NewStorageAt(ctx, path)
	assert.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, again.Close())
	})

	runs, err := again.Runs(ctx, storage.RunFilter{Kind: "job"})
	assert.NoError(t, err)
	assert.Len(t, runs, 1)
}

func TestNilStorageIsHarmless(t *testing.T) {
	var store *storage.Storage

	assert.Equal(t, "", store.Path())
	assert.NoError(t, store.Close())
	assert.Nil(t, store.Recorder())

	// A recorder nobody could open records nothing and says so quietly,
	// because the run it belongs to has already happened.
	assert.NoError(t, store.Recorder().Record(context.Background(), sleepRun(time.Now())))
}

func TestRecorderWritesThroughTheRunnerInterface(t *testing.T) {
	ctx := context.Background()
	store := newStorage(t)

	// Recorded as of now: a recorder prunes what it records as it
	// records it, and the fixture's own date is outside the window.
	assert.NoError(t, store.Recorder().Record(ctx, sleepRun(time.Now())))

	runs, err := store.Runs(ctx, storage.RunFilter{Kind: "job"})
	assert.NoError(t, err)
	assert.Len(t, runs, 1)

	// The recorder records where it was made, which is where the run
	// happened.
	pipelines, err := store.Runs(ctx, storage.RunFilter{Kind: "pipeline"})
	assert.NoError(t, err)
	assert.NotEmpty(t, pipelines[0].Dir)
}

func TestRecorderAppliesRetention(t *testing.T) {
	ctx := context.Background()
	store := newStorage(t)
	recorder := store.Recorder()

	// The window is 31 days and nobody chooses it. A run older than that
	// is outside it the moment it is recorded, and goes with the write
	// that recorded it: the window holds with no daemon to hold it.
	assert.NoError(t, recorder.Record(ctx, sleepRun(time.Now().Add(-storage.Retention-time.Hour))))
	assert.NoError(t, recorder.Record(ctx, sleepRun(time.Now().Add(-storage.Retention+time.Hour))))
	assert.NoError(t, recorder.Record(ctx, sleepRun(time.Now())))

	runs, err := store.Runs(ctx, storage.RunFilter{Kind: "job"})
	assert.NoError(t, err)
	assert.Len(t, runs, 2)

	commands, err := store.Commands(ctx, storage.CommandFilter{})
	assert.NoError(t, err)
	assert.Len(t, commands, 4)
}

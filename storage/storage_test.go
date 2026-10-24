package storage_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
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

// sleepRun is a job that runs two sleeps, so the commands cost a second
// and two, and the job that holds them costs three.
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

	// A directory that belongs to no project still records, so the
	// report counts a stray command in /tmp too. With no project, the
	// workspace's is the one database and the run records once.
	paths, err := storage.Discover(t.TempDir())
	assert.NoError(t, err)
	assert.Equal(t, "", paths.Project)
	assert.Equal(t, filepath.Join(home, ".atkins", "atkins.db"), paths.Workspace)
	assert.Equal(t, []string{paths.Workspace}, paths.List())
	assert.DirExists(t, filepath.Join(home, ".atkins"))
}

// ATKINS_DB moves the workspace's database out of $HOME, as a runner
// that sets HOME per checkout does to keep one history.
func TestDiscoverEnvDB(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	file := filepath.Join(t.TempDir(), "stats", "atkins.db")
	t.Setenv(storage.EnvDB, file)

	paths, err := storage.Discover(t.TempDir())
	assert.NoError(t, err)
	assert.Equal(t, file, paths.Workspace)
	assert.DirExists(t, filepath.Dir(file))
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

// NewStorage opens the databases of the working directory, the
// project's first.
func TestNewStorage(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	project := t.TempDir()
	assert.NoError(t, os.MkdirAll(filepath.Join(project, ".atkins"), 0o755))
	t.Chdir(project)

	stores, err := storage.NewStorage()
	assert.NoError(t, err)
	assert.Len(t, stores, 2)
	assert.Equal(t, filepath.Join(project, ".atkins", "atkins.db"), stores[0].Path())
	assert.Equal(t, filepath.Join(home, ".atkins", "atkins.db"), stores[1].Path())
	for _, store := range stores {
		assert.NoError(t, store.Close())
	}
}

func TestStorage_RawStartedAt(t *testing.T) {
	ctx := context.Background()
	store := newStorage(t)

	_, err := store.RawStartedAt(ctx)
	assert.Error(t, err)

	_, err = store.Record(ctx, storage.RecordRequest{Log: sleepRun(base)})
	assert.NoError(t, err)

	raw, err := store.RawStartedAt(ctx)
	assert.NoError(t, err)
	assert.True(t, strings.HasPrefix(raw, "2026-03-01 12:00:0"), raw)
}

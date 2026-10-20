package storage

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	"github.com/jmoiron/sqlx"
	ulid "github.com/oklog/ulid/v2"
	"github.com/titpetric/platform"

	"github.com/titpetric/atkins/internal/assert"
	"github.com/titpetric/atkins/server/model"
	"github.com/titpetric/atkins/server/schema"
)

// connectionSeq hands each test a database of its own.
var connectionSeq atomic.Int64

// testStorage returns a migrated, throwaway database and a JobStorage
// over it.
//
// A file and not :memory:, because the platform opens sqlite with a
// connection pool, and every connection to an in-memory sqlite database
// is a separate, empty database.
func testStorage(t *testing.T) (*sqlx.DB, *JobStorage) {
	t.Helper()

	connection := "atkins_retention_" + strconv.Itoa(int(connectionSeq.Add(1)))
	dsn := "sqlite://file:" + filepath.Join(t.TempDir(), "atkins.db")
	t.Setenv("PLATFORM_DB_"+strings.ToUpper(connection), dsn)
	platform.SetupConnections(os.Environ())

	db, err := DB(t.Context(), connection)
	assert.NoError(t, err)
	assert.NoError(t, Migrate(t.Context(), db, schema.Migrations()))

	return db, NewJobStorage(db, nil, 0, 0)
}

// seedJob writes one job with the given status and finish time, plus
// logs chunks of output.
//
// It writes rows directly and skips Create and Finish because
// retention is about the passage of time, and the only way to have a
// job that finished a week ago is to say so.
func seedJob(t *testing.T, db *sqlx.DB, status model.JobStatus, finished time.Time, logs int) string {
	t.Helper()

	job := &model.Job{
		ID:           ulid.Make().String(),
		RepositoryID: "repository",
		UserID:       "user",
		Command:      "atkins build",
		Status:       status,
	}
	job.RootID = job.ID
	job.SetCreatedAt(finished)
	job.SetUpdatedAt(finished)
	if model.TerminalJobStatus(status) {
		job.SetStartedAt(finished)
		job.SetFinishedAt(finished)
	}

	assert.NoError(t, client(db).Insert(t.Context(), model.JobTable, job))

	for seq := range logs {
		entry := &model.JobLog{
			ID:      ulid.Make().String(),
			JobID:   job.ID,
			Seq:     int64(seq),
			Stream:  StreamOutput,
			Content: "line\n",
		}
		entry.SetCreatedAt(finished)
		assert.NoError(t, client(db).Insert(t.Context(), model.JobLogTable, entry))
	}

	return job.ID
}

// countRows counts the rows remaining in a table.
func countRows(t *testing.T, db *sqlx.DB, table string) int64 {
	t.Helper()

	result, err := client(db).Get[struct {
		Total int64 `db:"total"`
	}](t.Context(), `SELECT COUNT(*) AS total FROM `+table)
	assert.NoError(t, err)

	return result.Total
}

// countLogs counts the output rows left for one job.
func countLogs(t *testing.T, db *sqlx.DB, jobID string) int64 {
	t.Helper()

	result, err := client(db).Get[struct {
		Total int64 `db:"total"`
	}](t.Context(), `SELECT COUNT(*) AS total FROM `+model.JobLogTable+` WHERE job_id = ?`, jobID)
	assert.NoError(t, err)

	return result.Total
}

func TestPurgeDropsOutputAndKeepsTheOutcome(t *testing.T) {
	db, jobs := testStorage(t)

	now := time.Now()
	old := seedJob(t, db, model.JobStatusFailed, now.Add(-48*time.Hour), 3)
	recent := seedJob(t, db, model.JobStatusPassed, now.Add(-1*time.Hour), 2)

	result, err := jobs.Purge(t.Context(), RetentionRequest{Logs: 24 * time.Hour})
	assert.NoError(t, err)

	assert.Equal(t, int64(3), result.Logs)
	assert.Equal(t, int64(0), result.Jobs)
	assert.False(t, result.Partial)

	// The job that lost its output kept its outcome, which is the whole
	// argument for two windows.
	assert.Equal(t, int64(2), countRows(t, db, model.JobTable))
	assert.Equal(t, int64(0), countLogs(t, db, old))
	assert.Equal(t, int64(2), countLogs(t, db, recent))

	stored, err := jobs.Get(t.Context(), old)
	assert.NoError(t, err)
	assert.Equal(t, model.JobStatusFailed, stored.Status)
}

func TestPurgeDeletesJobsWithTheirOutput(t *testing.T) {
	db, jobs := testStorage(t)

	now := time.Now()
	old := seedJob(t, db, model.JobStatusPassed, now.Add(-48*time.Hour), 4)
	recent := seedJob(t, db, model.JobStatusPassed, now.Add(-1*time.Hour), 2)

	result, err := jobs.Purge(t.Context(), RetentionRequest{Jobs: 24 * time.Hour})
	assert.NoError(t, err)

	assert.Equal(t, int64(1), result.Jobs)
	assert.Equal(t, int64(4), result.Logs)

	// Output never outlives the job it belongs to: nothing would ever
	// read it again, and nothing would ever clean it up.
	assert.Equal(t, int64(1), countRows(t, db, model.JobTable))
	assert.Equal(t, int64(2), countRows(t, db, model.JobLogTable))
	assert.Equal(t, int64(0), countLogs(t, db, old))
	assert.Equal(t, int64(2), countLogs(t, db, recent))
}

func TestPurgeLeavesUnsettledJobsAlone(t *testing.T) {
	db, jobs := testStorage(t)

	// A job that has not settled waits and does not age, no matter
	// how old its queue time.
	ancient := time.Now().Add(-365 * 24 * time.Hour)
	seedJob(t, db, model.JobStatusPending, ancient, 1)
	seedJob(t, db, model.JobStatusRunning, ancient, 1)

	result, err := jobs.Purge(t.Context(), RetentionRequest{
		Jobs: time.Minute,
		Logs: time.Minute,
	})
	assert.NoError(t, err)

	assert.True(t, result.Empty())
	assert.Equal(t, int64(2), countRows(t, db, model.JobTable))
	assert.Equal(t, int64(2), countRows(t, db, model.JobLogTable))
}

func TestPurgeWithoutWindowsDeletesNothing(t *testing.T) {
	db, jobs := testStorage(t)

	seedJob(t, db, model.JobStatusPassed, time.Now().Add(-365*24*time.Hour), 2)

	// Both windows default to keeping everything forever, and a sweep
	// on a server that never configured one must be a no-op.
	result, err := jobs.Purge(t.Context(), RetentionRequest{})
	assert.NoError(t, err)

	assert.True(t, result.Empty())
	assert.Equal(t, int64(1), countRows(t, db, model.JobTable))
	assert.Equal(t, int64(2), countRows(t, db, model.JobLogTable))
}

func TestPurgeBatchesJobs(t *testing.T) {
	db, jobs := testStorage(t)

	expired := time.Now().Add(-48 * time.Hour)
	for range 25 {
		seedJob(t, db, model.JobStatusPassed, expired, 1)
	}

	request := RetentionRequest{Jobs: 24 * time.Hour, Batch: 5, MaxBatches: 2}

	// One pass removes Batch × MaxBatches rows and stops, so a first
	// sweep of a year-old instance is a series of small deletes rather
	// than one that holds the table for minutes.
	first, err := jobs.Purge(t.Context(), request)
	assert.NoError(t, err)
	assert.Equal(t, int64(10), first.Jobs)
	assert.Equal(t, int64(10), first.Logs)
	assert.True(t, first.Partial)
	assert.Equal(t, int64(15), countRows(t, db, model.JobTable))

	// Each further pass picks up where the last one stopped.
	var passes int
	for countRows(t, db, model.JobTable) > 0 {
		passes++
		assert.Less(t, passes, 10, "retention is not making progress")

		_, err := jobs.Purge(t.Context(), request)
		assert.NoError(t, err)
	}

	assert.Equal(t, int64(0), countRows(t, db, model.JobLogTable))

	// And the pass that finishes the backlog says so.
	last, err := jobs.Purge(t.Context(), request)
	assert.NoError(t, err)
	assert.True(t, last.Empty())
	assert.False(t, last.Partial)
}

func TestPurgeBatchesLogs(t *testing.T) {
	db, jobs := testStorage(t)

	// One job with a lot of output is the shape that hurts: job_log is
	// where an instance grows, not job.
	job := seedJob(t, db, model.JobStatusFailed, time.Now().Add(-48*time.Hour), 25)

	request := RetentionRequest{Logs: 24 * time.Hour, Batch: 5, MaxBatches: 2}

	first, err := jobs.Purge(t.Context(), request)
	assert.NoError(t, err)
	assert.Equal(t, int64(10), first.Logs)
	assert.True(t, first.Partial)
	assert.Equal(t, int64(15), countLogs(t, db, job))

	// purgeJobLogs takes the batch from job_log, and not from job, so a
	// second pass over a job whose output is partly gone still makes
	// progress and never re-reads the same job forever.
	second, err := jobs.Purge(t.Context(), request)
	assert.NoError(t, err)
	assert.Equal(t, int64(10), second.Logs)
	assert.Equal(t, int64(5), countLogs(t, db, job))

	third, err := jobs.Purge(t.Context(), request)
	assert.NoError(t, err)
	assert.Equal(t, int64(5), third.Logs)
	assert.False(t, third.Partial)

	// The job itself stays intact throughout.
	assert.Equal(t, int64(1), countRows(t, db, model.JobTable))
}

func TestVisibleTo(t *testing.T) {
	db, jobs := testStorage(t)

	root := seedJob(t, db, model.JobStatusPassed, time.Now(), 0)

	// A child dispatched from inside a running pipeline belongs to the
	// agent's account, not to the human who started the run. That human
	// can still read it.
	child := &model.Job{
		ID:           ulid.Make().String(),
		ParentID:     root,
		RootID:       root,
		Depth:        1,
		RepositoryID: "repository",
		UserID:       "agent",
		Command:      "atkins analyzeTag",
		Status:       model.JobStatusPassed,
	}
	child.SetCreatedAt(time.Now())
	assert.NoError(t, client(db).Insert(t.Context(), model.JobTable, child))

	stored, err := jobs.Get(t.Context(), root)
	assert.NoError(t, err)

	visible, err := jobs.VisibleTo(t.Context(), stored, "user")
	assert.NoError(t, err)
	assert.True(t, visible)

	visible, err = jobs.VisibleTo(t.Context(), stored, "stranger")
	assert.NoError(t, err)
	assert.False(t, visible)

	visible, err = jobs.VisibleTo(t.Context(), child, "user")
	assert.NoError(t, err)
	assert.True(t, visible, "a user must see the jobs their own run dispatched")

	visible, err = jobs.VisibleTo(t.Context(), child, "stranger")
	assert.NoError(t, err)
	assert.False(t, visible)

	// Nothing is visible to nobody.
	visible, err = jobs.VisibleTo(t.Context(), stored, "")
	assert.NoError(t, err)
	assert.False(t, visible)
}

func TestPlaceholders(t *testing.T) {
	assert.Equal(t, "NULL", placeholders(0))
	assert.Equal(t, "?", placeholders(1))
	assert.Equal(t, "?,?,?", placeholders(3))
}

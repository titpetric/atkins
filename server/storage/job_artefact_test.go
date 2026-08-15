package storage_test

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	_ "github.com/titpetric/platform/pkg/drivers"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/titpetric/platform"

	"github.com/titpetric/atkins/server/blob"
	"github.com/titpetric/atkins/server/model"
	"github.com/titpetric/atkins/server/schema"
	"github.com/titpetric/atkins/server/storage"
)

// connectionSeq hands each test a unique database connection name.
var connectionSeq atomic.Int64

// testStorage returns artefact storage over a throwaway database and a
// blob root the test owns.
func testStorage(t *testing.T) (*storage.JobArtefactStorage, *blob.Dir, *sqlx.DB) {
	t.Helper()

	// The platform caches pools by connection name for the life of the
	// process, so each test claims a name of its own.
	connection := "atkins_storage_test_" + strconv.Itoa(int(connectionSeq.Add(1)))
	dsn := "sqlite://file:" + filepath.Join(t.TempDir(), "atkins.db")
	t.Setenv("PLATFORM_DB_"+strings.ToUpper(connection), dsn)
	platform.SetupConnections(os.Environ())

	ctx := t.Context()

	db, err := storage.DB(ctx, connection)
	require.NoError(t, err)
	require.NoError(t, storage.Migrate(ctx, db, schema.Migrations()))

	blobs := blob.NewDir(filepath.Join(t.TempDir(), "artefacts"))
	require.NoError(t, blobs.Prepare())

	return storage.NewJobArtefactStorage(db, blobs), blobs, db
}

// store writes one artefact for a job.
func store(t *testing.T, artefacts *storage.JobArtefactStorage, jobID, path, content string) *model.JobArtefact {
	t.Helper()

	artefact, err := artefacts.Create(t.Context(), storage.ArtefactRequest{
		JobID:    jobID,
		AgentID:  "agent-1",
		Path:     path,
		Content:  strings.NewReader(content),
		MaxSize:  1024,
		MaxCount: 10,
	})
	require.NoError(t, err)

	return artefact
}

// contents reads an artefact back.
func contents(t *testing.T, artefacts *storage.JobArtefactStorage, artefact *model.JobArtefact) string {
	t.Helper()

	reader, err := artefacts.Open(t.Context(), artefact)
	require.NoError(t, err)
	defer reader.Close()

	body, err := io.ReadAll(reader)
	require.NoError(t, err)

	return string(body)
}

func TestCreateStoresRowAndBytes(t *testing.T) {
	artefacts, blobs, _ := testStorage(t)

	artefact := store(t, artefacts, "job-1", "reports/scan.json", `{"findings":3}`)

	assert.Equal(t, int64(14), artefact.Size)
	assert.Len(t, artefact.Checksum, 64)
	// The row points at the bytes rather than holding them, which is
	// what keeps a swap to an object store to one interface.
	assert.Equal(t, blob.Key("job-1", artefact.ID), artefact.StorageKey)
	assert.FileExists(t, filepath.Join(blobs.Root(), "job-1", artefact.ID))

	assert.Equal(t, `{"findings":3}`, contents(t, artefacts, artefact))

	listed, err := artefacts.List(t.Context(), "job-1")
	require.NoError(t, err)
	require.Len(t, listed, 1)
	assert.Equal(t, artefact.ID, listed[0].ID)
}

func TestCreateRejectsAPathOutsideTheJob(t *testing.T) {
	artefacts, blobs, _ := testStorage(t)

	for _, path := range []string{"../../etc/passwd", "/etc/passwd", "", "   "} {
		_, err := artefacts.Create(t.Context(), storage.ArtefactRequest{
			JobID:   "job-1",
			Path:    path,
			Content: strings.NewReader("planted"),
			MaxSize: 1024,
		})
		assert.ErrorIs(t, err, model.ErrInvalidArtefactPath, "path %q", path)
	}

	// The path is checked before anything is written, so a refused
	// upload costs no disk at all.
	assert.NoDirExists(t, filepath.Join(blobs.Root(), "job-1"))
}

func TestCreateEnforcesTheSizeLimit(t *testing.T) {
	artefacts, blobs, _ := testStorage(t)

	_, err := artefacts.Create(t.Context(), storage.ArtefactRequest{
		JobID:   "job-1",
		Path:    "core.dump",
		Content: strings.NewReader(strings.Repeat("x", 65)),
		MaxSize: 64,
	})
	assert.ErrorIs(t, err, model.ErrArtefactTooLarge)

	// No row, and no half-written file: the two never disagree.
	listed, err := artefacts.List(t.Context(), "job-1")
	require.NoError(t, err)
	assert.Empty(t, listed)

	entries, err := os.ReadDir(filepath.Join(blobs.Root(), "job-1"))
	if err == nil {
		assert.Empty(t, entries)
	}
}

func TestCreateEnforcesTheCountLimit(t *testing.T) {
	artefacts, _, _ := testStorage(t)

	for _, path := range []string{"one.json", "two.json"} {
		_, err := artefacts.Create(t.Context(), storage.ArtefactRequest{
			JobID:    "job-1",
			Path:     path,
			Content:  strings.NewReader("{}"),
			MaxSize:  1024,
			MaxCount: 2,
		})
		require.NoError(t, err, path)
	}

	_, err := artefacts.Create(t.Context(), storage.ArtefactRequest{
		JobID:    "job-1",
		Path:     "three.json",
		Content:  strings.NewReader("{}"),
		MaxSize:  1024,
		MaxCount: 2,
	})
	assert.ErrorIs(t, err, model.ErrTooManyArtefacts)

	// A different job has its own allowance.
	_, err = artefacts.Create(t.Context(), storage.ArtefactRequest{
		JobID:    "job-2",
		Path:     "one.json",
		Content:  strings.NewReader("{}"),
		MaxSize:  1024,
		MaxCount: 2,
	})
	assert.NoError(t, err)
}

func TestCreateVerifiesTheChecksum(t *testing.T) {
	artefacts, blobs, _ := testStorage(t)

	_, err := artefacts.Create(t.Context(), storage.ArtefactRequest{
		JobID:    "job-1",
		Path:     "scan.json",
		Checksum: strings.Repeat("0", 64),
		Content:  strings.NewReader("{}"),
		MaxSize:  1024,
	})
	assert.ErrorIs(t, err, model.ErrChecksumMismatch)

	// The bytes that failed the comparison are dropped rather than left
	// for a later upload to trip over.
	entries, err := os.ReadDir(filepath.Join(blobs.Root(), "job-1"))
	if err == nil {
		assert.Empty(t, entries)
	}
}

func TestCreateReplacesTheSamePath(t *testing.T) {
	artefacts, blobs, _ := testStorage(t)

	first := store(t, artefacts, "job-1", "scan.json", `{"run":1}`)
	second := store(t, artefacts, "job-1", "scan.json", `{"run":2}`)

	listed, err := artefacts.List(t.Context(), "job-1")
	require.NoError(t, err)
	require.Len(t, listed, 1)
	assert.Equal(t, second.ID, listed[0].ID)

	// The superseded bytes are gone, not orphaned on the disk.
	assert.NoFileExists(t, filepath.Join(blobs.Root(), "job-1", first.ID))

	_, err = artefacts.Get(t.Context(), "job-1", first.ID)
	assert.ErrorIs(t, err, model.ErrArtefactNotFound)
}

func TestGetIsScopedToTheJob(t *testing.T) {
	artefacts, _, _ := testStorage(t)

	artefact := store(t, artefacts, "job-1", "scan.json", "{}")

	found, err := artefacts.Get(t.Context(), "job-1", artefact.ID)
	require.NoError(t, err)
	assert.Equal(t, artefact.ID, found.ID)

	// An ID from another job does not resolve here.
	_, err = artefacts.Get(t.Context(), "job-2", artefact.ID)
	assert.ErrorIs(t, err, model.ErrArtefactNotFound)
}

func TestOpenReportsSweptBytes(t *testing.T) {
	artefacts, blobs, _ := testStorage(t)

	artefact := store(t, artefacts, "job-1", "scan.json", "{}")
	require.NoError(t, blobs.Remove(context.Background(), artefact.StorageKey))

	// A row whose file an operator removed reads as a missing artefact,
	// not as a server error.
	_, err := artefacts.Open(t.Context(), artefact)
	assert.ErrorIs(t, err, model.ErrArtefactNotFound)
}

func TestPruneExpired(t *testing.T) {
	artefacts, blobs, db := testStorage(t)

	old := store(t, artefacts, "job-1", "old.json", "{}")
	fresh := store(t, artefacts, "job-1", "fresh.json", "{}")

	// Age one of them past the cutoff.
	backdated := time.Now().Add(-2 * time.Hour)
	_, err := db.ExecContext(t.Context(),
		"UPDATE job_artefact SET created_at = ? WHERE id = ?", backdated, old.ID)
	require.NoError(t, err)

	swept, err := artefacts.PruneExpired(t.Context(), time.Now().Add(-time.Hour))
	require.NoError(t, err)
	assert.Equal(t, int64(1), swept)

	listed, err := artefacts.List(t.Context(), "job-1")
	require.NoError(t, err)
	require.Len(t, listed, 1)
	assert.Equal(t, fresh.ID, listed[0].ID)

	// Retention is about the disk.
	assert.NoFileExists(t, filepath.Join(blobs.Root(), "job-1", old.ID))
	assert.FileExists(t, filepath.Join(blobs.Root(), "job-1", fresh.ID))

	// The row survives its bytes, so the page can still say the file
	// existed and how big it was.
	var count int
	require.NoError(t, db.GetContext(t.Context(), &count,
		"SELECT COUNT(*) FROM job_artefact WHERE id = ? AND deleted_at IS NOT NULL", old.ID))
	assert.Equal(t, 1, count)

	// A second pass has nothing left to do.
	swept, err = artefacts.PruneExpired(t.Context(), time.Now().Add(-time.Hour))
	require.NoError(t, err)
	assert.Equal(t, int64(0), swept)
}

func TestCountIgnoresSweptArtefacts(t *testing.T) {
	artefacts, _, db := testStorage(t)

	store(t, artefacts, "job-1", "one.json", "{}")
	store(t, artefacts, "job-1", "two.json", "{}")

	count, err := artefacts.Count(t.Context(), "job-1")
	require.NoError(t, err)
	assert.Equal(t, int64(2), count)

	_, err = db.ExecContext(t.Context(),
		"UPDATE job_artefact SET created_at = ? WHERE path = ?", time.Now().Add(-2*time.Hour), "one.json")
	require.NoError(t, err)

	_, err = artefacts.PruneExpired(t.Context(), time.Now().Add(-time.Hour))
	require.NoError(t, err)

	// A job whose artefacts were swept can produce new ones: the limit
	// counts what is on the disk, not what ever was.
	count, err = artefacts.Count(t.Context(), "job-1")
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)
}

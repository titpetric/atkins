package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/titpetric/platform/pkg/telemetry"
	"github.com/titpetric/platform/pkg/ulid"

	"github.com/titpetric/atkins/server/blob"
	"github.com/titpetric/atkins/server/model"
)

// JobArtefactStorage persists the files a job produced.
//
// It owns both halves of an artefact: the row that describes it and the
// bytes it points at. Keeping them behind one type is what stops the
// two from drifting — a caller cannot insert a row for bytes that were
// never written, or delete a row and forget the file.
type JobArtefactStorage struct {
	db    *sqlx.DB
	blobs blob.Store
}

// NewJobArtefactStorage returns a JobArtefactStorage backed by the
// given pool and blob store.
func NewJobArtefactStorage(db *sqlx.DB, blobs blob.Store) *JobArtefactStorage {
	return &JobArtefactStorage{db: db, blobs: blobs}
}

// ArtefactRequest is one uploaded file.
type ArtefactRequest struct {
	// JobID is the job that produced the file. The caller has already
	// established that it exists.
	JobID string

	// AgentID records which worker uploaded it.
	AgentID string

	// Path is the name the pipeline gave the file, relative to the
	// directory the job ran in.
	Path string

	// ContentType is what the agent thought it was uploading.
	ContentType string

	// Checksum, when set, is the SHA256 the agent computed. It is
	// compared against what actually arrived.
	Checksum string

	// Content is the file. It is streamed to the blob store rather than
	// read into memory: an artefact is as large as the limit allows.
	Content io.Reader

	// MaxSize bounds this upload; MaxCount bounds how many artefacts
	// the job may keep. Both come from the setting registry, so an
	// admin can change them without a restart.
	MaxSize  int64
	MaxCount int64
}

// Create stores an uploaded artefact.
//
// Uploading a path a job already has replaces it. A collection that
// runs twice — a retried upload, a pipeline that copies the same file
// from two steps — should leave one scan.json rather than two, and the
// count limit should not be spent on duplicates.
func (s *JobArtefactStorage) Create(ctx context.Context, req ArtefactRequest) (*model.JobArtefact, error) {
	ctx, span := telemetry.StartAuto(ctx, s.Create)
	defer span.End()

	path := model.ArtefactPath(req.Path)
	if path == "" {
		return nil, model.ErrInvalidArtefactPath
	}

	superseded, err := s.byPath(ctx, req.JobID, path)
	if err != nil {
		return nil, err
	}

	// A replacement takes the slot of the file it replaces, so a job
	// re-uploading its one artefact never runs into the count limit.
	if superseded == nil && req.MaxCount > 0 {
		count, err := s.Count(ctx, req.JobID)
		if err != nil {
			return nil, err
		}
		if count >= req.MaxCount {
			return nil, model.ErrTooManyArtefacts
		}
	}

	id := ulid.String()
	key := blob.Key(req.JobID, id)

	written, err := s.blobs.Put(ctx, key, req.Content, req.MaxSize)
	if err != nil {
		if errors.Is(err, blob.ErrTooLarge) {
			return nil, model.ErrArtefactTooLarge
		}
		return nil, fmt.Errorf("store artefact: %w", err)
	}

	// Hex, so case is presentation rather than meaning.
	if req.Checksum != "" && !strings.EqualFold(req.Checksum, written.Checksum) {
		_ = s.blobs.Remove(ctx, key)
		return nil, model.ErrChecksumMismatch
	}

	artefact := &model.JobArtefact{
		ID:          id,
		JobID:       req.JobID,
		Path:        path,
		StorageKey:  key,
		Size:        written.Size,
		ContentType: model.ArtefactContentType(req.ContentType),
		Checksum:    written.Checksum,
		AgentID:     req.AgentID,
	}
	artefact.SetCreatedAt(time.Now())

	if err := client(s.db).Insert(ctx, model.JobArtefactTable, artefact); err != nil {
		// Leave no bytes nobody can reach.
		_ = s.blobs.Remove(ctx, key)
		return nil, fmt.Errorf("create artefact: %w", err)
	}

	if superseded != nil {
		// Only now that the replacement is durable.
		if err := s.remove(ctx, superseded); err != nil {
			return nil, err
		}
	}

	return artefact, nil
}

// List returns the artefacts of a job whose bytes are still there.
func (s *JobArtefactStorage) List(ctx context.Context, jobID string) ([]model.JobArtefact, error) {
	ctx, span := telemetry.StartAuto(ctx, s.List)
	defer span.End()

	query := `SELECT * FROM ` + model.JobArtefactTable + ` WHERE job_id = ? AND deleted_at IS NULL ORDER BY path ASC`
	return client(s.db).Select[model.JobArtefact](ctx, query, jobID)
}

// Get returns one artefact of one job.
//
// The job is part of the lookup rather than checked afterwards: an
// artefact ID from another job must not resolve here just because the
// caller could read some job.
func (s *JobArtefactStorage) Get(ctx context.Context, jobID, artefactID string) (*model.JobArtefact, error) {
	query := `SELECT * FROM ` + model.JobArtefactTable + ` WHERE id = ? AND job_id = ? AND deleted_at IS NULL`

	artefact, err := client(s.db).Get[model.JobArtefact](ctx, query, artefactID, jobID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, model.ErrArtefactNotFound
		}
		return nil, err
	}
	return artefact, nil
}

// Open returns the bytes of an artefact.
func (s *JobArtefactStorage) Open(ctx context.Context, artefact *model.JobArtefact) (io.ReadCloser, error) {
	contents, err := s.blobs.Open(ctx, artefact.StorageKey)
	if err != nil {
		// The row survives its bytes: retention sweeps them, and so
		// does an operator with a full disk. Either way this is a 404
		// rather than a 500.
		return nil, fmt.Errorf("%w: %s", model.ErrArtefactNotFound, artefact.Path)
	}
	return contents, nil
}

// Count returns how many artefacts a job currently keeps.
func (s *JobArtefactStorage) Count(ctx context.Context, jobID string) (int64, error) {
	query := `SELECT COUNT(*) AS count FROM ` + model.JobArtefactTable + ` WHERE job_id = ? AND deleted_at IS NULL`

	result, err := client(s.db).Get[struct {
		Count int64 `db:"count"`
	}](ctx, query, jobID)
	if err != nil {
		return 0, fmt.Errorf("count artefacts: %w", err)
	}
	return result.Count, nil
}

// PruneExpired drops the bytes of artefacts older than the cutoff and
// reports how many it swept.
//
// The row is soft deleted rather than removed. It costs a few dozen
// bytes, it keeps the record that a 40MB scan.json was produced, and it
// makes the removal auditable: "swept by retention" and "never uploaded"
// are different answers to why a file isn't there.
func (s *JobArtefactStorage) PruneExpired(ctx context.Context, cutoff time.Time) (int64, error) {
	ctx, span := telemetry.StartAuto(ctx, s.PruneExpired)
	defer span.End()

	query := `SELECT * FROM ` + model.JobArtefactTable + `
		WHERE deleted_at IS NULL AND created_at IS NOT NULL AND created_at < ?
		ORDER BY created_at ASC LIMIT ?`
	expired, err := client(s.db).Select[model.JobArtefact](ctx, query, cutoff, pruneBatch)
	if err != nil {
		return 0, fmt.Errorf("select expired artefacts: %w", err)
	}

	var swept int64
	for _, artefact := range expired {
		if err := s.remove(ctx, &artefact); err != nil {
			return swept, err
		}
		swept++
	}

	return swept, nil
}

// pruneBatch bounds one retention sweep. The sweep runs on a ticker, so
// a backlog drains over several passes rather than holding the database
// and the disk for one very long delete.
const pruneBatch = 500

// byPath returns the live artefact a job already has under a path.
func (s *JobArtefactStorage) byPath(ctx context.Context, jobID, path string) (*model.JobArtefact, error) {
	query := `SELECT * FROM ` + model.JobArtefactTable + ` WHERE job_id = ? AND path = ? AND deleted_at IS NULL`

	artefact, err := client(s.db).Get[model.JobArtefact](ctx, query, jobID, path)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("look up artefact: %w", err)
	}
	return artefact, nil
}

// remove drops an artefact's bytes and marks the row deleted.
//
// Bytes first: a row that outlives its file reads as "swept", while a
// file that outlives its row is unreachable garbage nothing will ever
// clean up.
func (s *JobArtefactStorage) remove(ctx context.Context, artefact *model.JobArtefact) error {
	if err := s.blobs.Remove(ctx, artefact.StorageKey); err != nil {
		return fmt.Errorf("remove artefact bytes: %w", err)
	}

	query := `UPDATE ` + model.JobArtefactTable + ` SET deleted_at = ? WHERE id = ?`
	if err := client(s.db).Exec(ctx, query, time.Now(), artefact.ID); err != nil {
		return fmt.Errorf("delete artefact: %w", err)
	}
	return nil
}

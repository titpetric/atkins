package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/titpetric/platform/pkg/telemetry"
	"github.com/titpetric/platform/pkg/ulid"

	"github.com/titpetric/atkins/server/model"
)

// JobLogStorage persists output captured from a job.
type JobLogStorage struct {
	db *sqlx.DB
}

// NewJobLogStorage returns a JobLogStorage backed by the given pool.
func NewJobLogStorage(db *sqlx.DB) *JobLogStorage {
	return &JobLogStorage{db: db}
}

// Stream names for job output.
const (
	// StreamOutput is combined stdout and stderr from the command.
	StreamOutput = "output"

	// StreamError is the agent's own commentary: clone failures,
	// timeouts, anything that happened around the command rather than
	// inside it.
	StreamError = "error"
)

// MaxLogChunk bounds one appended chunk. A job that prints a gigabyte
// should fill the page slowly rather than the database quickly.
const MaxLogChunk = 256 * 1024

// Append adds a chunk of output to a job.
//
// Chunks are numbered per job so the page can render them in the order
// the agent produced them, without relying on timestamp resolution.
func (s *JobLogStorage) Append(ctx context.Context, jobID, stream, content string) error {
	ctx, span := telemetry.StartAuto(ctx, s.Append)
	defer span.End()

	if stream != StreamError {
		stream = StreamOutput
	}
	if len(content) > MaxLogChunk {
		content = content[:MaxLogChunk] + "\n... output truncated ...\n"
	}

	next, err := s.nextSeq(ctx, jobID)
	if err != nil {
		return err
	}

	entry := &model.JobLog{
		ID:      ulid.String(),
		JobID:   jobID,
		Seq:     next,
		Stream:  stream,
		Content: content,
	}
	entry.SetCreatedAt(time.Now())

	if err := client(s.db).Insert(ctx, model.JobLogTable, entry); err != nil {
		return fmt.Errorf("append job log: %w", err)
	}
	return nil
}

// List returns the log chunks for a job, in the order they arrived.
func (s *JobLogStorage) List(ctx context.Context, jobID string) ([]model.JobLog, error) {
	ctx, span := telemetry.StartAuto(ctx, s.List)
	defer span.End()

	query := `SELECT * FROM ` + model.JobLogTable + ` WHERE job_id = ? ORDER BY seq ASC`
	return client(s.db).Select[model.JobLog](ctx, query, jobID)
}

// nextSeq returns the next chunk number for a job.
func (s *JobLogStorage) nextSeq(ctx context.Context, jobID string) (int64, error) {
	query := `SELECT COALESCE(MAX(seq), -1) + 1 AS seq FROM ` + model.JobLogTable + ` WHERE job_id = ?`
	result, err := client(s.db).Get[struct {
		Seq int64 `db:"seq"`
	}](ctx, query, jobID)
	if err != nil {
		return 0, fmt.Errorf("next log sequence: %w", err)
	}
	return result.Seq, nil
}

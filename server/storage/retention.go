package storage

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/titpetric/oida"

	"github.com/titpetric/atkins/server/model"
)

// Retention defaults. They bound one pass rather than the whole
// backlog: the first sweep of an instance that has been running for a
// year has millions of job_log rows to remove, and a DELETE that large
// holds locks for minutes. A pass removes at most Batch × MaxBatches
// rows of each kind and leaves the rest for the next tick, so a server
// catching up stays a server.
const (
	// DefaultRetentionBatch is how many rows one DELETE removes.
	DefaultRetentionBatch = 500

	// DefaultRetentionBatches is how many batches one pass runs.
	DefaultRetentionBatches = 20
)

// RetentionRequest is one retention pass.
//
// The two windows are independent on purpose. Output is what grows
// without bound, and it stops being interesting long before the outcome
// does; an instance that keeps every job forever and every log line for
// a month is the common case.
type RetentionRequest struct {
	// Jobs is how long a settled job's record is kept, measured from
	// when it finished. Zero keeps job records forever.
	Jobs time.Duration

	// Logs is how long a settled job's captured output is kept. Zero
	// keeps output forever.
	Logs time.Duration

	// Batch is how many rows one statement removes. Zero selects
	// DefaultRetentionBatch.
	Batch int

	// MaxBatches bounds one pass. Zero selects DefaultRetentionBatches.
	MaxBatches int
}

// RetentionResult reports what a pass removed.
type RetentionResult struct {
	// Jobs is how many job records were deleted.
	Jobs int64

	// Logs is how many output rows were deleted, including those that
	// went with a deleted job.
	Logs int64

	// Partial is set when the pass stopped at MaxBatches with work
	// still to do. The next tick continues where this one stopped.
	Partial bool
}

// Empty reports whether the pass deleted nothing at all.
func (r RetentionResult) Empty() bool {
	return r.Jobs == 0 && r.Logs == 0
}

// withDefaults fills the bounds a caller left at zero.
func (r RetentionRequest) withDefaults() RetentionRequest {
	if r.Batch <= 0 {
		r.Batch = DefaultRetentionBatch
	}
	if r.MaxBatches <= 0 {
		r.MaxBatches = DefaultRetentionBatches
	}
	return r
}

// Purge applies the retention windows and reports what it removed.
//
// Job records go first, because deleting a job takes its output with it
// and there is no point sweeping logs that are about to disappear along
// with their job. Neither window touches a job that has not settled: a
// pending job is not old, it is waiting, and a running job that has
// lost its agent is the lease sweep's problem, not this one's.
func (s *JobStorage) Purge(ctx context.Context, req RetentionRequest) (RetentionResult, error) {
	ctx, span := oida.StartAuto(ctx, s.Purge, oida.KindDatabase)
	defer span.End()

	req = req.withDefaults()
	now := time.Now()

	var result RetentionResult

	if req.Jobs > 0 {
		jobs, logs, partial, err := s.purgeJobs(ctx, now.Add(-req.Jobs), req)
		result.Jobs += jobs
		result.Logs += logs
		result.Partial = result.Partial || partial
		if err != nil {
			return result, err
		}
	}

	if req.Logs > 0 {
		logs, partial, err := s.purgeJobLogs(ctx, now.Add(-req.Logs), req)
		result.Logs += logs
		result.Partial = result.Partial || partial
		if err != nil {
			return result, err
		}
	}

	return result, nil
}

// purgeJobs deletes settled jobs that finished before the cutoff, with
// their output, in batches.
//
// Progress is guaranteed: every selected row is deleted, so the next
// batch selects rows the previous one did not see.
func (s *JobStorage) purgeJobs(ctx context.Context, cutoff time.Time, req RetentionRequest) (int64, int64, bool, error) {
	var jobs, logs int64

	for range req.MaxBatches {
		ids, err := s.settledJobIDs(ctx, cutoff, req.Batch)
		if err != nil {
			return jobs, logs, false, err
		}
		if len(ids) == 0 {
			return jobs, logs, false, nil
		}

		removed, err := s.deleteJobs(ctx, ids)
		if err != nil {
			return jobs, logs, false, err
		}

		jobs += int64(len(ids))
		logs += removed

		// A short batch means the cutoff has been reached.
		if len(ids) < req.Batch {
			return jobs, logs, false, nil
		}
	}

	return jobs, logs, true, nil
}

// purgeJobLogs deletes the output of settled jobs that finished before
// the cutoff, leaving the job records themselves alone.
//
// The batch is selected from job_log rather than from job so that every
// pass makes progress. Selecting jobs and deleting their logs would
// keep re-reading the same jobs once their output was gone, and an
// instance with a large backlog would never reach the end of it.
func (s *JobStorage) purgeJobLogs(ctx context.Context, cutoff time.Time, req RetentionRequest) (int64, bool, error) {
	var deleted int64

	for range req.MaxBatches {
		ids, err := s.expiredLogIDs(ctx, cutoff, req.Batch)
		if err != nil {
			return deleted, false, err
		}
		if len(ids) == 0 {
			return deleted, false, nil
		}

		query := `DELETE FROM ` + model.JobLogTable + ` WHERE id IN (` + placeholders(len(ids)) + `)`
		if err := client(s.db).Exec(ctx, query, arguments(ids)...); err != nil {
			return deleted, false, fmt.Errorf("delete job logs: %w", err)
		}

		deleted += int64(len(ids))

		if len(ids) < req.Batch {
			return deleted, false, nil
		}
	}

	return deleted, true, nil
}

// idRow scans a bare id column.
type idRow struct {
	ID string `db:"id"`
}

// settledJobIDs returns up to limit settled jobs that finished before
// the cutoff, oldest first.
func (s *JobStorage) settledJobIDs(ctx context.Context, cutoff time.Time, limit int) ([]string, error) {
	terminal := model.TerminalJobStatuses()

	query := `SELECT id FROM ` + model.JobTable + `
		WHERE status IN (` + placeholders(len(terminal)) + `)
		  AND finished_at IS NOT NULL AND finished_at < ?
		ORDER BY finished_at ASC LIMIT ?`

	args := append(arguments(terminal), cutoff, limit)
	rows, err := client(s.db).Select[idRow](ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("select expired jobs: %w", err)
	}

	return identifiers(rows), nil
}

// expiredLogIDs returns up to limit output rows belonging to settled
// jobs that finished before the cutoff.
func (s *JobStorage) expiredLogIDs(ctx context.Context, cutoff time.Time, limit int) ([]string, error) {
	terminal := model.TerminalJobStatuses()

	query := `SELECT ` + model.JobLogTable + `.id FROM ` + model.JobLogTable + `
		INNER JOIN ` + model.JobTable + ` ON ` + model.JobTable + `.id = ` + model.JobLogTable + `.job_id
		WHERE ` + model.JobTable + `.status IN (` + placeholders(len(terminal)) + `)
		  AND ` + model.JobTable + `.finished_at IS NOT NULL
		  AND ` + model.JobTable + `.finished_at < ?
		LIMIT ?`

	args := append(arguments(terminal), cutoff, limit)
	rows, err := client(s.db).Select[idRow](ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("select expired job logs: %w", err)
	}

	return identifiers(rows), nil
}

// deleteJobs removes jobs and their output, and reports how many output
// rows went with them.
//
// One transaction per batch: a job record that outlives its output is a
// page that renders, and output that outlives its job is a row nothing
// will ever read again or clean up.
func (s *JobStorage) deleteJobs(ctx context.Context, ids []string) (int64, error) {
	db := client(s.db)
	if err := db.Begin(ctx); err != nil {
		return 0, fmt.Errorf("begin job purge: %w", err)
	}
	defer db.Rollback()

	args := arguments(ids)
	list := placeholders(len(ids))

	if err := db.Exec(ctx, `DELETE FROM `+model.JobLogTable+` WHERE job_id IN (`+list+`)`, args...); err != nil {
		return 0, fmt.Errorf("delete job output: %w", err)
	}
	logs := db.RowsAffected()

	if err := db.Exec(ctx, `DELETE FROM `+model.JobTable+` WHERE id IN (`+list+`)`, args...); err != nil {
		return 0, fmt.Errorf("delete jobs: %w", err)
	}

	if err := db.Commit(); err != nil {
		return 0, fmt.Errorf("commit job purge: %w", err)
	}

	return logs, nil
}

// placeholders renders an IN list of n bind parameters.
func placeholders(n int) string {
	if n <= 0 {
		return "NULL"
	}
	return strings.TrimSuffix(strings.Repeat("?,", n), ",")
}

// arguments widens a string slice into query arguments.
func arguments(values []string) []any {
	args := make([]any, 0, len(values))
	for _, value := range values {
		args = append(args, value)
	}
	return args
}

// identifiers pulls the ids out of scanned rows.
func identifiers(rows []idRow) []string {
	ids := make([]string, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ID)
	}
	return ids
}

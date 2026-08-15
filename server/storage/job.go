package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/titpetric/platform/pkg/telemetry"
	"github.com/titpetric/platform/pkg/ulid"

	"github.com/titpetric/atkins/server/model"
)

// defaultListLimit caps an unbounded listing.
const defaultListLimit = 100

// JobStorage persists dispatched jobs and their lifecycle.
type JobStorage struct {
	db *sqlx.DB

	// maxDepth bounds how deep a job tree may nest. A pipeline that
	// dispatches a child that dispatches a child is useful; one that
	// does so without a floor is a fork bomb with a queue in front.
	maxDepth int64

	// leaseTTL is how long an agent holds a claimed job before the
	// server considers it dead and reclaims the job.
	leaseTTL time.Duration
}

// Defaults applied when the corresponding JobStorage fields are zero.
const (
	// DefaultMaxDepth allows a dispatcher job to fan out to children
	// and those children to fan out once more.
	DefaultMaxDepth = 3

	// DefaultLeaseTTL is how long a claimed job stays claimed without
	// a heartbeat or a terminal report.
	DefaultLeaseTTL = 15 * time.Minute
)

// NewJobStorage returns a JobStorage backed by the given pool.
// Zero values for maxDepth and leaseTTL select the package defaults.
func NewJobStorage(db *sqlx.DB, maxDepth int64, leaseTTL time.Duration) *JobStorage {
	if maxDepth <= 0 {
		maxDepth = DefaultMaxDepth
	}
	if leaseTTL <= 0 {
		leaseTTL = DefaultLeaseTTL
	}
	return &JobStorage{db: db, maxDepth: maxDepth, leaseTTL: leaseTTL}
}

// JobRequest is the input for dispatching a job.
type JobRequest struct {
	// ParentID is the dispatching job, when a running pipeline queues
	// work of its own. Empty for a root job.
	ParentID string

	RepositoryID string
	UserID       string

	// WorkingDirectory is relative to the repository root. It is where
	// the agent runs the command, and it is what lets one repository
	// carry many independently dispatched projects.
	WorkingDirectory string

	// Command is the atkins invocation to run, verbatim.
	Command string

	// Ref is what to check out: a branch, a tag, a commit sha or a
	// fully qualified refname. Empty means the repository's default
	// branch, resolved by the agent when the job runs rather than here,
	// so a nightly trigger follows the branch it is pointed at.
	Ref string

	// CloneDepth limits the history of the job's work tree. 0 is the
	// whole history.
	CloneDepth int64

	Labels []string

	// Params is a JSON object handed to the job as ATKINS_JOB_PARAMS.
	Params string

	// Artefacts are glob patterns the agent collects after the command
	// exits, relative to the directory it ran in.
	Artefacts []string
}

// Create records a dispatched job.
//
// Depth and root are derived from the parent rather than trusted from
// the client: a job that lies about its depth would sidestep the
// runaway-dispatch limit.
func (s *JobStorage) Create(ctx context.Context, req JobRequest) (*model.Job, error) {
	ctx, span := telemetry.StartAuto(ctx, s.Create)
	defer span.End()

	id := ulid.String()
	rootID := id
	var depth int64

	if req.ParentID != "" {
		parent, err := s.Get(ctx, req.ParentID)
		if err != nil {
			return nil, err
		}
		depth = parent.Depth + 1
		if depth > s.maxDepth {
			return nil, model.ErrMaxDepthExceeded
		}
		rootID = parent.RootID
		if rootID == "" {
			rootID = parent.ID
		}
	}

	params := strings.TrimSpace(req.Params)
	if params == "" {
		params = "{}"
	}

	now := time.Now()
	job := &model.Job{
		ID:               id,
		ParentID:         req.ParentID,
		RootID:           rootID,
		Depth:            depth,
		RepositoryID:     req.RepositoryID,
		UserID:           req.UserID,
		WorkingDirectory: req.WorkingDirectory,
		Command:          req.Command,
		Ref:              req.Ref,
		CloneDepth:       req.CloneDepth,
		Labels:           strings.Join(req.Labels, ","),
		Params:           params,
		// Patterns are normalized on the way in, so an agent reading
		// the column never has to decide what `../../etc` means.
		ArtefactPaths: model.JoinArtefactPatterns(req.Artefacts),
		Status:        model.JobStatusPending,
	}
	job.SetCreatedAt(now)
	job.SetUpdatedAt(now)

	if err := client(s.db).Insert(ctx, model.JobTable, job); err != nil {
		return nil, fmt.Errorf("create job: %w", err)
	}

	return job, nil
}

// Get returns a job by ID.
func (s *JobStorage) Get(ctx context.Context, id string) (*model.Job, error) {
	query := `SELECT * FROM ` + model.JobTable + ` WHERE id = ?`
	return client(s.db).Get[model.Job](ctx, query, id)
}

// Claim takes an exclusive lease on the oldest pending job an agent can
// run, or returns sql.ErrNoRows when the queue holds nothing for it.
//
// The claim runs in a transaction and the UPDATE re-asserts
// `status = 'pending'`, so two agents racing for the same row produce
// one winner and one empty poll rather than a job running twice.
func (s *JobStorage) Claim(ctx context.Context, agentID string, labels []string) (*model.Job, error) {
	ctx, span := telemetry.StartAuto(ctx, s.Claim)
	defer span.End()

	db := client(s.db)
	if err := db.Begin(ctx); err != nil {
		return nil, fmt.Errorf("begin claim: %w", err)
	}
	defer db.Rollback()

	query := `SELECT * FROM ` + model.JobTable + ` WHERE status = ? ORDER BY created_at ASC LIMIT ?`
	candidates, err := db.Select[model.Job](ctx, query, model.JobStatusPending, defaultListLimit)
	if err != nil {
		return nil, fmt.Errorf("select pending jobs: %w", err)
	}

	now := time.Now()
	for _, job := range candidates {
		if !agentAcceptsJob(labels, job.Labels) {
			continue
		}

		update := `UPDATE ` + model.JobTable + ` SET status = ?, agent_id = ?, started_at = ?, lease_expires_at = ?, updated_at = ?
			WHERE id = ? AND status = ?`
		if err := db.Exec(ctx, update,
			model.JobStatusRunning, agentID, now, now.Add(s.leaseTTL), now,
			job.ID, model.JobStatusPending); err != nil {
			return nil, fmt.Errorf("claim job: %w", err)
		}
		if db.RowsAffected() == 0 {
			// Lost the race for this row; try the next candidate.
			continue
		}

		if err := db.Commit(); err != nil {
			return nil, fmt.Errorf("commit claim: %w", err)
		}

		job.Status = model.JobStatusRunning
		job.AgentID = agentID
		job.SetStartedAt(now)
		job.SetLeaseExpiresAt(now.Add(s.leaseTTL))
		job.SetUpdatedAt(now)
		return &job, nil
	}

	return nil, sql.ErrNoRows
}

// Heartbeat extends the lease on a claimed job. Only the agent holding
// the job may extend it.
func (s *JobStorage) Heartbeat(ctx context.Context, jobID, agentID string) error {
	now := time.Now()
	db := client(s.db)

	query := `UPDATE ` + model.JobTable + ` SET lease_expires_at = ?, updated_at = ?
		WHERE id = ? AND agent_id = ? AND status = ?`
	if err := db.Exec(ctx, query, now.Add(s.leaseTTL), now, jobID, agentID, model.JobStatusRunning); err != nil {
		return fmt.Errorf("heartbeat job: %w", err)
	}
	if db.RowsAffected() == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// CheckoutRequest is what an agent reports having checked out.
type CheckoutRequest struct {
	// Ref is the effective ref: the one the job named, or the default
	// branch the agent resolved for a job that named none.
	Ref string

	// CommitSHA is the commit the work tree was placed at.
	CommitSHA string
}

// RecordCheckout stores what an agent checked out for a job.
//
// It is guarded on the job still running, so a late report from an agent
// whose lease was already swept cannot rewrite the checkout of the run
// that replaced it.
func (s *JobStorage) RecordCheckout(ctx context.Context, jobID string, req CheckoutRequest) error {
	ctx, span := telemetry.StartAuto(ctx, s.RecordCheckout)
	defer span.End()

	now := time.Now()
	db := client(s.db)

	query := `UPDATE ` + model.JobTable + ` SET ref = ?, commit_sha = ?, updated_at = ?
		WHERE id = ? AND status = ?`
	if err := db.Exec(ctx, query, req.Ref, req.CommitSHA, now, jobID, model.JobStatusRunning); err != nil {
		return fmt.Errorf("record checkout: %w", err)
	}
	if db.RowsAffected() == 0 {
		return sql.ErrNoRows
	}

	return nil
}

// StatusRequest is the terminal report an agent (or the CLI itself)
// sends when a job settles.
type StatusRequest struct {
	Status   model.JobStatus
	ExitCode int64
	Error    string
}

// Finish settles a job into a terminal state.
//
// The update is guarded on the job not already being terminal so a late
// report from a timed-out agent can't overwrite the outcome the server
// already recorded.
func (s *JobStorage) Finish(ctx context.Context, jobID string, req StatusRequest) (*model.Job, error) {
	ctx, span := telemetry.StartAuto(ctx, s.Finish)
	defer span.End()

	if !model.TerminalJobStatus(req.Status) {
		return nil, fmt.Errorf("%q is not a terminal job status", req.Status)
	}

	now := time.Now()
	db := client(s.db)

	query := `UPDATE ` + model.JobTable + ` SET status = ?, exit_code = ?, error = ?, finished_at = ?,
			lease_expires_at = NULL, updated_at = ?
		WHERE id = ? AND status IN (?, ?)`
	if err := db.Exec(ctx, query,
		req.Status, req.ExitCode, req.Error, now, now,
		jobID, model.JobStatusPending, model.JobStatusRunning); err != nil {
		return nil, fmt.Errorf("finish job: %w", err)
	}
	if db.RowsAffected() == 0 {
		return nil, sql.ErrNoRows
	}

	return s.Get(ctx, jobID)
}

// ReclaimExpired marks running jobs whose lease has lapsed as timed out
// and returns how many were reclaimed. The server runs this on a ticker
// so an agent that disappears mid-job doesn't strand its work.
func (s *JobStorage) ReclaimExpired(ctx context.Context) (int64, error) {
	ctx, span := telemetry.StartAuto(ctx, s.ReclaimExpired)
	defer span.End()

	now := time.Now()
	db := client(s.db)

	query := `UPDATE ` + model.JobTable + ` SET status = ?, error = ?, finished_at = ?, updated_at = ?
		WHERE status = ? AND lease_expires_at IS NOT NULL AND lease_expires_at < ?`
	if err := db.Exec(ctx, query,
		model.JobStatusTimeout, "agent lease expired", now, now,
		model.JobStatusRunning, now); err != nil {
		return 0, fmt.Errorf("reclaim expired jobs: %w", err)
	}

	return db.RowsAffected(), nil
}

// ListFilter narrows a job listing. A zero filter lists everything.
type ListFilter struct {
	RepositoryID string
	UserID       string
	RootID       string
	Status       model.JobStatus
	Limit        int

	// ViewerID restricts the listing to what one user may see: their
	// own jobs, and everything under a job tree they started. Empty
	// means the caller may see everything, which is what an admin, an
	// agent and a public instance all are.
	ViewerID string
}

// List returns jobs matching the filter, newest first.
func (s *JobStorage) List(ctx context.Context, filter ListFilter) ([]model.Job, error) {
	ctx, span := telemetry.StartAuto(ctx, s.List)
	defer span.End()

	where := []string{"1 = 1"}
	args := []any{}

	if filter.RepositoryID != "" {
		where = append(where, "repository_id = ?")
		args = append(args, filter.RepositoryID)
	}
	if filter.UserID != "" {
		where = append(where, "user_id = ?")
		args = append(args, filter.UserID)
	}
	if filter.RootID != "" {
		where = append(where, "root_id = ?")
		args = append(args, filter.RootID)
	}
	if filter.Status != "" {
		where = append(where, "status = ?")
		args = append(args, filter.Status)
	}
	if filter.ViewerID != "" {
		// A job's owner is whoever dispatched it, and a job dispatched
		// from inside a running pipeline belongs to the agent's
		// account rather than to the human who started the pipeline.
		// Scoping on user_id alone would therefore hide a user's own
		// fan-out from them, so the root of the tree decides too.
		where = append(where, `(user_id = ? OR root_id IN (SELECT id FROM `+model.JobTable+` WHERE user_id = ?))`)
		args = append(args, filter.ViewerID, filter.ViewerID)
	}

	limit := filter.Limit
	if limit <= 0 {
		limit = defaultListLimit
	}
	args = append(args, limit)

	query := `SELECT * FROM ` + model.JobTable + ` WHERE ` + strings.Join(where, " AND ") + `
		ORDER BY created_at DESC LIMIT ?`
	return client(s.db).Select[model.Job](ctx, query, args...)
}

// VisibleTo reports whether a user may read one job.
//
// It is the single-job form of ListFilter.ViewerID, and answers the
// same question: the job is theirs, or it belongs to a tree they
// started. Callers who may see everything should not call this.
func (s *JobStorage) VisibleTo(ctx context.Context, job *model.Job, userID string) (bool, error) {
	if job == nil || userID == "" {
		return false, nil
	}
	if job.UserID == userID {
		return true, nil
	}

	// A root job has already been checked by its own user_id.
	if job.RootID == "" || job.RootID == job.ID {
		return false, nil
	}

	root, err := s.Get(ctx, job.RootID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// The root has been swept by retention. Whatever the child
			// belonged to is gone; fall back to the child's own owner,
			// which has already said no.
			return false, nil
		}
		return false, err
	}

	return root.UserID == userID, nil
}

// agentAcceptsJob reports whether an agent advertising the given labels
// may run a job requiring jobLabels (a comma separated list).
//
// A job with no labels runs anywhere. A job with labels needs the agent
// to advertise all of them, so `linux,arm64` does not land on an amd64
// box that only claims `linux`.
func agentAcceptsJob(agentLabels []string, jobLabels string) bool {
	required := splitLabels(jobLabels)
	if len(required) == 0 {
		return true
	}

	available := make(map[string]struct{}, len(agentLabels))
	for _, label := range agentLabels {
		if label = strings.TrimSpace(label); label != "" {
			available[label] = struct{}{}
		}
	}

	for _, label := range required {
		if _, ok := available[label]; !ok {
			return false
		}
	}
	return true
}

// splitLabels parses the comma separated labels column.
func splitLabels(labels string) []string {
	if strings.TrimSpace(labels) == "" {
		return nil
	}

	parts := strings.Split(labels, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if part = strings.TrimSpace(part); part != "" {
			result = append(result, part)
		}
	}
	return result
}

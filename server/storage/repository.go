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

// RepositoryStorage persists the git repositories the server has seen.
type RepositoryStorage struct {
	db *sqlx.DB
}

// NewRepositoryStorage returns a RepositoryStorage backed by the pool.
func NewRepositoryStorage(db *sqlx.DB) *RepositoryStorage {
	return &RepositoryStorage{db: db}
}

// RepositoryRequest is the repository detail a client reports.
type RepositoryRequest struct {
	RemoteURL     string `json:"remote_url"`
	DefaultBranch string `json:"default_branch"`
}

// Ensure returns the repository for the given remote, creating it on
// first sight.
//
// Dispatch is the only caller and it happens on every atkins run, so
// this is deliberately upsert-shaped: the client should never have to
// register a repository before using it. The slug is the identity, so
// the same repository cloned over ssh and https is one row.
func (s *RepositoryStorage) Ensure(ctx context.Context, userID string, req RepositoryRequest) (*model.Repository, error) {
	ctx, span := telemetry.StartAuto(ctx, s.Ensure)
	defer span.End()

	slug := model.RepositorySlug(req.RemoteURL)
	if slug == "" {
		return nil, model.ErrInvalidRepository
	}

	repository, err := s.GetBySlug(ctx, slug)
	switch {
	case err == nil:
		return repository, nil
	case !errors.Is(err, sql.ErrNoRows):
		return nil, err
	}

	now := time.Now()
	repository = &model.Repository{
		ID:              ulid.String(),
		Slug:            slug,
		RemoteURL:       req.RemoteURL,
		DefaultBranch:   req.DefaultBranch,
		CreatedByUserID: userID,
		IsActive:        true,
	}
	repository.SetCreatedAt(now)
	repository.SetUpdatedAt(now)

	if err := client(s.db).Insert(ctx, model.RepositoryTable, repository); err != nil {
		// Another dispatch raced us to the same slug. The unique index
		// held, so re-read rather than surfacing a conflict the caller
		// can do nothing about.
		if existing, getErr := s.GetBySlug(ctx, slug); getErr == nil {
			return existing, nil
		}
		return nil, fmt.Errorf("create repository: %w", err)
	}

	return repository, nil
}

// ProjectRequest is what a person fills in to add a project: a clone
// address, a name for it, and the defaults its runs start from.
//
// It is the other direction from Ensure. A repository discovered from a
// dispatch knows only what the checkout reported; a project somebody
// typed in has a name, and usually an opinion about which job to run.
type ProjectRequest struct {
	// Name is what the project is called on the pages. Empty falls back
	// to the last segment of the slug, so the field can be left alone.
	Name string

	// RemoteURL is the clone address. The slug derived from it is still
	// the identity: a project added here and a repository discovered
	// from a dispatch of the same remote are one row.
	RemoteURL string

	DefaultBranch string

	// Command is the default invocation for a run — the "which pipeline,
	// with which arguments" of the add form. Empty leaves the choice to
	// whoever presses run.
	Command string

	// Ref is the default ref to build. Empty means the repository's
	// default branch, resolved by the agent when the job runs.
	Ref string

	// WorkingDirectory is where in the work tree the pipeline lives, for
	// a repository carrying more than one project.
	WorkingDirectory string
}

// CreateProject records a project, or re-describes a repository the
// server had already seen.
//
// Adding a project whose remote is already known is an update rather
// than a conflict. The slug is the identity and it has not changed; what
// the person is doing is giving a row that arrived from a dispatch the
// name and the defaults it never had.
func (s *RepositoryStorage) CreateProject(ctx context.Context, userID string, req ProjectRequest) (*model.Repository, error) {
	ctx, span := telemetry.StartAuto(ctx, s.CreateProject)
	defer span.End()

	remote := strings.TrimSpace(req.RemoteURL)
	slug := model.RepositorySlug(remote)
	if slug == "" {
		return nil, model.ErrInvalidRepository
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		name = model.ProjectName(slug)
	}

	existing, err := s.GetBySlug(ctx, slug)
	switch {
	case err == nil:
		return s.UpdateProject(ctx, existing.ID, req)
	case !errors.Is(err, sql.ErrNoRows):
		return nil, err
	}

	now := time.Now()
	repository := &model.Repository{
		ID:               ulid.String(),
		Slug:             slug,
		Name:             name,
		RemoteURL:        remote,
		DefaultBranch:    strings.TrimSpace(req.DefaultBranch),
		Command:          strings.TrimSpace(req.Command),
		Ref:              strings.TrimSpace(req.Ref),
		WorkingDirectory: model.CleanWorkingDirectory(req.WorkingDirectory),
		CreatedByUserID:  userID,
		IsActive:         true,
	}
	repository.SetCreatedAt(now)
	repository.SetUpdatedAt(now)

	if err := client(s.db).Insert(ctx, model.RepositoryTable, repository); err != nil {
		// Somebody added the same remote while this form was open. The
		// unique index held; carry on with the row that won.
		if raced, getErr := s.GetBySlug(ctx, slug); getErr == nil {
			return s.UpdateProject(ctx, raced.ID, req)
		}
		return nil, fmt.Errorf("create project: %w", err)
	}

	return repository, nil
}

// UpdateProject rewrites the fields a person supplies, leaving the ones
// they left blank alone.
//
// The remote is deliberately not among them. Repointing a project at a
// different remote under the same slug is not a rename, it is a
// different project wearing this one's job history.
func (s *RepositoryStorage) UpdateProject(ctx context.Context, id string, req ProjectRequest) (*model.Repository, error) {
	ctx, span := telemetry.StartAuto(ctx, s.UpdateProject)
	defer span.End()

	current, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		name = current.Name
	}
	if name == "" {
		name = model.ProjectName(current.Slug)
	}

	branch := strings.TrimSpace(req.DefaultBranch)
	if branch == "" {
		branch = current.DefaultBranch
	}

	now := time.Now()
	db := client(s.db)

	query := `UPDATE ` + model.RepositoryTable + ` SET name = ?, default_branch = ?, command = ?, ref = ?,
			working_directory = ?, updated_at = ?
		WHERE id = ? AND deleted_at IS NULL`
	if err := db.Exec(ctx, query,
		name, branch,
		strings.TrimSpace(req.Command),
		strings.TrimSpace(req.Ref),
		model.CleanWorkingDirectory(req.WorkingDirectory),
		now, id); err != nil {
		return nil, fmt.Errorf("update project: %w", err)
	}
	if db.RowsAffected() == 0 {
		return nil, sql.ErrNoRows
	}

	return s.Get(ctx, id)
}

// SetPipelineJob records which job is going to produce the project's
// listing, and forgets the listing it replaces.
//
// Clearing the cache here rather than when the new one lands is what
// makes the page honest while the job is still queued: a tree from the
// previous commit shown beside a refresh in progress is the tree
// somebody will pick a stale job name out of.
func (s *RepositoryStorage) SetPipelineJob(ctx context.Context, id, jobID string) error {
	now := time.Now()
	db := client(s.db)

	query := `UPDATE ` + model.RepositoryTable + ` SET pipeline_job_id = ?, pipeline = '', pipeline_at = NULL, updated_at = ?
		WHERE id = ? AND deleted_at IS NULL`
	if err := db.Exec(ctx, query, jobID, now, id); err != nil {
		return fmt.Errorf("record pipeline job: %w", err)
	}
	if db.RowsAffected() == 0 {
		return sql.ErrNoRows
	}

	return nil
}

// SetPipeline caches the listing a pipeline job produced.
//
// It is stored rather than read from the artefact on every page load
// because the artefact is swept by retention and the tree is not: a
// project whose listing has aged out should still offer its jobs.
func (s *RepositoryStorage) SetPipeline(ctx context.Context, id, listing string) error {
	now := time.Now()
	db := client(s.db)

	query := `UPDATE ` + model.RepositoryTable + ` SET pipeline = ?, pipeline_at = ?, updated_at = ?
		WHERE id = ? AND deleted_at IS NULL`
	if err := db.Exec(ctx, query, listing, now, now, id); err != nil {
		return fmt.Errorf("cache pipeline listing: %w", err)
	}
	if db.RowsAffected() == 0 {
		return sql.ErrNoRows
	}

	return nil
}

// GetBySlug returns a repository by its normalized slug.
func (s *RepositoryStorage) GetBySlug(ctx context.Context, slug string) (*model.Repository, error) {
	query := `SELECT * FROM ` + model.RepositoryTable + ` WHERE slug = ? AND deleted_at IS NULL`
	return client(s.db).Get[model.Repository](ctx, query, slug)
}

// Get returns a repository by ID.
func (s *RepositoryStorage) Get(ctx context.Context, id string) (*model.Repository, error) {
	query := `SELECT * FROM ` + model.RepositoryTable + ` WHERE id = ? AND deleted_at IS NULL`
	return client(s.db).Get[model.Repository](ctx, query, id)
}

// List returns live repositories, newest first.
func (s *RepositoryStorage) List(ctx context.Context, limit int) ([]model.Repository, error) {
	ctx, span := telemetry.StartAuto(ctx, s.List)
	defer span.End()

	if limit <= 0 {
		limit = defaultListLimit
	}

	query := `SELECT * FROM ` + model.RepositoryTable + ` WHERE deleted_at IS NULL ORDER BY created_at DESC LIMIT ?`
	return client(s.db).Select[model.Repository](ctx, query, limit)
}

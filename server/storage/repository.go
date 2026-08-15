package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
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

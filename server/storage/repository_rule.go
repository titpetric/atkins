package storage

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/titpetric/oida"
	"github.com/titpetric/platform/pkg/ulid"

	"github.com/titpetric/atkins/server/model"
)

// RepositoryRuleStorage persists the repository allowlist.
type RepositoryRuleStorage struct {
	db *sqlx.DB
}

// NewRepositoryRuleStorage returns a RepositoryRuleStorage.
func NewRepositoryRuleStorage(db *sqlx.DB) *RepositoryRuleStorage {
	return &RepositoryRuleStorage{db: db}
}

// RuleRequest is the input for creating an allowlist rule.
type RuleRequest struct {
	// Pattern is matched against the repository slug. See
	// model.MatchRepository for the syntax.
	Pattern string `json:"pattern"`

	Description string `json:"description"`
}

// Create adds an allowlist rule.
func (s *RepositoryRuleStorage) Create(ctx context.Context, userID string, req RuleRequest) (*model.RepositoryRule, error) {
	ctx, span := oida.StartAuto(ctx, s.Create, oida.KindDatabase)
	defer span.End()

	pattern := strings.ToLower(strings.TrimSpace(req.Pattern))
	if pattern == "" {
		return nil, model.ErrInvalidPattern
	}

	now := time.Now()
	rule := &model.RepositoryRule{
		ID:              ulid.String(),
		Pattern:         pattern,
		Description:     strings.TrimSpace(req.Description),
		IsActive:        true,
		CreatedByUserID: userID,
	}
	rule.SetCreatedAt(now)
	rule.SetUpdatedAt(now)

	if err := client(s.db).Insert(ctx, model.RepositoryRuleTable, rule); err != nil {
		return nil, fmt.Errorf("create repository rule: %w", err)
	}

	return rule, nil
}

// Get returns a rule by ID.
func (s *RepositoryRuleStorage) Get(ctx context.Context, id string) (*model.RepositoryRule, error) {
	query := `SELECT * FROM ` + model.RepositoryRuleTable + ` WHERE id = ? AND deleted_at IS NULL`
	return client(s.db).Get[model.RepositoryRule](ctx, query, id)
}

// List returns the rules, newest first.
func (s *RepositoryRuleStorage) List(ctx context.Context) ([]model.RepositoryRule, error) {
	ctx, span := oida.StartAuto(ctx, s.List, oida.KindDatabase)
	defer span.End()

	query := `SELECT * FROM ` + model.RepositoryRuleTable + ` WHERE deleted_at IS NULL ORDER BY created_at DESC`
	return client(s.db).Select[model.RepositoryRule](ctx, query)
}

// ListActive returns only the rules that are in force.
func (s *RepositoryRuleStorage) ListActive(ctx context.Context) ([]model.RepositoryRule, error) {
	query := `SELECT * FROM ` + model.RepositoryRuleTable + ` WHERE deleted_at IS NULL AND is_active = 1 ORDER BY pattern ASC`
	return client(s.db).Select[model.RepositoryRule](ctx, query)
}

// SetActive enables or disables a rule without deleting it.
func (s *RepositoryRuleStorage) SetActive(ctx context.Context, id string, active bool) error {
	ctx, span := oida.StartAuto(ctx, s.SetActive, oida.KindDatabase)
	defer span.End()

	db := client(s.db)
	query := `UPDATE ` + model.RepositoryRuleTable + ` SET is_active = ?, updated_at = ? WHERE id = ? AND deleted_at IS NULL`
	if err := db.Exec(ctx, query, active, time.Now(), id); err != nil {
		return fmt.Errorf("update repository rule: %w", err)
	}
	if db.RowsAffected() == 0 {
		return model.ErrRuleNotFound
	}
	return nil
}

// Delete soft-deletes a rule.
func (s *RepositoryRuleStorage) Delete(ctx context.Context, id string) error {
	ctx, span := oida.StartAuto(ctx, s.Delete, oida.KindDatabase)
	defer span.End()

	now := time.Now()
	db := client(s.db)
	query := `UPDATE ` + model.RepositoryRuleTable + ` SET deleted_at = ?, updated_at = ? WHERE id = ? AND deleted_at IS NULL`
	if err := db.Exec(ctx, query, now, now, id); err != nil {
		return fmt.Errorf("delete repository rule: %w", err)
	}
	if db.RowsAffected() == 0 {
		return model.ErrRuleNotFound
	}
	return nil
}

// AllowedUnderPolicy reports whether a slug may be built under the
// given policy. Under "open" everything is; under "allowlist" a rule
// has to say so.
//
// Dispatch, triggers and the admin pages all ask this question, and an
// answer that differs between them is a hole.
func (s *RepositoryRuleStorage) AllowedUnderPolicy(ctx context.Context, policy model.RepositoryPolicy, slug string) (bool, error) {
	if policy != model.PolicyAllowlist {
		return true, nil
	}
	return s.Allowed(ctx, slug)
}

// Allowed reports whether a repository slug satisfies the allowlist.
//
// The caller decides whether the allowlist applies at all; this only
// answers "does any active rule match", which is false for an empty
// list. In allowlist mode that is the intended answer: an operator who
// turns the policy on without writing a rule has asked for nothing to
// run.
func (s *RepositoryRuleStorage) Allowed(ctx context.Context, slug string) (bool, error) {
	ctx, span := oida.StartAuto(ctx, s.Allowed, oida.KindDatabase)
	defer span.End()

	rules, err := s.ListActive(ctx)
	if err != nil {
		return false, err
	}

	for _, rule := range rules {
		if model.MatchRepository(rule.Pattern, slug) {
			return true, nil
		}
	}

	return false, nil
}

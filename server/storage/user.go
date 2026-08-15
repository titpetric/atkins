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
	"golang.org/x/crypto/bcrypt"

	"github.com/titpetric/atkins/server/model"
)

// UserStorage persists users and verifies their credentials.
type UserStorage struct {
	db *sqlx.DB
}

// NewUserStorage returns a UserStorage backed by the given pool.
func NewUserStorage(db *sqlx.DB) *UserStorage {
	return &UserStorage{db: db}
}

// dummyHash costs approximately the same to compare against as a real
// password hash. Authenticate spends it on the "no such user" path so
// response timing doesn't disclose which emails are registered.
var dummyHash []byte

func init() {
	dummyHash, _ = bcrypt.GenerateFromPassword([]byte("dummy-password-for-timing"), bcrypt.DefaultCost)
}

// CreateRequest is the input for creating a user. It is what
// `atkins --register` collects at the prompt.
type CreateRequest struct {
	Email    string `json:"email"`
	Username string `json:"username"`
	FullName string `json:"full_name"`
	Password string `json:"password"`
}

// Create inserts a user with a bcrypt-hashed password.
//
// The first user on a fresh server is made an admin: a self-hosted
// instance has nobody to grant the first grant, so the bootstrap has to
// come from somewhere.
func (s *UserStorage) Create(ctx context.Context, req CreateRequest) (*model.User, error) {
	ctx, span := telemetry.StartAuto(ctx, s.Create)
	defer span.End()

	email := strings.ToLower(strings.TrimSpace(req.Email))
	username := strings.TrimSpace(req.Username)

	if _, err := s.GetByEmail(ctx, email); err == nil {
		return nil, model.ErrEmailTaken
	} else if !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("check email: %w", err)
	}

	if _, err := s.getByUsername(ctx, username); err == nil {
		return nil, model.ErrUsernameTaken
	} else if !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("check username: %w", err)
	}

	_, hashSpan := telemetry.Start(ctx, "bcrypt.GenerateFromPassword")
	hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	hashSpan.End()
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	count, err := s.Count(ctx)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	user := &model.User{
		ID:       ulid.String(),
		Email:    email,
		Username: username,
		FullName: strings.TrimSpace(req.FullName),
		Password: string(hashed),
		IsAdmin:  count == 0,
		IsActive: true,
	}
	user.SetCreatedAt(now)
	user.SetUpdatedAt(now)

	if err := client(s.db).Insert(ctx, model.UserTable, user); err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	return user, nil
}

// Authenticate verifies an email/password pair and returns the user.
//
// Every failure mode collapses into model.ErrInvalidCredentials except
// an explicitly deactivated account, which is worth telling the caller
// about since retrying the password won't help.
func (s *UserStorage) Authenticate(ctx context.Context, email, password string) (*model.User, error) {
	ctx, span := telemetry.StartAuto(ctx, s.Authenticate)
	defer span.End()

	user, err := s.GetByEmail(ctx, strings.ToLower(strings.TrimSpace(email)))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Spend the comparison anyway; see dummyHash.
			_ = bcrypt.CompareHashAndPassword(dummyHash, []byte(password))
			return nil, model.ErrInvalidCredentials
		}
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, model.ErrInvalidCredentials
	}

	if !user.IsActive {
		return nil, model.ErrUserInactive
	}

	return user, nil
}

// Get returns a user by ID. Soft-deleted users are not returned.
func (s *UserStorage) Get(ctx context.Context, id string) (*model.User, error) {
	ctx, span := telemetry.StartAuto(ctx, s.Get)
	defer span.End()

	query := `SELECT * FROM ` + model.UserTable + ` WHERE id = ? AND deleted_at IS NULL`
	return client(s.db).Get[model.User](ctx, query, id)
}

// GetByEmail returns a user by email. Soft-deleted users are not returned.
func (s *UserStorage) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	ctx, span := telemetry.StartAuto(ctx, s.GetByEmail)
	defer span.End()

	query := `SELECT * FROM ` + model.UserTable + ` WHERE email = ? AND deleted_at IS NULL`
	return client(s.db).Get[model.User](ctx, query, email)
}

func (s *UserStorage) getByUsername(ctx context.Context, username string) (*model.User, error) {
	query := `SELECT * FROM ` + model.UserTable + ` WHERE username = ? AND deleted_at IS NULL`
	return client(s.db).Get[model.User](ctx, query, username)
}

// List returns live users, newest first.
func (s *UserStorage) List(ctx context.Context) ([]model.User, error) {
	ctx, span := telemetry.StartAuto(ctx, s.List)
	defer span.End()

	query := `SELECT * FROM ` + model.UserTable + ` WHERE deleted_at IS NULL ORDER BY created_at ASC`
	return client(s.db).Select[model.User](ctx, query)
}

// Flags is the administrative state of a user. A nil field is left
// alone, so an admin can revoke one privilege without restating the
// others.
type Flags struct {
	IsAdmin  *bool `json:"is_admin"`
	IsActive *bool `json:"is_active"`
	IsAgent  *bool `json:"is_agent"`
}

// SetFlags updates a user's administrative state.
func (s *UserStorage) SetFlags(ctx context.Context, id string, flags Flags) (*model.User, error) {
	ctx, span := telemetry.StartAuto(ctx, s.SetFlags)
	defer span.End()

	assignments := []string{}
	args := []any{}

	if flags.IsAdmin != nil {
		assignments = append(assignments, "is_admin = ?")
		args = append(args, *flags.IsAdmin)
	}
	if flags.IsActive != nil {
		assignments = append(assignments, "is_active = ?")
		args = append(args, *flags.IsActive)
	}
	if flags.IsAgent != nil {
		assignments = append(assignments, "is_agent = ?")
		args = append(args, *flags.IsAgent)
	}
	if len(assignments) == 0 {
		return s.Get(ctx, id)
	}

	assignments = append(assignments, "updated_at = ?")
	args = append(args, time.Now(), id)

	db := client(s.db)
	query := `UPDATE ` + model.UserTable + ` SET ` + strings.Join(assignments, ", ") + ` WHERE id = ? AND deleted_at IS NULL`
	if err := db.Exec(ctx, query, args...); err != nil {
		return nil, fmt.Errorf("update user: %w", err)
	}
	if db.RowsAffected() == 0 {
		return nil, sql.ErrNoRows
	}

	return s.Get(ctx, id)
}

// GuardLastAdmin refuses a flag change that would leave the instance
// with no active admin.
//
// The invariant lives here rather than in a handler because both the
// JSON API and the admin pages change these flags, and an instance that
// can be locked out of itself through one door but not the other is
// locked out either way.
func (s *UserStorage) GuardLastAdmin(ctx context.Context, id string, flags Flags) error {
	losingAdmin := flags.IsAdmin != nil && !*flags.IsAdmin
	losingAccess := flags.IsActive != nil && !*flags.IsActive
	if !losingAdmin && !losingAccess {
		return nil
	}

	target, err := s.Get(ctx, id)
	if err != nil {
		return err
	}
	if !target.IsAdmin || !target.IsActive {
		return nil
	}

	count, err := s.CountAdmins(ctx)
	if err != nil {
		return err
	}
	if count <= 1 {
		return model.ErrLastAdmin
	}

	return nil
}

// CountAdmins returns how many active admins remain. It exists so the
// API can refuse to remove the last one and lock everybody out.
func (s *UserStorage) CountAdmins(ctx context.Context) (int64, error) {
	query := `SELECT COUNT(*) AS count FROM ` + model.UserTable + `
		WHERE deleted_at IS NULL AND is_admin = 1 AND is_active = 1`
	result, err := client(s.db).Get[struct {
		Count int64 `db:"count"`
	}](ctx, query)
	if err != nil {
		return 0, fmt.Errorf("count admins: %w", err)
	}
	return result.Count, nil
}

// EnsureAgent returns the account for an enrolling agent, creating it
// on first contact.
//
// Agents authenticate with a shared enrolment token rather than a
// password, so the account gets a random one it never learns: there is
// no password login path for an agent to leak.
func (s *UserStorage) EnsureAgent(ctx context.Context, agentID string) (*model.User, error) {
	ctx, span := telemetry.StartAuto(ctx, s.EnsureAgent)
	defer span.End()

	username := "agent-" + strings.TrimSpace(agentID)
	email := username + "@agent.local"

	user, err := s.GetByEmail(ctx, email)
	if err == nil {
		if !user.IsAgent {
			// An account by this name predates the agent role, or
			// somebody registered it. Don't silently promote it.
			return nil, model.ErrForbidden
		}
		if !user.IsActive {
			return nil, model.ErrUserInactive
		}
		return user, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	password, err := newRefreshToken()
	if err != nil {
		return nil, err
	}

	created, err := s.Create(ctx, CreateRequest{
		Email:    email,
		Username: username,
		FullName: "atkins agent " + agentID,
		Password: password,
	})
	if err != nil {
		return nil, err
	}

	// Explicitly not an admin: Create promotes the first account, and
	// an agent enrolling before any human must not inherit that.
	yes, no := true, false
	return s.SetFlags(ctx, created.ID, Flags{IsAgent: &yes, IsAdmin: &no})
}

// Count returns the number of live human users.
//
// Agents are excluded on purpose. It is what decides whether the next
// registration is the bootstrap one, and an agent enrolling first
// should neither claim that slot nor close the door behind it.
func (s *UserStorage) Count(ctx context.Context) (int64, error) {
	ctx, span := telemetry.StartAuto(ctx, s.Count)
	defer span.End()

	query := `SELECT COUNT(*) AS count FROM ` + model.UserTable + ` WHERE deleted_at IS NULL AND is_agent = 0`
	result, err := client(s.db).Get[struct {
		Count int64 `db:"count"`
	}](ctx, query)
	if err != nil {
		return 0, fmt.Errorf("count users: %w", err)
	}
	return result.Count, nil
}

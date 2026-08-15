package storage

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/titpetric/platform/pkg/telemetry"
	"github.com/titpetric/platform/pkg/ulid"

	"github.com/titpetric/atkins/server/model"
)

// SessionStorage persists CLI login sessions and their refresh tokens.
type SessionStorage struct {
	db *sqlx.DB

	// ttl is how long a session (and therefore its refresh token)
	// stays valid without a re-login.
	ttl time.Duration
}

// DefaultSessionTTL is how long a CLI login lasts before the user has to
// run `atkins --login` again. It's long on purpose: the access token is
// short-lived and refreshed silently, so this is the "how often do I get
// interrupted" knob.
const DefaultSessionTTL = 30 * 24 * time.Hour

// NewSessionStorage returns a SessionStorage backed by the given pool.
// A zero ttl selects DefaultSessionTTL.
func NewSessionStorage(db *sqlx.DB, ttl time.Duration) *SessionStorage {
	if ttl <= 0 {
		ttl = DefaultSessionTTL
	}
	return &SessionStorage{db: db, ttl: ttl}
}

// SessionRequest carries the client detail recorded with a login. All
// fields are advisory; they exist so a user can tell their machines
// apart when reviewing active sessions.
type SessionRequest struct {
	Hostname   string
	UserAgent  string
	RemoteAddr string
}

// Create issues a new session with a fresh refresh token.
func (s *SessionStorage) Create(ctx context.Context, userID string, req SessionRequest) (*model.Session, error) {
	ctx, span := telemetry.StartAuto(ctx, s.Create)
	defer span.End()

	token, err := newRefreshToken()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	session := &model.Session{
		ID:           ulid.String(),
		UserID:       userID,
		RefreshToken: token,
		Hostname:     req.Hostname,
		UserAgent:    req.UserAgent,
		RemoteAddr:   req.RemoteAddr,
	}
	session.SetLastSeenAt(now)
	session.SetExpiresAt(now.Add(s.ttl))
	session.SetCreatedAt(now)
	session.SetUpdatedAt(now)

	if err := client(s.db).Insert(ctx, model.SessionTable, session); err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}

	return session, nil
}

// GetByRefreshToken returns a live session for the given refresh token.
// A revoked or expired session is reported as such rather than as a
// missing row, so the client can tell "log in again" from "bad token".
func (s *SessionStorage) GetByRefreshToken(ctx context.Context, token string) (*model.Session, error) {
	ctx, span := telemetry.StartAuto(ctx, s.GetByRefreshToken)
	defer span.End()

	query := `SELECT * FROM ` + model.SessionTable + ` WHERE refresh_token = ?`
	session, err := client(s.db).Get[model.Session](ctx, query, token)
	if err != nil {
		return nil, err
	}

	if session.RevokedAt != nil {
		return nil, model.ErrSessionRevoked
	}
	if session.ExpiresAt != nil && time.Now().After(*session.ExpiresAt) {
		return nil, model.ErrSessionExpired
	}

	return session, nil
}

// Rotate replaces the refresh token on a session and extends its expiry.
//
// The old token stops working the moment this returns: a refresh token
// is single-use, so a leaked one is detectable (the legitimate client's
// next refresh fails) and short-lived.
func (s *SessionStorage) Rotate(ctx context.Context, sessionID string) (*model.Session, error) {
	ctx, span := telemetry.StartAuto(ctx, s.Rotate)
	defer span.End()

	token, err := newRefreshToken()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	db := client(s.db)

	query := `UPDATE ` + model.SessionTable + ` SET refresh_token = ?, last_seen_at = ?, expires_at = ?, updated_at = ?
		WHERE id = ? AND revoked_at IS NULL`
	if err := db.Exec(ctx, query, token, now, now.Add(s.ttl), now, sessionID); err != nil {
		return nil, fmt.Errorf("rotate session: %w", err)
	}
	if db.RowsAffected() == 0 {
		return nil, sql.ErrNoRows
	}

	return s.Get(ctx, sessionID)
}

// Get returns a session by ID regardless of its state.
func (s *SessionStorage) Get(ctx context.Context, id string) (*model.Session, error) {
	query := `SELECT * FROM ` + model.SessionTable + ` WHERE id = ?`
	return client(s.db).Get[model.Session](ctx, query, id)
}

// Revoke marks a session as logged out. Revoking an already-revoked or
// unknown session is not an error: logout is idempotent.
func (s *SessionStorage) Revoke(ctx context.Context, sessionID string) error {
	ctx, span := telemetry.StartAuto(ctx, s.Revoke)
	defer span.End()

	now := time.Now()
	query := `UPDATE ` + model.SessionTable + ` SET revoked_at = ?, updated_at = ? WHERE id = ? AND revoked_at IS NULL`
	if err := client(s.db).Exec(ctx, query, now, now, sessionID); err != nil {
		return fmt.Errorf("revoke session: %w", err)
	}
	return nil
}

// RevokeByRefreshToken revokes the session holding the given token.
func (s *SessionStorage) RevokeByRefreshToken(ctx context.Context, token string) error {
	ctx, span := telemetry.StartAuto(ctx, s.RevokeByRefreshToken)
	defer span.End()

	now := time.Now()
	query := `UPDATE ` + model.SessionTable + ` SET revoked_at = ?, updated_at = ? WHERE refresh_token = ? AND revoked_at IS NULL`
	if err := client(s.db).Exec(ctx, query, now, now, token); err != nil {
		return fmt.Errorf("revoke session: %w", err)
	}
	return nil
}

// Touch records that a session was seen. Best-effort; failures here are
// not worth failing a request over.
func (s *SessionStorage) Touch(ctx context.Context, sessionID string) error {
	query := `UPDATE ` + model.SessionTable + ` SET last_seen_at = ? WHERE id = ?`
	return client(s.db).Exec(ctx, query, time.Now(), sessionID)
}

// newRefreshToken returns 32 bytes of entropy, base64url encoded.
func newRefreshToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate refresh token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

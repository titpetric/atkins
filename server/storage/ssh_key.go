package storage

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/titpetric/oida"
	"github.com/titpetric/platform/pkg/ulid"
	"golang.org/x/crypto/ssh"

	"github.com/titpetric/atkins/server/model"
)

// SSHKeyStorage persists the deploy keys agents clone with.
//
// The private material never leaves this package except through
// ListForAgent, which is reachable only by an enrolled agent. Admin
// listings carry the fingerprint and the public key, which is what an
// operator needs to install the key on the forge.
type SSHKeyStorage struct {
	db *sqlx.DB
}

// NewSSHKeyStorage returns an SSHKeyStorage backed by the given pool.
func NewSSHKeyStorage(db *sqlx.DB) *SSHKeyStorage {
	return &SSHKeyStorage{db: db}
}

// SSHKeyRequest is the input for adding a key.
type SSHKeyRequest struct {
	// Name identifies the key to operators. It is unique.
	Name string `json:"name"`

	// Host scopes the key to one git host, e.g. github.com. Empty
	// offers the key for any host.
	Host string `json:"host"`

	// PrivateKey is the PEM-encoded key. It must not be passphrase
	// protected: an agent has nobody to ask.
	PrivateKey string `json:"private_key"`

	// KnownHosts pins the host keys the agent will accept. When
	// empty, the agent trusts the host on first use.
	KnownHosts string `json:"known_hosts"`
}

// Create stores a key, deriving its public half and fingerprint so an
// operator never has to paste those separately.
func (s *SSHKeyStorage) Create(ctx context.Context, userID string, req SSHKeyRequest) (*model.SSHKey, error) {
	ctx, span := oida.StartAuto(ctx, s.Create, oida.KindDatabase)
	defer span.End()

	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, fmt.Errorf("name is required")
	}

	private := strings.TrimSpace(req.PrivateKey) + "\n"
	signer, err := ssh.ParsePrivateKey([]byte(private))
	if err != nil {
		// Parsing here rather than at clone time turns a typo into a
		// 400 now, instead of a mysterious job failure later.
		return nil, fmt.Errorf("%w: %s", model.ErrInvalidSSHKey, err)
	}

	now := time.Now()
	key := &model.SSHKey{
		ID:              ulid.String(),
		Name:            name,
		Host:            strings.ToLower(strings.TrimSpace(req.Host)),
		PrivateKey:      private,
		PublicKey:       strings.TrimSpace(string(ssh.MarshalAuthorizedKey(signer.PublicKey()))),
		Fingerprint:     ssh.FingerprintSHA256(signer.PublicKey()),
		KnownHosts:      strings.TrimSpace(req.KnownHosts),
		IsActive:        true,
		CreatedByUserID: userID,
	}
	key.SetCreatedAt(now)
	key.SetUpdatedAt(now)

	if err := client(s.db).Insert(ctx, model.SSHKeyTable, key); err != nil {
		return nil, fmt.Errorf("create ssh key: %w", err)
	}

	return key, nil
}

// List returns the keys, newest first, including private material.
// Callers rendering these to a user must project them first.
func (s *SSHKeyStorage) List(ctx context.Context) ([]model.SSHKey, error) {
	ctx, span := oida.StartAuto(ctx, s.List, oida.KindDatabase)
	defer span.End()

	query := `SELECT * FROM ` + model.SSHKeyTable + ` WHERE deleted_at IS NULL ORDER BY created_at DESC`
	return client(s.db).Select[model.SSHKey](ctx, query)
}

// ListForAgent returns the active keys an agent should install.
func (s *SSHKeyStorage) ListForAgent(ctx context.Context) ([]model.SSHKey, error) {
	ctx, span := oida.StartAuto(ctx, s.ListForAgent, oida.KindDatabase)
	defer span.End()

	query := `SELECT * FROM ` + model.SSHKeyTable + ` WHERE deleted_at IS NULL AND is_active = 1 ORDER BY host ASC, created_at ASC`
	return client(s.db).Select[model.SSHKey](ctx, query)
}

// SetActive enables or disables a key.
func (s *SSHKeyStorage) SetActive(ctx context.Context, id string, active bool) error {
	db := client(s.db)
	query := `UPDATE ` + model.SSHKeyTable + ` SET is_active = ?, updated_at = ? WHERE id = ? AND deleted_at IS NULL`
	if err := db.Exec(ctx, query, active, time.Now(), id); err != nil {
		return fmt.Errorf("update ssh key: %w", err)
	}
	if db.RowsAffected() == 0 {
		return model.ErrSSHKeyNotFound
	}
	return nil
}

// Delete soft-deletes a key.
func (s *SSHKeyStorage) Delete(ctx context.Context, id string) error {
	ctx, span := oida.StartAuto(ctx, s.Delete, oida.KindDatabase)
	defer span.End()

	now := time.Now()
	db := client(s.db)
	query := `UPDATE ` + model.SSHKeyTable + ` SET deleted_at = ?, updated_at = ? WHERE id = ? AND deleted_at IS NULL`
	if err := db.Exec(ctx, query, now, now, id); err != nil {
		return fmt.Errorf("delete ssh key: %w", err)
	}
	if db.RowsAffected() == 0 {
		return model.ErrSSHKeyNotFound
	}
	return nil
}

// MarkUsed records that an agent installed the key. Best effort.
func (s *SSHKeyStorage) MarkUsed(ctx context.Context, ids []string) {
	for _, id := range ids {
		query := `UPDATE ` + model.SSHKeyTable + ` SET last_used_at = ? WHERE id = ?`
		_ = client(s.db).Exec(ctx, query, time.Now(), id)
	}
}

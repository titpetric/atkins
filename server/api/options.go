package api

import (
	"time"

	"github.com/titpetric/atkins/server/storage"
)

// Options is passed from the server module scope. Every field has a
// usable zero value except the storages, which the module wires after
// it has a database handle.
type Options struct {
	// SigningKey signs access tokens.
	SigningKey string

	// TokenTTL is the lifetime of an issued access token. Zero selects
	// DefaultTokenTTL.
	TokenTTL time.Duration

	// AllowRegistration opens /api/user/register to anyone. When
	// false, registration still succeeds while the instance has no
	// users at all, so a fresh server can be bootstrapped without
	// shell access to the database.
	AllowRegistration bool

	// AgentToken is the shared secret agents present to enrol. Empty
	// means no agent can enrol, and the queue is reachable only by an
	// account an admin flagged by hand.
	AgentToken string

	UserStorage           *storage.UserStorage
	SessionStorage        *storage.SessionStorage
	RepositoryStorage     *storage.RepositoryStorage
	JobStorage            *storage.JobStorage
	JobLogStorage         *storage.JobLogStorage
	JobArtefactStorage    *storage.JobArtefactStorage
	RepositoryRuleStorage *storage.RepositoryRuleStorage
	SettingStorage        *storage.SettingStorage
	SSHKeyStorage         *storage.SSHKeyStorage
}

// DefaultTokenTTL is the access token lifetime. It is short because the
// CLI holds a refresh token and renews silently; a leaked access token
// is only useful for the rest of the hour.
const DefaultTokenTTL = time.Hour

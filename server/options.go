package server

import (
	"time"

	"github.com/titpetric/atkins/config"
)

// Options configures the atkins CI/CD server module.
//
// Values come from .atkins/config.yml with the ATKINS_* overlay already
// applied; see FromConfig. The struct stays plain so a caller embedding
// the module can build one by hand.
type Options struct {
	// SigningKey signs access tokens. Changing it invalidates every
	// issued token, which is the intended way to log everyone out.
	SigningKey string

	// AgentToken is the shared secret an agent presents to
	// /api/agent/enrol. Without one, no agent can join and the queue
	// stays empty of workers.
	AgentToken string

	// Connection is the platform database connection name. Empty
	// selects storage.ConnectionName ("atkins", from
	// PLATFORM_DB_ATKINS, falling back to PLATFORM_DB_DEFAULT).
	Connection string

	// ArtefactDir is the root the server writes the bytes of job
	// artefacts under. Empty selects DefaultArtefactDir.
	ArtefactDir string

	// TokenTTL is the access token lifetime.
	TokenTTL time.Duration

	// SessionTTL is how long a login lasts before a re-login.
	SessionTTL time.Duration

	// AllowRegistration opens /api/user/register. Registration always
	// stays open while the instance has no human users, so an operator
	// can bootstrap a fresh server with `atkins --register`.
	AllowRegistration bool

	// MaxJobDepth bounds job nesting. A job at the limit cannot
	// dispatch children.
	MaxJobDepth int64

	// LeaseTTL is how long an agent holds a claimed job before the
	// server reclaims it as timed out.
	LeaseTTL time.Duration

	// ReclaimInterval is how often the server sweeps expired leases. Zero
	// disables the sweep.
	ReclaimInterval time.Duration

	// RetentionInterval is how often job.retention and
	// job.log_retention apply. Zero disables the sweep. This is a
	// start-up flag and not a setting because it describes the machine
	// (how much load it will take to keep the tables trimmed), while the
	// windows themselves are policy an admin changes from the API.
	RetentionInterval time.Duration
}

// Environment variable names, kept here because error messages point at
// them. The configuration package owns the actual overlay.
const (
	EnvSigningKey = "ATKINS_SIGNING_KEY"
	EnvAgentToken = "ATKINS_AGENT_TOKEN"
)

// DefaultArtefactDir is where artefact bytes go when nothing says
// otherwise: beside the database, where the rest of a small instance's
// state already lives.
const DefaultArtefactDir = "artefacts"

// FromConfig builds module options from a resolved server configuration.
func FromConfig(cfg config.ServerConfig) *Options {
	return &Options{
		SigningKey:        cfg.SigningKey,
		AgentToken:        cfg.AgentToken,
		Connection:        cfg.Connection,
		ArtefactDir:       cfg.ArtefactDir,
		TokenTTL:          cfg.TokenTTL,
		SessionTTL:        cfg.SessionTTL,
		AllowRegistration: cfg.AllowRegistration,
		MaxJobDepth:       cfg.MaxJobDepth,
		LeaseTTL:          cfg.LeaseTTL,
		ReclaimInterval:   cfg.ReclaimInterval,
		RetentionInterval: cfg.RetentionInterval,
	}
}

// NewOptions returns options from the embedded default configuration.
// It covers a caller that has no resolved configuration.
func NewOptions() *Options {
	cfg, err := config.Default()
	if err != nil {
		return &Options{}
	}
	return FromConfig(cfg.Server)
}

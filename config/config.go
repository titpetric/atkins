// Package config owns the atkins configuration document.
//
// Configuration has two layers. The document — .atkins/config.yml,
// falling back to the embedded RuntimeDefaultConfig — is the source of
// truth. Environment variables are an overlay on top of it, for
// containers and CI, and only where they are actually set: an unset or
// empty ATKINS_* variable leaves the configured value alone.
//
// Every field is checked on load, and an empty one takes the default
// from the embedded document. A half-written config fails at start-up
// with a message naming the field, rather than at the first request.
package config

import "time"

// Version is the document version this build understands.
const Version = 1

// Config is the .atkins/config.yml document.
type Config struct {
	// Version identifies the document format.
	Version int `yaml:"version" json:"version"`

	// Client is how this machine talks to a server.
	Client ClientConfig `yaml:"client" json:"client"`

	// Server configures `atkins server`.
	Server ServerConfig `yaml:"server" json:"server"`

	// Agent configures `atkins worker`.
	Agent AgentConfig `yaml:"agent" json:"agent"`
}

// ClientConfig is the dispatching side.
type ClientConfig struct {
	// Server is the URL runs are dispatched to. Empty falls back to
	// the server this machine last logged in to.
	Server string `yaml:"server" json:"server" env:"ATKINS_SERVER"`

	// Credentials is where login tokens are stored. Empty means
	// $HOME/.atkins/credentials.json.
	Credentials string `yaml:"credentials" json:"credentials" env:"ATKINS_CREDENTIALS"`

	// Labels constrain which agents may run this machine's jobs.
	Labels []string `yaml:"labels" json:"labels" env:"ATKINS_LABELS"`

	// Dispatch hands runs to the server when logged in. Off means
	// always run locally.
	Dispatch bool `yaml:"dispatch" json:"dispatch" env:"ATKINS_DISPATCH"`

	// Timeout bounds an API call before the run falls back to local.
	Timeout time.Duration `yaml:"timeout" json:"timeout" env:"ATKINS_TIMEOUT"`
}

// ServerConfig is the CI/CD server.
type ServerConfig struct {
	Addr       string `yaml:"addr" json:"addr" env:"PLATFORM_SERVER_ADDR"`
	Database   string `yaml:"database" json:"database" env:"PLATFORM_DB_DEFAULT"`
	Connection string `yaml:"connection" json:"connection" env:"ATKINS_DB_CONNECTION"`

	// SigningKey signs access tokens. Required; there is deliberately
	// no default.
	SigningKey string `yaml:"signing_key" json:"signing_key" env:"ATKINS_SIGNING_KEY"`

	// AgentToken is the shared secret agents enrol with.
	AgentToken string `yaml:"agent_token" json:"agent_token" env:"ATKINS_AGENT_TOKEN"`

	// ArtefactDir is the root the bytes of job artefacts are written
	// under. The database records what an artefact is; this is where
	// the file itself goes.
	ArtefactDir string `yaml:"artefact_dir" json:"artefact_dir" env:"ATKINS_ARTEFACT_DIR"`

	AllowRegistration bool          `yaml:"allow_registration" json:"allow_registration" env:"ATKINS_ALLOW_REGISTRATION"`
	TokenTTL          time.Duration `yaml:"token_ttl" json:"token_ttl" env:"ATKINS_TOKEN_TTL"`
	SessionTTL        time.Duration `yaml:"session_ttl" json:"session_ttl" env:"ATKINS_SESSION_TTL"`
	MaxJobDepth       int64         `yaml:"max_job_depth" json:"max_job_depth" env:"ATKINS_MAX_JOB_DEPTH"`
	LeaseTTL          time.Duration `yaml:"lease_ttl" json:"lease_ttl" env:"ATKINS_LEASE_TTL"`
	ReclaimInterval   time.Duration `yaml:"reclaim_interval" json:"reclaim_interval" env:"ATKINS_RECLAIM_INTERVAL"`
}

// AgentConfig is the worker that runs jobs.
type AgentConfig struct {
	// ID is the agent's identity in job leases. Empty means the
	// hostname.
	ID string `yaml:"id" json:"id" env:"ATKINS_AGENT_ID"`

	// Token is the enrolment secret, matching the server's
	// agent_token.
	Token string `yaml:"token" json:"token" env:"ATKINS_AGENT_TOKEN"`

	// Labels advertise what this agent can run.
	Labels []string `yaml:"labels" json:"labels" env:"ATKINS_LABELS"`

	DataDir           string        `yaml:"data_dir" json:"data_dir" env:"ATKINS_DATA_DIR"`
	PollInterval      time.Duration `yaml:"poll_interval" json:"poll_interval" env:"ATKINS_POLL_INTERVAL"`
	JobTimeout        time.Duration `yaml:"job_timeout" json:"job_timeout" env:"ATKINS_JOB_TIMEOUT"`
	HeartbeatInterval time.Duration `yaml:"heartbeat_interval" json:"heartbeat_interval" env:"ATKINS_HEARTBEAT_INTERVAL"`
	Shell             string        `yaml:"shell" json:"shell" env:"ATKINS_SHELL"`
	PreferHTTPS       bool          `yaml:"prefer_https" json:"prefer_https" env:"ATKINS_GIT_PREFER_HTTPS"`
}

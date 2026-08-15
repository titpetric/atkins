package worker

import (
	"time"

	"github.com/titpetric/atkins/config"
)

// Options configures an agent.
//
// Values come from .atkins/config.yml with the ATKINS_* overlay already
// applied; see FromConfig. The struct stays plain so a caller embedding
// the package can build one by hand.
type Options struct {
	// Server is the atkins server to claim jobs from. Empty uses the
	// stored default from `atkins --login`.
	Server string

	// Token is the shared enrolment secret. It is the normal way an
	// agent joins: one secret in the environment, traded for the
	// rotating credentials the agent then uses.
	Token string

	// Email and Password log the agent in as an ordinary account that
	// an admin has flagged as an agent. Used when Token is empty.
	Email    string
	Password string

	// AgentID identifies this worker in job leases. Empty uses the
	// hostname.
	AgentID string

	// Labels advertise what this agent can run. A job requiring labels
	// only lands here if all of them are advertised.
	Labels []string

	// DataDir holds the repository cache and job work trees.
	DataDir string

	// PollInterval is how long to wait after an empty claim.
	PollInterval time.Duration

	// JobTimeout bounds a single job. The agent kills the command and
	// reports a timeout rather than holding the lease forever.
	JobTimeout time.Duration

	// HeartbeatInterval is how often a running job's lease is renewed.
	HeartbeatInterval time.Duration

	// Shell runs the job command. It receives `-c <command>`.
	Shell string

	// PreferHTTPS rewrites ssh remotes to https before cloning. A
	// container usually has no key agent, so `git@host:owner/repo.git`
	// would fail where `https://host/owner/repo.git` works.
	PreferHTTPS bool
}

// EnvToken is the enrolment secret's environment name. It is named here
// because the "no credentials configured" error points at it.
const EnvToken = "ATKINS_AGENT_TOKEN"

// FromConfig builds agent options from a resolved configuration.
func FromConfig(cfg *config.Config) *Options {
	return &Options{
		Server:            cfg.Client.Server,
		Token:             cfg.Agent.Token,
		AgentID:           cfg.Agent.ID,
		Labels:            cfg.Agent.Labels,
		DataDir:           cfg.Agent.DataDir,
		PollInterval:      cfg.Agent.PollInterval,
		JobTimeout:        cfg.Agent.JobTimeout,
		HeartbeatInterval: cfg.Agent.HeartbeatInterval,
		Shell:             cfg.Agent.Shell,
		PreferHTTPS:       cfg.Agent.PreferHTTPS,
	}
}

// NewOptions returns options from the embedded default configuration.
// It is the fallback for a caller that has no resolved config.
func NewOptions() *Options {
	cfg, err := config.Default()
	if err != nil {
		return &Options{}
	}
	return FromConfig(cfg)
}

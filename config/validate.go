package config

import (
	"fmt"
	"strings"
	"time"
)

// Validate fills empty fields from the embedded defaults and checks
// what remains.
//
// Defaulting and checking are one pass on purpose: "unset" and
// "invalid" are different problems, and only the second is worth
// refusing to start over. A missing timeout is a value nobody chose; a
// negative one is a value somebody got wrong.
func (c *Config) Validate() error {
	defaults, err := Default()
	if err != nil {
		return err
	}

	if c.Version == 0 {
		c.Version = defaults.Version
	}
	if c.Version > Version {
		return fmt.Errorf("config version %d is newer than this build understands (%d)", c.Version, Version)
	}

	c.Client.applyDefaults(defaults.Client)
	c.Server.applyDefaults(defaults.Server)
	c.Agent.applyDefaults(defaults.Agent)

	var problems []string
	problems = append(problems, c.Client.problems()...)
	problems = append(problems, c.Server.problems()...)
	problems = append(problems, c.Agent.problems()...)

	if len(problems) > 0 {
		return fmt.Errorf("invalid configuration:\n  - %s", strings.Join(problems, "\n  - "))
	}

	return nil
}

// applyDefaults fills empty client fields.
func (c *ClientConfig) applyDefaults(defaults ClientConfig) {
	c.Server = strings.TrimRight(strings.TrimSpace(c.Server), "/")
	if c.Timeout == 0 {
		c.Timeout = defaults.Timeout
	}
}

// problems reports client values that cannot work.
func (c *ClientConfig) problems() []string {
	var problems []string

	if c.Server != "" && !strings.HasPrefix(c.Server, "http://") && !strings.HasPrefix(c.Server, "https://") {
		problems = append(problems, fmt.Sprintf("client.server %q must start with http:// or https://", c.Server))
	}
	if c.Timeout < 0 {
		problems = append(problems, "client.timeout must not be negative")
	}

	return problems
}

// applyDefaults fills empty server fields.
func (s *ServerConfig) applyDefaults(defaults ServerConfig) {
	if s.Addr == "" {
		s.Addr = defaults.Addr
	}
	if s.Database == "" {
		s.Database = defaults.Database
	}
	if s.Connection == "" {
		s.Connection = defaults.Connection
	}
	if s.ArtefactDir == "" {
		s.ArtefactDir = defaults.ArtefactDir
	}
	if s.TokenTTL <= 0 {
		s.TokenTTL = defaults.TokenTTL
	}
	if s.SessionTTL <= 0 {
		s.SessionTTL = defaults.SessionTTL
	}
	if s.MaxJobDepth <= 0 {
		s.MaxJobDepth = defaults.MaxJobDepth
	}
	if s.LeaseTTL <= 0 {
		s.LeaseTTL = defaults.LeaseTTL
	}
	if s.ReclaimInterval < 0 {
		s.ReclaimInterval = defaults.ReclaimInterval
	}
	if s.RetentionInterval < 0 {
		s.RetentionInterval = defaults.RetentionInterval
	}
}

// problems reports server values that cannot work.
//
// A missing signing key is not listed here: it only matters when the
// server actually starts, and `atkins --config` should not refuse to
// open a document just because the machine isn't a server.
func (s *ServerConfig) problems() []string {
	var problems []string

	if !strings.Contains(s.Addr, ":") {
		problems = append(problems, fmt.Sprintf("server.addr %q must include a port, e.g. :3200", s.Addr))
	}
	if !strings.Contains(s.Database, "://") {
		problems = append(problems, fmt.Sprintf("server.database %q must be a DSN, e.g. sqlite://file:atkins.db", s.Database))
	}
	if s.SessionTTL < s.TokenTTL {
		problems = append(problems, "server.session_ttl must be at least server.token_ttl, or a login expires before its first refresh")
	}

	return problems
}

// applyDefaults fills empty agent fields.
func (a *AgentConfig) applyDefaults(defaults AgentConfig) {
	if a.DataDir == "" {
		a.DataDir = defaults.DataDir
	}
	if a.PollInterval <= 0 {
		a.PollInterval = defaults.PollInterval
	}
	if a.JobTimeout <= 0 {
		a.JobTimeout = defaults.JobTimeout
	}
	if a.HeartbeatInterval <= 0 {
		a.HeartbeatInterval = defaults.HeartbeatInterval
	}
	if a.Shell == "" {
		a.Shell = defaults.Shell
	}
}

// problems reports agent values that cannot work.
func (a *AgentConfig) problems() []string {
	var problems []string

	if a.HeartbeatInterval >= a.JobTimeout {
		problems = append(problems, "agent.heartbeat_interval must be shorter than agent.job_timeout")
	}
	if a.PollInterval > time.Minute {
		problems = append(problems, "agent.poll_interval above a minute makes the queue feel broken; keep it under 60s")
	}

	return problems
}

package api

import (
	"crypto/subtle"
	"errors"
	"net/http"
	"strings"

	"github.com/titpetric/platform"

	"github.com/titpetric/atkins/server/model"
)

// EnrolRequest is the body of POST /api/agent/enrol.
type EnrolRequest struct {
	// Token is the shared enrolment secret configured on the server.
	Token string `json:"token"`

	// AgentID names the agent. Enrolling twice with the same ID
	// returns the same account, so a restarted agent keeps its
	// identity and its job history.
	AgentID string `json:"agent_id"`

	// Labels are advertised for information; the claim call is what
	// actually filters the queue.
	Labels []string `json:"labels"`
}

// PolicyResponse tells an agent what it is allowed to run.
//
// The agent enforces this itself before cloning anything. The server
// already refuses a disallowed dispatch, but a job can outlive the rule
// that admitted it: a queued job whose repository was removed from the
// allowlist must not run just because it was queued first.
type PolicyResponse struct {
	// Policy is "open" or "allowlist".
	Policy string `json:"policy"`

	// Patterns are the active allowlist rules. Empty under an open
	// policy, and meaningful only under "allowlist".
	Patterns []string `json:"patterns"`
}

// AgentSSHKey is a deploy key handed to an agent, private half and all.
type AgentSSHKey struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Host        string `json:"host"`
	PrivateKey  string `json:"private_key"`
	KnownHosts  string `json:"known_hosts"`
	Fingerprint string `json:"fingerprint"`
}

// MountAgent registers the routes only agents use.
func (s *Handlers) MountAgent(r platform.Router) {
	r.Group(func(r platform.Router) {
		r.Post("/api/agent/enrol", s.Enrol)
		r.Get("/api/agent/policy", s.AgentPolicy)
		r.Get("/api/agent/ssh-key", s.AgentSSHKeys)
	})
}

// requireAgent authenticates the caller and refuses anyone who is not
// an agent.
//
// An admin is not admitted, deliberately. `is_agent` means what the
// roles section says it means, and widening it for convenience would
// undo something built on purpose: SSHKeyView withholds private key
// material from the admin surface, and these routes hand it over. An
// admin who needs to act as an agent enrols one, which is a decision
// written down rather than a side effect.
//
// Reporting on a job asks a narrower question than this one — who ran
// it — and reportableJob answers it.
func (s *Handlers) requireAgent(r *http.Request) (*model.User, error) {
	user, _, err := s.authenticateUser(r)
	if err != nil {
		return nil, err
	}
	if !user.IsAgent {
		return nil, requestError(http.StatusForbidden, model.ErrForbidden)
	}
	return user, nil
}

// Enrol exchanges the shared enrolment token for agent credentials.
//
// Agents don't have passwords. An operator puts one secret in the
// agent's environment, and the agent trades it for the same rotating
// token pair a logged-in human holds. That keeps one long-lived secret
// in one place instead of a password per worker.
func (s *Handlers) Enrol(w http.ResponseWriter, r *http.Request) {
	s.respond(w, r, s.enrol(w, r))
}

func (s *Handlers) enrol(w http.ResponseWriter, r *http.Request) error {
	var req EnrolRequest
	if err := decode(r, &req); err != nil {
		return err
	}

	if !s.validAgentToken(req.Token) {
		return requestError(http.StatusUnauthorized, model.ErrInvalidAgentToken)
	}

	agentID := strings.TrimSpace(req.AgentID)
	if agentID == "" {
		return requestError(http.StatusBadRequest, errors.New("agent_id is required"))
	}

	user, err := s.users.EnsureAgent(r.Context(), agentID)
	if err != nil {
		if errors.Is(err, model.ErrForbidden) {
			return requestError(http.StatusConflict, errors.New("that agent id is taken by a non-agent account"))
		}
		if errors.Is(err, model.ErrUserInactive) {
			return requestError(http.StatusForbidden, model.ErrUserInactive)
		}
		return err
	}

	response, err := s.issueWithHostname(r, user, agentID)
	if err != nil {
		return err
	}

	platform.JSON(w, r, http.StatusOK, response)
	return nil
}

// validAgentToken compares the presented token in constant time. A
// server with no token configured enrols nobody.
func (s *Handlers) validAgentToken(presented string) bool {
	if s.agentToken == "" || presented == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(s.agentToken), []byte(presented)) == 1
}

// AgentPolicy returns the repository policy in force.
func (s *Handlers) AgentPolicy(w http.ResponseWriter, r *http.Request) {
	s.respond(w, r, s.agentPolicy(w, r))
}

func (s *Handlers) agentPolicy(w http.ResponseWriter, r *http.Request) error {
	if _, err := s.requireAgent(r); err != nil {
		return err
	}

	response := PolicyResponse{
		Policy:   s.settings.Get(model.SettingRepositoryPolicy),
		Patterns: []string{},
	}

	if response.Policy == model.PolicyAllowlist {
		rules, err := s.rules.ListActive(r.Context())
		if err != nil {
			return err
		}
		for _, rule := range rules {
			response.Patterns = append(response.Patterns, rule.Pattern)
		}
	}

	platform.JSON(w, r, http.StatusOK, response)
	return nil
}

// AgentSSHKeys returns the deploy keys an agent should install.
func (s *Handlers) AgentSSHKeys(w http.ResponseWriter, r *http.Request) {
	s.respond(w, r, s.agentSSHKeys(w, r))
}

func (s *Handlers) agentSSHKeys(w http.ResponseWriter, r *http.Request) error {
	if _, err := s.requireAgent(r); err != nil {
		return err
	}

	keys, err := s.sshKeys.ListForAgent(r.Context())
	if err != nil {
		return err
	}

	views := make([]AgentSSHKey, 0, len(keys))
	ids := make([]string, 0, len(keys))
	for _, key := range keys {
		views = append(views, AgentSSHKey{
			ID:          key.ID,
			Name:        key.Name,
			Host:        key.Host,
			PrivateKey:  key.PrivateKey,
			KnownHosts:  key.KnownHosts,
			Fingerprint: key.Fingerprint,
		})
		ids = append(ids, key.ID)
	}

	s.sshKeys.MarkUsed(r.Context(), ids)

	platform.JSON(w, r, http.StatusOK, views)
	return nil
}

// allowedRepository reports whether a slug may be built under the
// policy in force.
func (s *Handlers) allowedRepository(r *http.Request, slug string) (bool, error) {
	return s.rules.AllowedUnderPolicy(r.Context(), s.settings.Get(model.SettingRepositoryPolicy), slug)
}

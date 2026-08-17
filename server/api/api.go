// Package api implements the JSON endpoints of the atkins CI/CD server.
//
// The endpoints fall into two groups. `/api/user/*` is the login flow the
// `atkins --login https://domain` client drives. `/api/dispatch` and
// `/api/job/*` are the queue: the CLI records a run, agents claim work
// and report back.
//
// Handlers decode, validate, map errors to status codes and encode. They
// call storage rather than issuing SQL.
package api

import (
	"database/sql"
	"errors"
	"net/http"
	"time"

	"github.com/titpetric/platform"

	"github.com/titpetric/atkins/server/auth"
	"github.com/titpetric/atkins/server/model"
	"github.com/titpetric/atkins/server/storage"
	"github.com/titpetric/atkins/server/stream"
)

// Handlers serves the atkins server API.
type Handlers struct {
	jwt      *auth.JWT
	tokenTTL time.Duration

	allowRegistration bool

	// agentToken enrols agents. Empty disables enrolment entirely.
	agentToken string

	users        *storage.UserStorage
	sessions     *storage.SessionStorage
	repositories *storage.RepositoryStorage
	jobs         *storage.JobStorage
	jobLogs      *storage.JobLogStorage
	artefacts    *storage.JobArtefactStorage
	rules        *storage.RepositoryRuleStorage
	settings     *storage.SettingStorage
	sshKeys      *storage.SSHKeyStorage

	// live is the running jobs' terminals. A nil hub means the module
	// wired none, and every path through here tolerates that: output is
	// stored either way, and a job with nowhere to publish is simply one
	// nobody can watch live.
	live *stream.Hub
}

// NewHandlers returns Handlers configured from opts.
func NewHandlers(opts Options) *Handlers {
	ttl := opts.TokenTTL
	if ttl <= 0 {
		ttl = DefaultTokenTTL
	}

	return &Handlers{
		jwt:               auth.NewJWT(opts.SigningKey),
		tokenTTL:          ttl,
		allowRegistration: opts.AllowRegistration,
		agentToken:        opts.AgentToken,
		users:             opts.UserStorage,
		sessions:          opts.SessionStorage,
		repositories:      opts.RepositoryStorage,
		jobs:              opts.JobStorage,
		jobLogs:           opts.JobLogStorage,
		artefacts:         opts.JobArtefactStorage,
		rules:             opts.RepositoryRuleStorage,
		settings:          opts.SettingStorage,
		sshKeys:           opts.SSHKeyStorage,
		live:              opts.Stream,
	}
}

// Mount registers the API routes on the given router.
func (s *Handlers) Mount(r platform.Router) {
	r.Group(func(r platform.Router) {
		r.Post("/api/user/register", s.Register)
		r.Post("/api/user/login", s.Login)
		r.Post("/api/user/refreshToken", s.RefreshToken)
		r.Post("/api/user/logout", s.Logout)
		r.Get("/api/user/whoami", s.Whoami)

		r.Post("/api/dispatch", s.Dispatch)

		r.Get("/api/repository", s.ListRepositories)
		r.Post("/api/repository/{repositoryID}/trigger", s.Trigger)

		r.Get("/api/job", s.ListJobs)
		r.Post("/api/job/{jobID}/retry", s.RetryJob)
		r.Post("/api/job/{jobID}/cancel", s.CancelJob)
		r.Post("/api/job/claim", s.ClaimJob)
		r.Get("/api/job/{jobID}", s.GetJob)
		r.Post("/api/job/{jobID}/status", s.JobStatus)
		r.Post("/api/job/{jobID}/checkout", s.JobCheckout)
		r.Post("/api/job/{jobID}/heartbeat", s.JobHeartbeat)
		r.Get("/api/job/{jobID}/log", s.GetJobLog)
		r.Post("/api/job/{jobID}/log", s.AppendJobLog)
		r.Get("/api/job/{jobID}/input", s.CollectJobInput)

		r.Get("/api/job/{jobID}/artefact", s.ListArtefacts)
		r.Post("/api/job/{jobID}/artefact", s.UploadArtefact)
		r.Get("/api/job/{jobID}/artefact/{artefactID}", s.DownloadArtefact)
	})

	s.MountAgent(r)
	s.MountAdmin(r)
}

// registrationOpen reports whether anyone may register right now. The
// stored setting wins over the start-up flag, so an admin can open and
// close registration without a restart.
func (s *Handlers) registrationOpen() bool {
	if s.settings != nil && s.settings.Bool(model.SettingRegistrationOpen) {
		return true
	}
	return s.allowRegistration
}

// jobScope returns the user id job reads should be narrowed to, or an
// empty string for a caller who may see every job.
//
// Admins see everything because that is what the flag is for, and
// agents because a worker operates on the whole queue. Everyone else
// sees their own jobs unless the instance has been made public, which
// is the single-team mode the server started out in.
func (s *Handlers) jobScope(user *model.User) string {
	if user == nil {
		return ""
	}
	if user.IsAdmin || user.IsAgent {
		return ""
	}
	if s.settings != nil && s.settings.Get(model.SettingJobVisibility) == model.VisibilityPublic {
		return ""
	}
	return user.ID
}

// readableJob loads a job the caller is allowed to read.
//
// A job the caller may not see is reported as missing rather than as
// forbidden: "not yours" and "not here" should look the same, or the
// endpoint tells a stranger which job IDs exist.
func (s *Handlers) readableJob(r *http.Request, user *model.User, jobID string) (*model.Job, error) {
	job, err := s.jobs.Get(r.Context(), jobID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, requestError(http.StatusNotFound, errJobNotFound)
		}
		return nil, err
	}

	scope := s.jobScope(user)
	if scope == "" {
		return job, nil
	}

	visible, err := s.jobs.VisibleTo(r.Context(), job, scope)
	if err != nil {
		return nil, err
	}
	if !visible {
		return nil, requestError(http.StatusNotFound, errJobNotFound)
	}

	return job, nil
}

// errJobNotFound covers both "no such job" and "not yours".
var errJobNotFound = errors.New("job not found")

// viewToken returns the token that opens a job page without a session,
// or an empty string on an instance whose pages are public anyway.
func (s *Handlers) viewToken(jobID string) string {
	if s.settings != nil && s.settings.Get(model.SettingJobVisibility) == model.VisibilityPublic {
		return ""
	}
	return s.jwt.ViewToken(jobID)
}

// respond writes err as JSON, or does nothing when the handler already
// wrote its response.
func (s *Handlers) respond(w http.ResponseWriter, r *http.Request, err error) {
	if err == nil {
		return
	}

	var requestErr *RequestError
	if errors.As(err, &requestErr) {
		platform.Error(w, r, requestErr.StatusCode, requestErr.Err)
		return
	}

	platform.Error(w, r, http.StatusInternalServerError, err)
}

// authenticate validates the Authorization header and returns its claims.
func (s *Handlers) authenticate(r *http.Request) (*auth.Claims, error) {
	header := r.Header.Get("Authorization")
	if header == "" {
		return nil, requestError(http.StatusUnauthorized, errors.New("missing authorization header"))
	}

	claims, err := s.jwt.Claims(header)
	if err != nil {
		return nil, requestError(http.StatusUnauthorized, auth.ErrInvalidToken)
	}

	return claims, nil
}

// authenticateUser validates the Authorization header and loads the user
// it belongs to, rejecting tokens whose session has since been revoked.
//
// Access tokens are short-lived but not zero-lived; checking the session
// here is what makes `atkins --logout` on a lost laptop take effect now
// rather than when the token happens to expire.
func (s *Handlers) authenticateUser(r *http.Request) (*model.User, *auth.Claims, error) {
	claims, err := s.authenticate(r)
	if err != nil {
		return nil, nil, err
	}

	if claims.SessionID != "" {
		session, err := s.sessions.Get(r.Context(), claims.SessionID)
		if err != nil || session.RevokedAt != nil {
			return nil, nil, requestError(http.StatusUnauthorized, model.ErrSessionRevoked)
		}
	}

	user, err := s.users.Get(r.Context(), claims.UserID)
	if err != nil {
		return nil, nil, requestError(http.StatusUnauthorized, model.ErrInvalidCredentials)
	}
	if !user.IsActive {
		return nil, nil, requestError(http.StatusForbidden, model.ErrUserInactive)
	}

	return user, claims, nil
}

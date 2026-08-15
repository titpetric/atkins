package api

import (
	"database/sql"
	"errors"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/titpetric/platform"

	"github.com/titpetric/atkins/server/model"
	"github.com/titpetric/atkins/server/storage"
)

// LoginRequest is the body of /api/user/login.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`

	// Hostname identifies the machine logging in, so a user can tell
	// their sessions apart. Optional.
	Hostname string `json:"hostname"`
}

// TokenResponse is what the login and refresh endpoints return. It is
// exactly what the CLI persists in ~/.atkins/credentials.json.
type TokenResponse struct {
	UserID       string `json:"user_id"`
	Username     string `json:"username"`
	Token        string `json:"token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresAt    int64  `json:"expires_at"`
}

// RegisterRequest is the body of /api/user/register.
type RegisterRequest struct {
	Email    string `json:"email"`
	Username string `json:"username"`
	FullName string `json:"full_name"`
	Password string `json:"password"`
}

// RefreshRequest is the body of /api/user/refreshToken.
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// LogoutRequest is the body of /api/user/logout. The refresh token is
// optional: an authenticated request already names its session.
type LogoutRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// WhoamiResponse describes the authenticated user.
type WhoamiResponse struct {
	UserID   string `json:"user_id"`
	Email    string `json:"email"`
	Username string `json:"username"`
	FullName string `json:"full_name"`
	IsAdmin  bool   `json:"is_admin"`
}

// minPasswordLength is the shortest password the server accepts.
const minPasswordLength = 8

// Register creates a user.
func (s *Handlers) Register(w http.ResponseWriter, r *http.Request) {
	s.respond(w, r, s.register(w, r))
}

func (s *Handlers) register(w http.ResponseWriter, r *http.Request) error {
	var req RegisterRequest
	if err := decode(r, &req); err != nil {
		return err
	}

	if err := s.registrationAllowed(r); err != nil {
		return err
	}

	if strings.TrimSpace(req.Email) == "" || strings.TrimSpace(req.Username) == "" {
		return requestError(http.StatusBadRequest, errors.New("email and username are required"))
	}
	if len(req.Password) < minPasswordLength {
		return requestError(http.StatusBadRequest, errors.New("password must be at least 8 characters"))
	}

	user, err := s.users.Create(r.Context(), storage.CreateRequest{
		Email:    req.Email,
		Username: req.Username,
		FullName: req.FullName,
		Password: req.Password,
	})
	if err != nil {
		if errors.Is(err, model.ErrEmailTaken) || errors.Is(err, model.ErrUsernameTaken) {
			return requestError(http.StatusConflict, err)
		}
		return err
	}

	response, err := s.issue(r, user)
	if err != nil {
		return err
	}

	platform.JSON(w, r, http.StatusCreated, response)
	return nil
}

// registrationAllowed gates open registration.
//
// A brand new instance has to let somebody in, so registration is open
// while the user table is empty regardless of configuration. Once a
// first user exists, AllowRegistration decides.
func (s *Handlers) registrationAllowed(r *http.Request) error {
	if s.registrationOpen() {
		return nil
	}

	count, err := s.users.Count(r.Context())
	if err != nil {
		return err
	}
	if count == 0 {
		return nil
	}

	return requestError(http.StatusForbidden, model.ErrRegistrationClosed)
}

// Login exchanges email and password for an access and refresh token.
func (s *Handlers) Login(w http.ResponseWriter, r *http.Request) {
	s.respond(w, r, s.login(w, r))
}

func (s *Handlers) login(w http.ResponseWriter, r *http.Request) error {
	var req LoginRequest
	if err := decode(r, &req); err != nil {
		return err
	}

	user, err := s.users.Authenticate(r.Context(), req.Email, req.Password)
	if err != nil {
		if errors.Is(err, model.ErrInvalidCredentials) {
			return requestError(http.StatusUnauthorized, model.ErrInvalidCredentials)
		}
		if errors.Is(err, model.ErrUserInactive) {
			return requestError(http.StatusForbidden, model.ErrUserInactive)
		}
		return err
	}

	response, err := s.issueWithHostname(r, user, req.Hostname)
	if err != nil {
		return err
	}

	platform.JSON(w, r, http.StatusOK, response)
	return nil
}

// RefreshToken exchanges a refresh token for a new access token.
func (s *Handlers) RefreshToken(w http.ResponseWriter, r *http.Request) {
	s.respond(w, r, s.refreshToken(w, r))
}

func (s *Handlers) refreshToken(w http.ResponseWriter, r *http.Request) error {
	var req RefreshRequest
	if err := decode(r, &req); err != nil {
		return err
	}
	if req.RefreshToken == "" {
		return requestError(http.StatusBadRequest, errors.New("refresh_token is required"))
	}

	session, err := s.sessions.GetByRefreshToken(r.Context(), req.RefreshToken)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows),
			errors.Is(err, model.ErrSessionExpired),
			errors.Is(err, model.ErrSessionRevoked):
			return requestError(http.StatusUnauthorized, errors.New("invalid refresh token"))
		}
		return err
	}

	user, err := s.users.Get(r.Context(), session.UserID)
	if err != nil {
		return requestError(http.StatusUnauthorized, errors.New("invalid refresh token"))
	}
	if !user.IsActive {
		return requestError(http.StatusForbidden, model.ErrUserInactive)
	}

	// Rotate: the presented refresh token is spent, and the client
	// stores the replacement returned below.
	rotated, err := s.sessions.Rotate(r.Context(), session.ID)
	if err != nil {
		return requestError(http.StatusUnauthorized, errors.New("invalid refresh token"))
	}

	token, err := s.jwt.Create(user.ID, rotated.ID, s.tokenTTL)
	if err != nil {
		return err
	}

	platform.JSON(w, r, http.StatusOK, TokenResponse{
		UserID:       user.ID,
		Username:     user.Username,
		Token:        token,
		RefreshToken: rotated.RefreshToken,
		ExpiresAt:    time.Now().Add(s.tokenTTL).Unix(),
	})
	return nil
}

// Logout revokes the session behind the presented credentials.
//
// It accepts either an Authorization header or a refresh token in the
// body, so a client whose access token has already expired can still
// log out cleanly. Logging out twice is not an error.
func (s *Handlers) Logout(w http.ResponseWriter, r *http.Request) {
	s.respond(w, r, s.logout(w, r))
}

func (s *Handlers) logout(w http.ResponseWriter, r *http.Request) error {
	var req LogoutRequest
	// A logout with no body is valid when the header carries a token.
	_ = decode(r, &req)

	revoked := false

	if req.RefreshToken != "" {
		if err := s.sessions.RevokeByRefreshToken(r.Context(), req.RefreshToken); err != nil {
			return err
		}
		revoked = true
	}

	if header := r.Header.Get("Authorization"); header != "" {
		claims, err := s.jwt.Claims(header)
		if err == nil && claims.SessionID != "" {
			if err := s.sessions.Revoke(r.Context(), claims.SessionID); err != nil {
				return err
			}
			revoked = true
		}
	}

	if !revoked {
		return requestError(http.StatusUnauthorized, errors.New("no session to revoke"))
	}

	w.WriteHeader(http.StatusNoContent)
	return nil
}

// Whoami reports the authenticated user. The CLI uses it to confirm a
// stored credential still works.
func (s *Handlers) Whoami(w http.ResponseWriter, r *http.Request) {
	s.respond(w, r, s.whoami(w, r))
}

func (s *Handlers) whoami(w http.ResponseWriter, r *http.Request) error {
	user, _, err := s.authenticateUser(r)
	if err != nil {
		return err
	}

	platform.JSON(w, r, http.StatusOK, WhoamiResponse{
		UserID:   user.ID,
		Email:    user.Email,
		Username: user.Username,
		FullName: user.FullName,
		IsAdmin:  user.IsAdmin,
	})
	return nil
}

// issue creates a session and access token for a user.
func (s *Handlers) issue(r *http.Request, user *model.User) (TokenResponse, error) {
	return s.issueWithHostname(r, user, "")
}

func (s *Handlers) issueWithHostname(r *http.Request, user *model.User, hostname string) (TokenResponse, error) {
	session, err := s.sessions.Create(r.Context(), user.ID, storage.SessionRequest{
		Hostname:   hostname,
		UserAgent:  r.UserAgent(),
		RemoteAddr: remoteAddr(r),
	})
	if err != nil {
		return TokenResponse{}, err
	}

	token, err := s.jwt.Create(user.ID, session.ID, s.tokenTTL)
	if err != nil {
		return TokenResponse{}, err
	}

	return TokenResponse{
		UserID:       user.ID,
		Username:     user.Username,
		Token:        token,
		RefreshToken: session.RefreshToken,
		ExpiresAt:    time.Now().Add(s.tokenTTL).Unix(),
	}, nil
}

// remoteAddr returns the client IP without its port.
func remoteAddr(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

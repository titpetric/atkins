package api

import (
	"database/sql"
	"errors"
	"net/http"
	"strings"

	"github.com/titpetric/platform"

	"github.com/titpetric/atkins/server/model"
	"github.com/titpetric/atkins/server/storage"
)

// UserView is how a user is described over the API. It exists so a
// password hash can never reach a response by accident.
type UserView struct {
	ID       string `json:"id"`
	Email    string `json:"email"`
	Username string `json:"username"`
	FullName string `json:"full_name"`
	IsAdmin  bool   `json:"is_admin"`
	IsActive bool   `json:"is_active"`
	IsAgent  bool   `json:"is_agent"`
}

// userView projects a stored user.
func userView(user *model.User) UserView {
	return UserView{
		ID:       user.ID,
		Email:    user.Email,
		Username: user.Username,
		FullName: user.FullName,
		IsAdmin:  user.IsAdmin,
		IsActive: user.IsActive,
		IsAgent:  user.IsAgent,
	}
}

// SSHKeyView is how a key is described to an operator: everything
// needed to recognize and install it, and none of the private half.
type SSHKeyView struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Host        string `json:"host"`
	PublicKey   string `json:"public_key"`
	Fingerprint string `json:"fingerprint"`
	IsActive    bool   `json:"is_active"`
	LastUsedAt  string `json:"last_used_at,omitempty"`
}

// sshKeyView projects a stored key.
func sshKeyView(key model.SSHKey) SSHKeyView {
	view := SSHKeyView{
		ID:          key.ID,
		Name:        key.Name,
		Host:        key.Host,
		PublicKey:   key.PublicKey,
		Fingerprint: key.Fingerprint,
		IsActive:    key.IsActive,
	}
	if key.LastUsedAt != nil {
		view.LastUsedAt = key.LastUsedAt.UTC().Format("2006-01-02T15:04:05Z")
	}
	return view
}

// MountAdmin registers the administrative routes.
func (s *Handlers) MountAdmin(r platform.Router) {
	r.Group(func(r platform.Router) {
		r.Get("/api/admin/user", s.ListUsers)
		r.Post("/api/admin/user/{userID}", s.UpdateUser)

		r.Get("/api/admin/repository", s.ListRules)
		r.Post("/api/admin/repository", s.CreateRule)
		r.Post("/api/admin/repository/{ruleID}", s.UpdateRule)
		r.Delete("/api/admin/repository/{ruleID}", s.DeleteRule)

		r.Get("/api/admin/setting", s.ListSettings)
		r.Post("/api/admin/setting/{name}", s.UpdateSetting)
		r.Delete("/api/admin/setting/{name}", s.ResetSetting)

		r.Get("/api/admin/ssh-key", s.ListSSHKeys)
		r.Post("/api/admin/ssh-key", s.CreateSSHKey)
		r.Post("/api/admin/ssh-key/{keyID}", s.UpdateSSHKey)
		r.Delete("/api/admin/ssh-key/{keyID}", s.DeleteSSHKey)
	})
}

// requireAdmin authenticates the caller and refuses non-admins.
func (s *Handlers) requireAdmin(r *http.Request) (*model.User, error) {
	user, _, err := s.authenticateUser(r)
	if err != nil {
		return nil, err
	}
	if !user.IsAdmin {
		return nil, requestError(http.StatusForbidden, model.ErrForbidden)
	}
	return user, nil
}

// ListUsers returns every account.
func (s *Handlers) ListUsers(w http.ResponseWriter, r *http.Request) {
	s.respond(w, r, s.listUsers(w, r))
}

func (s *Handlers) listUsers(w http.ResponseWriter, r *http.Request) error {
	if _, err := s.requireAdmin(r); err != nil {
		return err
	}

	users, err := s.users.List(r.Context())
	if err != nil {
		return err
	}

	views := make([]UserView, 0, len(users))
	for _, user := range users {
		views = append(views, userView(&user))
	}

	platform.JSON(w, r, http.StatusOK, views)
	return nil
}

// UpdateUser changes a user's administrative flags.
func (s *Handlers) UpdateUser(w http.ResponseWriter, r *http.Request) {
	s.respond(w, r, s.updateUser(w, r))
}

func (s *Handlers) updateUser(w http.ResponseWriter, r *http.Request) error {
	if _, err := s.requireAdmin(r); err != nil {
		return err
	}

	var flags storage.Flags
	if err := decode(r, &flags); err != nil {
		return err
	}

	userID := platform.URLParam(r, "userID")

	// Refuse the two moves that lock everyone out: an admin removing
	// their own last privilege, or the last admin being deactivated.
	losingAdmin := flags.IsAdmin != nil && !*flags.IsAdmin
	losingAccess := flags.IsActive != nil && !*flags.IsActive
	if losingAdmin || losingAccess {
		if err := s.guardLastAdmin(r, userID); err != nil {
			return err
		}
	}

	user, err := s.users.SetFlags(r.Context(), userID, flags)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return requestError(http.StatusNotFound, errors.New("user not found"))
		}
		return err
	}

	platform.JSON(w, r, http.StatusOK, userView(user))
	return nil
}

// guardLastAdmin refuses a change that would leave no active admin.
func (s *Handlers) guardLastAdmin(r *http.Request, userID string) error {
	target, err := s.users.Get(r.Context(), userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return requestError(http.StatusNotFound, errors.New("user not found"))
		}
		return err
	}
	if !target.IsAdmin || !target.IsActive {
		return nil
	}

	count, err := s.users.CountAdmins(r.Context())
	if err != nil {
		return err
	}
	if count <= 1 {
		return requestError(http.StatusConflict,
			errors.New("refusing to remove the last active admin; promote someone else first"))
	}

	return nil
}

// ListRules returns the repository allowlist.
func (s *Handlers) ListRules(w http.ResponseWriter, r *http.Request) {
	s.respond(w, r, s.listRules(w, r))
}

func (s *Handlers) listRules(w http.ResponseWriter, r *http.Request) error {
	if _, err := s.requireAdmin(r); err != nil {
		return err
	}

	rules, err := s.rules.List(r.Context())
	if err != nil {
		return err
	}

	platform.JSON(w, r, http.StatusOK, rules)
	return nil
}

// CreateRule adds a repository allowlist rule.
func (s *Handlers) CreateRule(w http.ResponseWriter, r *http.Request) {
	s.respond(w, r, s.createRule(w, r))
}

func (s *Handlers) createRule(w http.ResponseWriter, r *http.Request) error {
	admin, err := s.requireAdmin(r)
	if err != nil {
		return err
	}

	var req storage.RuleRequest
	if err := decode(r, &req); err != nil {
		return err
	}

	rule, err := s.rules.Create(r.Context(), admin.ID, req)
	if err != nil {
		if errors.Is(err, model.ErrInvalidPattern) {
			return requestError(http.StatusBadRequest, err)
		}
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return requestError(http.StatusConflict, errors.New("that pattern already exists"))
		}
		return err
	}

	platform.JSON(w, r, http.StatusCreated, rule)
	return nil
}

// UpdateRule enables or disables a rule.
func (s *Handlers) UpdateRule(w http.ResponseWriter, r *http.Request) {
	s.respond(w, r, s.updateRule(w, r))
}

func (s *Handlers) updateRule(w http.ResponseWriter, r *http.Request) error {
	if _, err := s.requireAdmin(r); err != nil {
		return err
	}

	var req struct {
		IsActive bool `json:"is_active"`
	}
	if err := decode(r, &req); err != nil {
		return err
	}

	ruleID := platform.URLParam(r, "ruleID")
	if err := s.rules.SetActive(r.Context(), ruleID, req.IsActive); err != nil {
		if errors.Is(err, model.ErrRuleNotFound) {
			return requestError(http.StatusNotFound, err)
		}
		return err
	}

	rule, err := s.rules.Get(r.Context(), ruleID)
	if err != nil {
		return err
	}

	platform.JSON(w, r, http.StatusOK, rule)
	return nil
}

// DeleteRule removes a rule.
func (s *Handlers) DeleteRule(w http.ResponseWriter, r *http.Request) {
	s.respond(w, r, s.deleteRule(w, r))
}

func (s *Handlers) deleteRule(w http.ResponseWriter, r *http.Request) error {
	if _, err := s.requireAdmin(r); err != nil {
		return err
	}

	if err := s.rules.Delete(r.Context(), platform.URLParam(r, "ruleID")); err != nil {
		if errors.Is(err, model.ErrRuleNotFound) {
			return requestError(http.StatusNotFound, err)
		}
		return err
	}

	w.WriteHeader(http.StatusNoContent)
	return nil
}

// ListSettings returns every setting with its effective value.
func (s *Handlers) ListSettings(w http.ResponseWriter, r *http.Request) {
	s.respond(w, r, s.listSettings(w, r))
}

func (s *Handlers) listSettings(w http.ResponseWriter, r *http.Request) error {
	if _, err := s.requireAdmin(r); err != nil {
		return err
	}

	platform.JSON(w, r, http.StatusOK, s.settings.All())
	return nil
}

// UpdateSetting overrides one setting.
func (s *Handlers) UpdateSetting(w http.ResponseWriter, r *http.Request) {
	s.respond(w, r, s.updateSetting(w, r))
}

func (s *Handlers) updateSetting(w http.ResponseWriter, r *http.Request) error {
	admin, err := s.requireAdmin(r)
	if err != nil {
		return err
	}

	var req struct {
		Value string `json:"value"`
	}
	if err := decode(r, &req); err != nil {
		return err
	}

	name := platform.URLParam(r, "name")
	if err := s.settings.Set(r.Context(), name, req.Value, admin.ID); err != nil {
		return requestError(http.StatusBadRequest, err)
	}

	platform.JSON(w, r, http.StatusOK, s.settings.All())
	return nil
}

// ResetSetting returns a setting to its default.
func (s *Handlers) ResetSetting(w http.ResponseWriter, r *http.Request) {
	s.respond(w, r, s.resetSetting(w, r))
}

func (s *Handlers) resetSetting(w http.ResponseWriter, r *http.Request) error {
	if _, err := s.requireAdmin(r); err != nil {
		return err
	}

	if err := s.settings.Reset(r.Context(), platform.URLParam(r, "name")); err != nil {
		return requestError(http.StatusBadRequest, err)
	}

	platform.JSON(w, r, http.StatusOK, s.settings.All())
	return nil
}

// ListSSHKeys returns the deploy keys, without private material.
func (s *Handlers) ListSSHKeys(w http.ResponseWriter, r *http.Request) {
	s.respond(w, r, s.listSSHKeys(w, r))
}

func (s *Handlers) listSSHKeys(w http.ResponseWriter, r *http.Request) error {
	if _, err := s.requireAdmin(r); err != nil {
		return err
	}

	keys, err := s.sshKeys.List(r.Context())
	if err != nil {
		return err
	}

	views := make([]SSHKeyView, 0, len(keys))
	for _, key := range keys {
		views = append(views, sshKeyView(key))
	}

	platform.JSON(w, r, http.StatusOK, views)
	return nil
}

// CreateSSHKey stores a deploy key.
func (s *Handlers) CreateSSHKey(w http.ResponseWriter, r *http.Request) {
	s.respond(w, r, s.createSSHKey(w, r))
}

func (s *Handlers) createSSHKey(w http.ResponseWriter, r *http.Request) error {
	admin, err := s.requireAdmin(r)
	if err != nil {
		return err
	}

	var req storage.SSHKeyRequest
	if err := decode(r, &req); err != nil {
		return err
	}

	key, err := s.sshKeys.Create(r.Context(), admin.ID, req)
	if err != nil {
		if errors.Is(err, model.ErrInvalidSSHKey) {
			return requestError(http.StatusBadRequest, err)
		}
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return requestError(http.StatusConflict, errors.New("a key with that name already exists"))
		}
		return requestError(http.StatusBadRequest, err)
	}

	platform.JSON(w, r, http.StatusCreated, sshKeyView(*key))
	return nil
}

// UpdateSSHKey enables or disables a key.
func (s *Handlers) UpdateSSHKey(w http.ResponseWriter, r *http.Request) {
	s.respond(w, r, s.updateSSHKey(w, r))
}

func (s *Handlers) updateSSHKey(w http.ResponseWriter, r *http.Request) error {
	if _, err := s.requireAdmin(r); err != nil {
		return err
	}

	var req struct {
		IsActive bool `json:"is_active"`
	}
	if err := decode(r, &req); err != nil {
		return err
	}

	if err := s.sshKeys.SetActive(r.Context(), platform.URLParam(r, "keyID"), req.IsActive); err != nil {
		if errors.Is(err, model.ErrSSHKeyNotFound) {
			return requestError(http.StatusNotFound, err)
		}
		return err
	}

	w.WriteHeader(http.StatusNoContent)
	return nil
}

// DeleteSSHKey removes a key.
func (s *Handlers) DeleteSSHKey(w http.ResponseWriter, r *http.Request) {
	s.respond(w, r, s.deleteSSHKey(w, r))
}

func (s *Handlers) deleteSSHKey(w http.ResponseWriter, r *http.Request) error {
	if _, err := s.requireAdmin(r); err != nil {
		return err
	}

	if err := s.sshKeys.Delete(r.Context(), platform.URLParam(r, "keyID")); err != nil {
		if errors.Is(err, model.ErrSSHKeyNotFound) {
			return requestError(http.StatusNotFound, err)
		}
		return err
	}

	w.WriteHeader(http.StatusNoContent)
	return nil
}

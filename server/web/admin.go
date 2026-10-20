package web

import (
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/titpetric/platform"

	"github.com/titpetric/atkins/server/api"
	"github.com/titpetric/atkins/server/model"
	"github.com/titpetric/atkins/server/storage"
)

// Section names, used to mark the current entry in the navigation.
const (
	sectionRepositories = "repository"
	sectionAllowlist    = "allowlist"
	sectionSettings     = "setting"
	sectionUsers        = "user"
	sectionKeys         = "ssh-key"
)

// Page is the state every admin template needs, whatever it shows:
// the signed-in user, the token its forms have to echo, the current nav
// entry, and whatever the last redirect had to say.
//
// The package exports it because a template cannot reach a field
// promoted through an unexported embedded one.
type Page struct {
	User    api.UserView
	CSRF    string
	Section string
	Notice  string
	Error   string
}

// page builds the common state for a section.
func (h *Handlers) page(r *http.Request, current *session, section string) Page {
	query := r.URL.Query()

	return Page{
		User:    api.NewUserView(current.User),
		CSRF:    current.CSRF,
		Section: section,
		Notice:  query.Get("notice"),
		Error:   query.Get("error"),
	}
}

// after sends the browser back to a page with something to say. A form
// post that answered with a page would re-submit on reload; a redirect
// is what makes the back button behave.
func after(w http.ResponseWriter, r *http.Request, path, notice string, err error) {
	switch {
	case err != nil:
		path += "?error=" + url.QueryEscape(err.Error())
	case notice != "":
		path += "?notice=" + url.QueryEscape(notice)
	}

	http.Redirect(w, r, path, http.StatusSeeOther)
}

// repositoriesPage lists the repositories the server knows, each with a trigger.
type repositoriesPage struct {
	Page

	// Policy is the repository policy in force, shown here because it
	// decides whether the trigger button will work.
	Policy       string
	Repositories []repositoryRow
}

// repositoryRow is one repository with the state an operator wants.
type repositoryRow struct {
	model.Repository

	// LastJob is the most recent run, or nil for a repository the server
	// knows but never built.
	LastJob *model.Job

	// Allowed reports whether the policy in force admits this
	// repository. The page shows it ahead of the trigger button.
	Allowed bool
}

// Repositories lists the known repositories with their last job.
func (h *Handlers) Repositories(w http.ResponseWriter, r *http.Request, current *session) {
	ctx := r.Context()

	repositories, err := h.repositories.List(ctx, 0)
	if err != nil {
		h.fail(w, r, http.StatusInternalServerError, err)
		return
	}

	policy := h.settings.Get(model.SettingRepositoryPolicy)

	rows := make([]repositoryRow, 0, len(repositories))
	for _, repository := range repositories {
		row := repositoryRow{Repository: repository, Allowed: true}

		// One query per repository. The listing has a cap and an operator
		// loads this page by hand; the dispatch path never calls it.
		if jobs, err := h.jobs.List(ctx, storage.ListFilter{RepositoryID: repository.ID, Limit: 1}); err == nil && len(jobs) > 0 {
			row.LastJob = &jobs[0]
		}
		if allowed, err := h.rules.AllowedUnderPolicy(ctx, policy, repository.Slug); err == nil {
			row.Allowed = allowed
		}

		rows = append(rows, row)
	}

	h.render(w, r, repositoriesView(&repositoriesPage{
		Page:         h.page(r, current, sectionRepositories),
		Policy:       policy,
		Repositories: rows,
	}, h.links()))
}

// TriggerRepository queues a job against a known repository.
//
// This is the form behind `POST /api/repository/{id}/trigger`: copying
// a repository ID into a curl invocation to run a nightly by hand is
// exactly the sort of thing a page is for.
func (h *Handlers) TriggerRepository(w http.ResponseWriter, r *http.Request, current *session) {
	ctx := r.Context()
	const back = "/admin/repository"

	repository, err := h.repositories.Get(ctx, platform.URLParam(r, "repositoryID"))
	if err != nil {
		after(w, r, back, "", errors.New("no such repository"))
		return
	}

	command := strings.TrimSpace(r.PostFormValue("command"))
	if command == "" {
		job := strings.TrimSpace(r.PostFormValue("job"))
		if job == "" {
			after(w, r, back, "", errors.New("a job name is required"))
			return
		}
		command = "atkins " + job
	}

	// The allowlist governs a triggered job exactly as it governs a
	// dispatched one. The page hides the button for a repository the
	// policy refuses, but a hidden button is not a check.
	allowed, err := h.rules.AllowedUnderPolicy(ctx, h.settings.Get(model.SettingRepositoryPolicy), repository.Slug)
	if err != nil {
		after(w, r, back, "", err)
		return
	}
	if !allowed {
		after(w, r, back, "", model.ErrRepositoryNotAllowed)
		return
	}

	job, err := h.jobs.Create(ctx, storage.JobRequest{
		RepositoryID: repository.ID,
		UserID:       current.User.ID,
		Command:      command,
		// Empty means "resolve the repository's default branch when the
		// job runs", the meaning of a blank form field. Filling in the
		// default here would pin the branch tip at the moment of the
		// button press, a different thing.
		Ref: strings.TrimSpace(r.PostFormValue("ref")),
		// The same guard the API applies: an agent will cd into this,
		// so it can only ever mean somewhere inside the checkout.
		WorkingDirectory: model.CleanWorkingDirectory(r.PostFormValue("working_directory")),
	})
	if err != nil {
		after(w, r, back, "", err)
		return
	}

	// Straight to the job: the next thing anybody wants after pressing
	// the button is to watch it run. The link has to carry the view
	// token, or a private instance refuses the operator the page for
	// the job they just started.
	http.Redirect(w, r, h.links().Job(job.ID), http.StatusSeeOther)
}

// allowlistPage shows the policy and the rules that qualify it.
type allowlistPage struct {
	Page

	Policy string
	Rules  []model.RepositoryRule

	// ActiveRules counts the rules actually in force. Zero under an
	// allowlist policy means nothing runs at all, which is worth saying
	// out loud.
	ActiveRules int
}

// Allowlist lists the repository rules.
func (h *Handlers) Allowlist(w http.ResponseWriter, r *http.Request, current *session) {
	rules, err := h.rules.List(r.Context())
	if err != nil {
		h.fail(w, r, http.StatusInternalServerError, err)
		return
	}

	active := 0
	for _, rule := range rules {
		if rule.IsActive {
			active++
		}
	}

	h.render(w, r, allowlistView(&allowlistPage{
		Page:        h.page(r, current, sectionAllowlist),
		Policy:      h.settings.Get(model.SettingRepositoryPolicy),
		Rules:       rules,
		ActiveRules: active,
	}))
}

// CreateRule adds an allowlist rule.
func (h *Handlers) CreateRule(w http.ResponseWriter, r *http.Request, current *session) {
	const back = "/admin/allowlist"

	pattern := strings.TrimSpace(r.PostFormValue("pattern"))
	if pattern == "" {
		after(w, r, back, "", model.ErrInvalidPattern)
		return
	}

	_, err := h.rules.Create(r.Context(), current.User.ID, storage.RuleRequest{
		Pattern:     pattern,
		Description: r.PostFormValue("description"),
	})
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			err = errors.New("that pattern already exists")
		}
		after(w, r, back, "", err)
		return
	}

	after(w, r, back, "Rule added: "+pattern, nil)
}

// UpdateRule enables, disables or removes a rule.
func (h *Handlers) UpdateRule(w http.ResponseWriter, r *http.Request, _ *session) {
	const back = "/admin/allowlist"

	ctx := r.Context()
	ruleID := platform.URLParam(r, "ruleID")

	switch r.PostFormValue("action") {
	case "enable":
		after(w, r, back, "Rule enabled.", h.rules.SetActive(ctx, ruleID, true))
	case "disable":
		after(w, r, back, "Rule disabled.", h.rules.SetActive(ctx, ruleID, false))
	case "remove":
		after(w, r, back, "Rule removed.", h.rules.Delete(ctx, ruleID))
	default:
		after(w, r, back, "", errors.New("unknown action"))
	}
}

// settingsPage renders the setting registry itself.
type settingsPage struct {
	Page

	Settings []settingRow
}

// settingRow is one setting as a form control.
//
// Everything a control needs comes off the registry entry, so a setting
// added to model/setting.go appears here without a template change.
type settingRow struct {
	Name        string
	Kind        string
	Value       string
	Default     string
	Description string
	Values      []string
	IsDefault   bool
}

// Settings renders every registered setting with its effective value.
func (h *Handlers) Settings(w http.ResponseWriter, r *http.Request, current *session) {
	values := h.settings.All()

	rows := make([]settingRow, 0, len(values))
	for _, value := range values {
		rows = append(rows, settingRow{
			Name:        value.Name,
			Kind:        string(value.Kind),
			Value:       value.Value,
			Default:     value.Default,
			Description: value.Description,
			Values:      value.Values,
			IsDefault:   value.IsDefault,
		})
	}

	h.render(w, r, settingsView(&settingsPage{
		Page:     h.page(r, current, sectionSettings),
		Settings: rows,
	}))
}

// UpdateSetting overrides or resets one setting.
func (h *Handlers) UpdateSetting(w http.ResponseWriter, r *http.Request, current *session) {
	const back = "/admin/setting"

	ctx := r.Context()
	name := r.PostFormValue("name")

	if r.PostFormValue("action") == "reset" {
		after(w, r, back, name+" reset to its default.", h.settings.Reset(ctx, name))
		return
	}

	value := strings.TrimSpace(r.PostFormValue("value"))
	if err := h.settings.Set(ctx, name, value, current.User.ID); err != nil {
		after(w, r, back, "", err)
		return
	}

	after(w, r, back, name+" set to "+value+".", nil)
}

// usersPage lists the accounts and their flags.
type usersPage struct {
	Page

	Users []userRow
}

// userRow is an account projected for display, plus the one thing the
// projection cannot know.
type userRow struct {
	api.UserView

	// IsLastAdmin marks the account that must keep its admin and active
	// flags. The API refuses that change; the page says so ahead of the
	// click.
	IsLastAdmin bool
}

// Users lists accounts.
func (h *Handlers) Users(w http.ResponseWriter, r *http.Request, current *session) {
	ctx := r.Context()

	users, err := h.users.List(ctx)
	if err != nil {
		h.fail(w, r, http.StatusInternalServerError, err)
		return
	}

	admins, err := h.users.CountAdmins(ctx)
	if err != nil {
		h.fail(w, r, http.StatusInternalServerError, err)
		return
	}

	rows := make([]userRow, 0, len(users))
	for _, user := range users {
		rows = append(rows, userRow{
			UserView:    api.NewUserView(&user),
			IsLastAdmin: admins <= 1 && user.IsAdmin && user.IsActive,
		})
	}

	h.render(w, r, usersView(&usersPage{
		Page:  h.page(r, current, sectionUsers),
		Users: rows,
	}))
}

// UpdateUser toggles one flag on one account.
func (h *Handlers) UpdateUser(w http.ResponseWriter, r *http.Request, _ *session) {
	const back = "/admin/user"

	ctx := r.Context()
	userID := platform.URLParam(r, "userID")

	value, err := strconv.ParseBool(r.PostFormValue("value"))
	if err != nil {
		after(w, r, back, "", errors.New("value must be true or false"))
		return
	}

	var flags storage.Flags
	field := r.PostFormValue("field")
	switch field {
	case "admin":
		flags.IsAdmin = &value
	case "active":
		flags.IsActive = &value
	case "agent":
		flags.IsAgent = &value
	default:
		after(w, r, back, "", errors.New("unknown field"))
		return
	}

	if err := h.users.GuardLastAdmin(ctx, userID, flags); err != nil {
		after(w, r, back, "", err)
		return
	}

	user, err := h.users.SetFlags(ctx, userID, flags)
	if err != nil {
		after(w, r, back, "", err)
		return
	}

	after(w, r, back, user.Username+": "+field+" is now "+strconv.FormatBool(value)+".", nil)
}

// sshKeysPage lists deploy keys through the projection that has no
// private field to leak.
type sshKeysPage struct {
	Page

	Keys []api.SSHKeyView
}

// SSHKeys lists the deploy keys.
func (h *Handlers) SSHKeys(w http.ResponseWriter, r *http.Request, current *session) {
	keys, err := h.sshKeys.List(r.Context())
	if err != nil {
		h.fail(w, r, http.StatusInternalServerError, err)
		return
	}

	h.render(w, r, sshKeysView(&sshKeysPage{
		Page: h.page(r, current, sectionKeys),
		// api.SSHKeyView carries no private key at all, so no template
		// can render one, whatever its markup.
		Keys: api.SSHKeyViews(keys),
	}))
}

// CreateSSHKey stores a deploy key.
func (h *Handlers) CreateSSHKey(w http.ResponseWriter, r *http.Request, current *session) {
	const back = "/admin/ssh-key"

	key, err := h.sshKeys.Create(r.Context(), current.User.ID, storage.SSHKeyRequest{
		Name:       r.PostFormValue("name"),
		Host:       r.PostFormValue("host"),
		PrivateKey: r.PostFormValue("private_key"),
		KnownHosts: r.PostFormValue("known_hosts"),
	})
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			err = errors.New("a key with that name already exists")
		}
		after(w, r, back, "", err)
		return
	}

	// The fingerprint, not the key: this string lands in a URL, a
	// browser history and possibly a log.
	after(w, r, back, "Added "+key.Name+" ("+key.Fingerprint+").", nil)
}

// UpdateSSHKey activates, deactivates or removes a key.
func (h *Handlers) UpdateSSHKey(w http.ResponseWriter, r *http.Request, _ *session) {
	const back = "/admin/ssh-key"

	ctx := r.Context()
	keyID := platform.URLParam(r, "keyID")

	switch r.PostFormValue("action") {
	case "activate":
		after(w, r, back, "Key activated.", h.sshKeys.SetActive(ctx, keyID, true))
	case "deactivate":
		after(w, r, back, "Key deactivated.", h.sshKeys.SetActive(ctx, keyID, false))
	case "remove":
		after(w, r, back, "Key removed.", h.sshKeys.Delete(ctx, keyID))
	default:
		after(w, r, back, "", errors.New("unknown action"))
	}
}

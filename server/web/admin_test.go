package web

import (
	"html"
	"io/fs"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/titpetric/atkins/server/api"
	"github.com/titpetric/atkins/server/model"
)

// xss is what a client can put in any of these fields. Repository slugs
// come off a git remote, job commands off a Taskfile, rule patterns and
// key names off a form: none of them are the server's own text.
const xss = `<script>alert("x")</script>`

// adminHandlers returns Handlers with the templates parsed and no
// storage: enough to render a page from a fixture.
func adminHandlers(t *testing.T) *Handlers {
	t.Helper()

	handlers, err := NewHandlers(Options{})
	require.NoError(t, err)

	return handlers
}

// samplePage is the common state an admin template needs.
func samplePage(section string) Page {
	return Page{
		User:    api.UserView{Username: "ci", Email: "ci@example.com"},
		CSRF:    "csrf-token-value",
		Section: section,
	}
}

func TestAdminTemplatesParse(t *testing.T) {
	handlers := adminHandlers(t)

	for _, name := range []string{
		"login.html",
		"admin_repository.html",
		"admin_allowlist.html",
		"admin_setting.html",
		"admin_user.html",
		"admin_ssh_key.html",
	} {
		assert.NotNil(t, handlers.templates.Lookup(name), name)
	}
}

// TestAdminFormsCarryTheCSRFField guards the name the templates and the
// handler have to agree on. A rename on one side alone would turn every
// form into a 403 nobody could explain.
func TestAdminFormsCarryTheCSRFField(t *testing.T) {
	entries, err := fs.Glob(files, "templates/admin_*.html")
	require.NoError(t, err)
	require.NotEmpty(t, entries)

	for _, entry := range entries {
		body, err := fs.ReadFile(files, entry)
		require.NoError(t, err)
		assert.Contains(t, string(body), `name="`+csrfField+`"`, entry)
	}
}

func TestRepositoriesPageRenders(t *testing.T) {
	handlers := adminHandlers(t)

	job := &model.Job{ID: "01ARZ3NDEKTSV4RRFFQ69G5FAV", Command: "atkins test:build", Status: model.JobStatusPassed}
	job.SetCreatedAt(time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC))

	var out strings.Builder
	err := handlers.templates.ExecuteTemplate(&out, "admin_repository.html", &repositoriesPage{
		Page:   samplePage(sectionRepositories),
		Policy: model.PolicyOpen,
		Repositories: []repositoryRow{
			{
				Repository: model.Repository{
					ID:            "repo-1",
					Slug:          "github.com/titpetric/atkins",
					RemoteURL:     "git@github.com:titpetric/atkins.git",
					DefaultBranch: "main",
				},
				LastJob: job,
				Allowed: true,
			},
			{
				Repository: model.Repository{ID: "repo-2", Slug: "github.com/someone/else"},
				Allowed:    false,
			},
		},
	})
	require.NoError(t, err)

	rendered := out.String()
	assert.Contains(t, rendered, "github.com/titpetric/atkins")
	assert.Contains(t, rendered, "atkins test:build")
	assert.Contains(t, rendered, `action="/admin/repository/repo-1/trigger"`)
	assert.Contains(t, rendered, "csrf-token-value")

	// A repository the policy refuses gets no trigger button at all.
	assert.NotContains(t, rendered, `action="/admin/repository/repo-2/trigger"`)
	assert.Contains(t, rendered, "the allowlist refuses this repository")
}

func TestRepositoriesPageEscapesSlugAndCommand(t *testing.T) {
	handlers := adminHandlers(t)

	var out strings.Builder
	err := handlers.templates.ExecuteTemplate(&out, "admin_repository.html", &repositoriesPage{
		Page:   samplePage(sectionRepositories),
		Policy: model.PolicyOpen,
		Repositories: []repositoryRow{{
			Repository: model.Repository{ID: "repo-1", Slug: xss, RemoteURL: xss, DefaultBranch: xss},
			LastJob:    &model.Job{ID: "job-1", Command: xss, Status: model.JobStatusFailed},
			Allowed:    true,
		}},
	})
	require.NoError(t, err)

	assert.NotContains(t, out.String(), xss)
	assert.Contains(t, out.String(), "&lt;script&gt;")
}

func TestAllowlistPageWarnsWhenNothingCanRun(t *testing.T) {
	handlers := adminHandlers(t)

	render := func(page *allowlistPage) string {
		var out strings.Builder
		require.NoError(t, handlers.templates.ExecuteTemplate(&out, "admin_allowlist.html", page))
		return out.String()
	}

	// Allowlist on, no active rule: the page has to say that this stops
	// everything, because the symptom is otherwise a 403 per dispatch.
	stopped := render(&allowlistPage{
		Page:        samplePage(sectionAllowlist),
		Policy:      model.PolicyAllowlist,
		Rules:       []model.RepositoryRule{{ID: "rule-1", Pattern: "github.com/titpetric/*"}},
		ActiveRules: 0,
	})
	assert.Contains(t, stopped, "nothing will run")

	running := render(&allowlistPage{
		Page:        samplePage(sectionAllowlist),
		Policy:      model.PolicyAllowlist,
		Rules:       []model.RepositoryRule{{ID: "rule-1", Pattern: "github.com/titpetric/*", IsActive: true}},
		ActiveRules: 1,
	})
	assert.NotContains(t, running, "nothing will run")
	assert.Contains(t, running, "github.com/titpetric/*")
	assert.Contains(t, running, `value="disable"`)

	// Under an open policy the rules are inert, and saying so is kinder
	// than letting somebody write rules that do nothing.
	open := render(&allowlistPage{Page: samplePage(sectionAllowlist), Policy: model.PolicyOpen})
	assert.Contains(t, open, "not consulted")
}

func TestAllowlistPageEscapesPatterns(t *testing.T) {
	handlers := adminHandlers(t)

	var out strings.Builder
	err := handlers.templates.ExecuteTemplate(&out, "admin_allowlist.html", &allowlistPage{
		Page:   samplePage(sectionAllowlist),
		Policy: model.PolicyAllowlist,
		Rules:  []model.RepositoryRule{{ID: "rule-1", Pattern: xss, Description: xss, IsActive: true}},
	})
	require.NoError(t, err)

	assert.NotContains(t, out.String(), xss)
}

// TestSettingsPageRendersFromTheRegistry drives the page with the real
// registry, so a setting added to model/setting.go appears here without
// anybody editing a template.
func TestSettingsPageRendersFromTheRegistry(t *testing.T) {
	handlers := adminHandlers(t)

	rows := []settingRow{}
	for _, definition := range model.SettingDefinitions() {
		rows = append(rows, settingRow{
			Name:        definition.Name,
			Kind:        string(definition.Kind),
			Value:       definition.Default,
			Default:     definition.Default,
			Description: definition.Description,
			Values:      definition.Values,
			IsDefault:   true,
		})
	}

	var out strings.Builder
	err := handlers.templates.ExecuteTemplate(&out, "admin_setting.html", &settingsPage{
		Page:     samplePage(sectionSettings),
		Settings: rows,
	})
	require.NoError(t, err)

	rendered := out.String()
	for _, definition := range model.SettingDefinitions() {
		assert.Contains(t, rendered, definition.Name)
		// Escaped, not raw: a description containing an apostrophe —
		// "a finished job's record" — reaches the page as `&#39;`.
		assert.Contains(t, rendered, html.EscapeString(definition.Description))
	}

	// The enum renders as a choice, not as free text.
	assert.Contains(t, rendered, `<option value="allowlist"`)
	assert.Contains(t, rendered, `<option value="open" selected`)

	// Nothing is overridden, so nothing offers a reset.
	assert.NotContains(t, rendered, `value="reset"`)
}

func TestSettingsPageMarksAnOverride(t *testing.T) {
	handlers := adminHandlers(t)

	var out strings.Builder
	err := handlers.templates.ExecuteTemplate(&out, "admin_setting.html", &settingsPage{
		Page: samplePage(sectionSettings),
		Settings: []settingRow{{
			Name:    "job.max_depth",
			Kind:    string(model.KindInt),
			Value:   "7",
			Default: "3",
		}},
	})
	require.NoError(t, err)

	rendered := out.String()
	assert.Contains(t, rendered, "overridden")
	assert.Contains(t, rendered, `value="reset"`)
	assert.Contains(t, rendered, `value="7"`)
}

func TestSettingsPageEscapesValues(t *testing.T) {
	handlers := adminHandlers(t)

	var out strings.Builder
	err := handlers.templates.ExecuteTemplate(&out, "admin_setting.html", &settingsPage{
		Page: samplePage(sectionSettings),
		Settings: []settingRow{{
			Name:  "job.max_depth",
			Kind:  string(model.KindInt),
			Value: `" onfocus="alert(1)`,
		}},
	})
	require.NoError(t, err)

	// The value lands in an attribute, which is the context where a bare
	// quote would break out of it.
	assert.NotContains(t, out.String(), `onfocus="alert(1)"`)
	assert.Contains(t, out.String(), "&#34;")
}

func TestUsersPageDisablesTheLastAdmin(t *testing.T) {
	handlers := adminHandlers(t)

	var out strings.Builder
	err := handlers.templates.ExecuteTemplate(&out, "admin_user.html", &usersPage{
		Page: samplePage(sectionUsers),
		Users: []userRow{
			{
				UserView:    api.UserView{ID: "user-1", Username: "ci", Email: "ci@example.com", IsAdmin: true, IsActive: true},
				IsLastAdmin: true,
			},
			{
				UserView: api.UserView{ID: "user-2", Username: "dev", Email: "dev@example.com", IsActive: true},
			},
		},
	})
	require.NoError(t, err)

	rendered := out.String()
	assert.Contains(t, rendered, "the last active admin")
	assert.Contains(t, rendered, "disabled")
	assert.Contains(t, rendered, `action="/admin/user/user-2"`)
}

// TestUserViewHasNoPasswordField is why the users page cannot leak a
// bcrypt hash: the type it renders has nowhere to hold one.
func TestUserViewHasNoPasswordField(t *testing.T) {
	view := reflect.TypeOf(api.UserView{})

	for i := range view.NumField() {
		name := strings.ToLower(view.Field(i).Name)
		assert.NotContains(t, name, "password", view.Field(i).Name)
	}
}

func TestUsersPageEscapesAccountFields(t *testing.T) {
	handlers := adminHandlers(t)

	var out strings.Builder
	err := handlers.templates.ExecuteTemplate(&out, "admin_user.html", &usersPage{
		Page:  samplePage(sectionUsers),
		Users: []userRow{{UserView: api.UserView{ID: "user-1", Username: xss, Email: xss, FullName: xss}}},
	})
	require.NoError(t, err)

	assert.NotContains(t, out.String(), xss)
}

// TestSSHKeyViewHasNoPrivateField is the guard behind the guard. The
// deploy key page renders api.SSHKeyView, and the reason nothing on it
// can leak private material is that the type has nowhere to put it.
func TestSSHKeyViewHasNoPrivateField(t *testing.T) {
	view := reflect.TypeOf(api.SSHKeyView{})

	for i := range view.NumField() {
		name := strings.ToLower(view.Field(i).Name)
		assert.NotContains(t, name, "private", view.Field(i).Name)
		assert.NotContains(t, name, "secret", view.Field(i).Name)
	}
}

func TestSSHKeyPageRendersWithoutPrivateMaterial(t *testing.T) {
	handlers := adminHandlers(t)

	stored := model.SSHKey{
		ID:          "key-1",
		Name:        "github",
		Host:        "github.com",
		PrivateKey:  "-----BEGIN OPENSSH PRIVATE KEY-----\nnot-a-real-key\n-----END OPENSSH PRIVATE KEY-----",
		PublicKey:   "ssh-ed25519 AAAAC3Nz",
		Fingerprint: "SHA256:abcdef",
		KnownHosts:  "github.com ssh-ed25519 AAAAC3Nz",
		IsActive:    true,
	}

	var out strings.Builder
	err := handlers.templates.ExecuteTemplate(&out, "admin_ssh_key.html", &sshKeysPage{
		Page: samplePage(sectionKeys),
		Keys: api.SSHKeyViews([]model.SSHKey{stored}),
	})
	require.NoError(t, err)

	rendered := out.String()
	assert.Contains(t, rendered, "SHA256:abcdef")
	assert.Contains(t, rendered, "ssh-ed25519 AAAAC3Nz")
	assert.NotContains(t, rendered, "not-a-real-key")
	assert.NotContains(t, rendered, "PRIVATE KEY-----\nnot")
}

func TestSSHKeyPageEscapesKeyNames(t *testing.T) {
	handlers := adminHandlers(t)

	var out strings.Builder
	err := handlers.templates.ExecuteTemplate(&out, "admin_ssh_key.html", &sshKeysPage{
		Page: samplePage(sectionKeys),
		Keys: api.SSHKeyViews([]model.SSHKey{{ID: "key-1", Name: xss, Host: xss, PublicKey: xss, Fingerprint: xss}}),
	})
	require.NoError(t, err)

	assert.NotContains(t, out.String(), xss)
}

// TestFlashMessagesAreEscaped covers the notice and error a redirect
// puts in the query string. They are the one place on these pages where
// text from a URL is rendered back.
func TestFlashMessagesAreEscaped(t *testing.T) {
	handlers := adminHandlers(t)

	page := samplePage(sectionAllowlist)
	page.Notice = xss
	page.Error = xss

	var out strings.Builder
	err := handlers.templates.ExecuteTemplate(&out, "admin_allowlist.html", &allowlistPage{
		Page:   page,
		Policy: model.PolicyOpen,
	})
	require.NoError(t, err)

	assert.NotContains(t, out.String(), xss)
	assert.Contains(t, out.String(), "&lt;script&gt;")
}

func TestLoginPageRenders(t *testing.T) {
	handlers := adminHandlers(t)

	var out strings.Builder
	err := handlers.templates.ExecuteTemplate(&out, "login.html", &loginPage{
		Next:  "/admin/user",
		Error: "That email and password do not match an account.",
	})
	require.NoError(t, err)

	rendered := out.String()
	assert.Contains(t, rendered, `name="email"`)
	assert.Contains(t, rendered, `name="password"`)
	assert.Contains(t, rendered, `value="/admin/user"`)
	assert.Contains(t, rendered, "do not match an account")
}

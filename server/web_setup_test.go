package server_test

import (
	"io"
	"net/http"
	urlpkg "net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// submitSetup posts the setup form and returns the page it produced.
//
// It does not go through browser.post, which reads a CSRF token off the
// page it is submitting from: the setup form carries none, for the same
// reason the login form carries none — there is no session yet to derive
// one from, and a forged setup creates an account the attacker would
// then have to know the password of.
func submitSetup(t *testing.T, visitor *browser, form urlpkg.Values) (int, string) {
	t.Helper()

	response, err := visitor.client.PostForm(visitor.base+"/setup", form)
	require.NoError(t, err)
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	require.NoError(t, err)

	return response.StatusCode, string(body)
}

// setupForm is what somebody claiming a new instance fills in.
func setupForm() urlpkg.Values {
	return urlpkg.Values{
		"username":              {"ci"},
		"email":                 {"ci@example.com"},
		"password":              {"correct-horse"},
		"password_confirmation": {"correct-horse"},
	}
}

// An instance with no accounts has one thing worth saying, and it says
// it wherever somebody arrives: the front door, the login form, and any
// admin page they were linked to.
func TestAnEmptyInstanceSendsEverybodyToSetup(t *testing.T) {
	url := testServer(t)

	visitor := newBrowser(t, url)
	for _, path := range []string{"/", "/login", "/admin/project", "/admin/user"} {
		response := visitor.hop(http.MethodGet, path, nil)
		assert.Equal(t, http.StatusSeeOther, response.StatusCode, path)
		assert.Equal(t, "/setup", response.Header.Get("Location"), path)
	}

	status, body := visitor.get("/setup")
	assert.Equal(t, http.StatusOK, status)
	assert.Contains(t, body, `name="password_confirmation"`)
}

func TestSetupCreatesTheAdminAndSignsThemIn(t *testing.T) {
	url := testServer(t)

	visitor := newBrowser(t, url)

	response := visitor.hop(http.MethodPost, "/setup", setupForm())
	require.Equal(t, http.StatusSeeOther, response.StatusCode)
	assert.Equal(t, "/admin/project", response.Header.Get("Location"))

	// Signed in already: no second trip through the login form.
	cookies := response.Cookies()
	require.Len(t, cookies, 1)
	assert.True(t, cookies[0].HttpOnly)

	// And an administrator, by the same rule registration bootstraps
	// under: the first account on an empty instance.
	status, body := visitor.get("/admin/user")
	assert.Equal(t, http.StatusOK, status)
	assert.Contains(t, body, "ci@example.com")
}

// The page is the bootstrap, and a bootstrap that stays open is an open
// registration nobody asked for.
func TestSetupClosesOnceAnAccountExists(t *testing.T) {
	url := testServer(t)
	register(t, url)

	visitor := newBrowser(t, url)

	response := visitor.hop(http.MethodGet, "/setup", nil)
	assert.Equal(t, http.StatusSeeOther, response.StatusCode)
	assert.Equal(t, "/login", response.Header.Get("Location"))

	// Not just the form: the post is refused too, or the page would be
	// closed only to whoever bothered to load it first.
	form := setupForm()
	form.Set("username", "intruder")
	form.Set("email", "intruder@example.com")

	response = visitor.hop(http.MethodPost, "/setup", form)
	assert.Equal(t, http.StatusSeeOther, response.StatusCode)
	assert.Equal(t, "/login", response.Header.Get("Location"))

	// The account was not created, so the login form still refuses it.
	status := call(t, http.MethodPost, url+"/api/user/login", "", map[string]string{
		"email":    "intruder@example.com",
		"password": "correct-horse",
	}, nil)
	assert.Equal(t, http.StatusUnauthorized, status)
}

// An agent enrolling before any person must not claim the bootstrap slot
// or close the door behind it — the same rule registration follows.
func TestSetupStaysOpenAfterAnAgentEnrols(t *testing.T) {
	url := testServer(t)

	var enrolled tokenResponse
	status := call(t, http.MethodPost, url+"/api/agent/enrol", "", map[string]any{
		"token":    testAgentToken,
		"agent_id": "agent-1",
	}, &enrolled)
	require.Equal(t, http.StatusOK, status)

	visitor := newBrowser(t, url)

	code, body := visitor.get("/setup")
	assert.Equal(t, http.StatusOK, code)
	assert.Contains(t, body, `name="password_confirmation"`)

	response := visitor.hop(http.MethodPost, "/setup", setupForm())
	assert.Equal(t, http.StatusSeeOther, response.StatusCode)
	assert.Equal(t, "/admin/project", response.Header.Get("Location"))
}

func TestSetupRefusesAFormItCannotUse(t *testing.T) {
	for _, badForm := range []struct {
		name    string
		adjust  func(urlpkg.Values)
		message string
	}{
		{"no username", func(f urlpkg.Values) { f.Set("username", "") }, "username is required"},
		{"no email", func(f urlpkg.Values) { f.Set("email", "") }, "email address is required"},
		{"short password", func(f urlpkg.Values) {
			f.Set("password", "short")
			f.Set("password_confirmation", "short")
		}, "at least 8 characters"},
		{"mistyped password", func(f urlpkg.Values) { f.Set("password_confirmation", "correct-horse-x") }, "do not match"},
	} {
		t.Run(badForm.name, func(t *testing.T) {
			url := testServer(t)
			visitor := newBrowser(t, url)

			form := setupForm()
			badForm.adjust(form)

			status, body := submitSetup(t, visitor, form)
			assert.Equal(t, http.StatusOK, status)
			assert.Contains(t, body, badForm.message)

			// Still nobody's instance, so the page is still open.
			response := visitor.hop(http.MethodGet, "/setup", nil)
			assert.Equal(t, http.StatusOK, response.StatusCode)
		})
	}
}

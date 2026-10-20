package server_test

import (
	"io"
	"net/http"
	"net/http/cookiejar"
	urlpkg "net/url"
	"regexp"
	"strings"
	"testing"

	"github.com/titpetric/atkins/internal/assert"
	"github.com/titpetric/atkins/server/model"
)

// adminPages are the five pages that make up the admin UI.
var adminPages = []string{
	"/admin/repository",
	"/admin/allowlist",
	"/admin/setting",
	"/admin/user",
	"/admin/ssh-key",
}

// csrfPattern finds the token a page hands to its forms.
var csrfPattern = regexp.MustCompile(`name="csrf_token" value="([^"]+)"`)

// browser stands in for a person with a cookie jar. The tests drive the
// admin pages the way a browser drives them (form posts and redirects), so
// the tests exercise the same path a person does.
type browser struct {
	t      *testing.T
	base   string
	client *http.Client
}

func newBrowser(t *testing.T, base string) *browser {
	t.Helper()

	jar, err := cookiejar.New(nil)
	assert.NoError(t, err)

	return &browser{t: t, base: base, client: &http.Client{Jar: jar}}
}

// get fetches a page, following redirects, and returns the final status
// and body.
func (b *browser) get(path string) (int, string) {
	b.t.Helper()

	response, err := b.client.Get(b.base + path)
	assert.NoError(b.t, err)
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	assert.NoError(b.t, err)

	return response.StatusCode, string(body)
}

// hop issues a request without following the redirect, so a test can
// assert where the server tried to send the browser.
func (b *browser) hop(method, path string, form urlpkg.Values) *http.Response {
	b.t.Helper()

	var body io.Reader
	if form != nil {
		body = strings.NewReader(form.Encode())
	}

	request, err := http.NewRequest(method, b.base+path, body)
	assert.NoError(b.t, err)
	if form != nil {
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	client := &http.Client{
		Jar: b.client.Jar,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	response, err := client.Do(request)
	assert.NoError(b.t, err)
	_, _ = io.Copy(io.Discard, response.Body)
	response.Body.Close()

	return response
}

// csrf reads the token a page hands to its forms.
func (b *browser) csrf(path string) string {
	b.t.Helper()

	status, body := b.get(path)
	assert.Equal(b.t, http.StatusOK, status, path)

	match := csrfPattern.FindStringSubmatch(body)
	assert.Len(b.t, match, 2, "no csrf token on "+path)

	return match[1]
}

// post submits a form with the token from the page it belongs to.
func (b *browser) post(from, path string, form urlpkg.Values) (int, string) {
	b.t.Helper()

	if form == nil {
		form = urlpkg.Values{}
	}
	form.Set("csrf_token", b.csrf(from))

	response, err := b.client.PostForm(b.base+path, form)
	assert.NoError(b.t, err)
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	assert.NoError(b.t, err)

	return response.StatusCode, string(body)
}

// login signs the browser in and leaves the cookie in the jar.
//
// It stops at the redirect and does not follow it: a non-admin logs
// in perfectly well and the page it lands on then refuses it, and that
// is a case worth being able to set up.
func (b *browser) login(email, password string) {
	b.t.Helper()

	status, body := b.get("/login")
	assert.Equal(b.t, http.StatusOK, status)
	assert.Contains(b.t, body, `name="password"`)

	response := b.hop(http.MethodPost, "/login", urlpkg.Values{
		"email":    {email},
		"password": {password},
	})
	assert.Equal(b.t, http.StatusSeeOther, response.StatusCode)

	cookies := response.Cookies()
	assert.Len(b.t, cookies, 1)
	assert.True(b.t, cookies[0].HttpOnly)
}

// openRegistration lets a second account register.
func openRegistration(t *testing.T, url, token string) {
	t.Helper()
	setSetting(t, url, token, model.SettingRegistrationOpen, "true")
}

func TestAdminPagesRequireASession(t *testing.T) {
	url := testServer(t)
	register(t, url)

	anonymous := newBrowser(t, url)
	for _, path := range adminPages {
		response := anonymous.hop(http.MethodGet, path, nil)
		assert.Equal(t, http.StatusSeeOther, response.StatusCode, path)
		assert.Contains(t, response.Header.Get("Location"), "/login", path)
	}
}

func TestAdminPagesRefuseNonAdmins(t *testing.T) {
	url := testServer(t)
	admin := register(t, url)
	openRegistration(t, url, admin.Token)

	status := call(t, http.MethodPost, url+"/api/user/register", "", map[string]string{
		"email":    "dev@example.com",
		"username": "dev",
		"password": "correct-horse",
	}, nil)
	assert.Equal(t, http.StatusCreated, status)

	// The account is real and the session is real. It just isn't an admin.
	dev := newBrowser(t, url)
	dev.login("dev@example.com", "correct-horse")

	for _, path := range adminPages {
		code, body := dev.get(path)
		assert.Equal(t, http.StatusForbidden, code, path)
		assert.Contains(t, body, "administrators", path)
	}
}

func TestAdminPagesRenderForAnAdmin(t *testing.T) {
	url := testServer(t)
	admin := register(t, url)
	dispatch(t, url, admin.Token, "atkins test:build", "")

	operator := newBrowser(t, url)
	operator.login("ci@example.com", "correct-horse")

	for _, page := range []struct {
		path     string
		contains string
	}{
		{"/admin/repository", "github.com/titpetric/atkins"},
		{"/admin/allowlist", "Repository allowlist"},
		{"/admin/setting", model.SettingJobMaxDepth},
		{"/admin/user", "ci@example.com"},
		{"/admin/ssh-key", "Deploy keys"},
	} {
		status, body := operator.get(page.path)
		assert.Equal(t, http.StatusOK, status, page.path)
		assert.Contains(t, body, page.contains, page.path)
		assert.Contains(t, body, "csrf_token", page.path)
	}

	// /admin without a section lands on the repositories page.
	status, body := operator.get("/admin")
	assert.Equal(t, http.StatusOK, status)
	assert.Contains(t, body, "Repositories")
}

func TestLoginRefusesBadCredentials(t *testing.T) {
	url := testServer(t)
	register(t, url)

	visitor := newBrowser(t, url)
	response := visitor.hop(http.MethodPost, "/login", urlpkg.Values{
		"email":    {"ci@example.com"},
		"password": {"wrong"},
	})

	// The form returns a message and no redirect, and the handler sets
	// nothing on the way out.
	assert.Equal(t, http.StatusOK, response.StatusCode)
	assert.Empty(t, response.Cookies())

	response = visitor.hop(http.MethodGet, "/admin/user", nil)
	assert.Equal(t, http.StatusSeeOther, response.StatusCode)
}

// TestLoginPageSkipsTheFormWhenSignedIn: there is nothing to ask an
// admin who is already here.
func TestLoginPageSkipsTheFormWhenSignedIn(t *testing.T) {
	url := testServer(t)
	register(t, url)

	operator := newBrowser(t, url)
	operator.login("ci@example.com", "correct-horse")

	response := operator.hop(http.MethodGet, "/login", nil)
	assert.Equal(t, http.StatusSeeOther, response.StatusCode)
	assert.Contains(t, response.Header.Get("Location"), "/admin/repository")
}

func TestLogoutEndsTheSession(t *testing.T) {
	url := testServer(t)
	register(t, url)

	operator := newBrowser(t, url)
	operator.login("ci@example.com", "correct-horse")

	status, _ := operator.get("/admin/user")
	assert.Equal(t, http.StatusOK, status)

	_, _ = operator.post("/admin/user", "/logout", nil)

	response := operator.hop(http.MethodGet, "/admin/user", nil)
	assert.Equal(t, http.StatusSeeOther, response.StatusCode)
	assert.Contains(t, response.Header.Get("Location"), "/login")
}

// TestFormsRefuseAMissingOrWrongCSRFToken is the whole reason the token
// exists: a browser sends a session cookie with a cross-site post too.
func TestFormsRefuseAMissingOrWrongCSRFToken(t *testing.T) {
	url := testServer(t)
	register(t, url)

	operator := newBrowser(t, url)
	operator.login("ci@example.com", "correct-horse")

	for _, form := range []urlpkg.Values{
		{"pattern": {"github.com/**"}},
		{"pattern": {"github.com/**"}, "csrf_token": {"not-the-token"}},
	} {
		response := operator.hop(http.MethodPost, "/admin/allowlist", form)
		assert.Equal(t, http.StatusForbidden, response.StatusCode)
	}

	// And the store holds nothing new.
	var rules []map[string]any
	admin := loginToken(t, url)
	status := call(t, http.MethodGet, url+"/api/admin/repository", admin, nil, &rules)
	assert.Equal(t, http.StatusOK, status)
	assert.Empty(t, rules)
}

// TestFormsRefuseACrossOriginPost covers the browser that ignores
// SameSite.
func TestFormsRefuseACrossOriginPost(t *testing.T) {
	url := testServer(t)
	register(t, url)

	operator := newBrowser(t, url)
	operator.login("ci@example.com", "correct-horse")

	token := operator.csrf("/admin/allowlist")

	request, err := http.NewRequest(http.MethodPost, url+"/admin/allowlist",
		strings.NewReader(urlpkg.Values{"pattern": {"github.com/**"}, "csrf_token": {token}}.Encode()))
	assert.NoError(t, err)
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("Origin", "https://evil.example.com")

	client := &http.Client{
		Jar:           operator.client.Jar,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
	response, err := client.Do(request)
	assert.NoError(t, err)
	defer response.Body.Close()

	assert.Equal(t, http.StatusBadRequest, response.StatusCode)
}

func TestAllowlistFormLifecycle(t *testing.T) {
	url := testServer(t)
	register(t, url)

	operator := newBrowser(t, url)
	operator.login("ci@example.com", "correct-horse")

	status, body := operator.post("/admin/allowlist", "/admin/allowlist", urlpkg.Values{
		"pattern":     {"github.com/titpetric/*"},
		"description": {"our repositories"},
	})
	assert.Equal(t, http.StatusOK, status)
	assert.Contains(t, body, "github.com/titpetric/*")
	assert.Contains(t, body, "our repositories")
	assert.Contains(t, body, "active")

	ruleID := attributeValue(t, body, `action="/admin/allowlist/`)

	// The page refuses a duplicate, and says so.
	_, body = operator.post("/admin/allowlist", "/admin/allowlist", urlpkg.Values{
		"pattern": {"github.com/titpetric/*"},
	})
	assert.Contains(t, body, "already exists")

	_, body = operator.post("/admin/allowlist", "/admin/allowlist/"+ruleID, urlpkg.Values{"action": {"disable"}})
	assert.Contains(t, body, "Rule disabled")
	assert.Contains(t, body, "disabled</span>")

	_, body = operator.post("/admin/allowlist", "/admin/allowlist/"+ruleID, urlpkg.Values{"action": {"enable"}})
	assert.Contains(t, body, "Rule enabled")

	_, body = operator.post("/admin/allowlist", "/admin/allowlist/"+ruleID, urlpkg.Values{"action": {"remove"}})
	assert.Contains(t, body, "Rule removed")
	assert.Contains(t, body, "No rules yet")
	assert.NotContains(t, body, "/admin/allowlist/"+ruleID)
}

// TestAllowlistPageEscapesAPattern proves the escaping end to end: the
// test writes the pattern through the API and the page renders it.
func TestAllowlistPageEscapesAPattern(t *testing.T) {
	url := testServer(t)
	admin := register(t, url)

	allowRepository(t, url, admin.Token, `<script>alert(1)</script>`)

	operator := newBrowser(t, url)
	operator.login("ci@example.com", "correct-horse")

	status, body := operator.get("/admin/allowlist")
	assert.Equal(t, http.StatusOK, status)
	assert.NotContains(t, body, "<script>alert(1)</script>")
	assert.Contains(t, body, "&lt;script&gt;")
}

func TestSettingFormSetsAndResets(t *testing.T) {
	url := testServer(t)
	register(t, url)

	operator := newBrowser(t, url)
	operator.login("ci@example.com", "correct-horse")

	status, body := operator.post("/admin/setting", "/admin/setting", urlpkg.Values{
		"name":   {model.SettingRepositoryPolicy},
		"value":  {model.PolicyAllowlist},
		"action": {"set"},
	})
	assert.Equal(t, http.StatusOK, status)
	assert.Contains(t, body, "overridden")
	assert.Contains(t, body, `<option value="allowlist" selected`)

	// The allowlist page now warns that nothing can run.
	_, body = operator.get("/admin/allowlist")
	assert.Contains(t, body, "nothing will run")

	// The page reports a value the registry refuses, and stores nothing.
	_, body = operator.post("/admin/setting", "/admin/setting", urlpkg.Values{
		"name":   {model.SettingJobMaxDepth},
		"value":  {"many"},
		"action": {"set"},
	})
	assert.Contains(t, body, "whole number")

	_, body = operator.post("/admin/setting", "/admin/setting", urlpkg.Values{
		"name":   {model.SettingRepositoryPolicy},
		"action": {"reset"},
	})
	assert.Contains(t, body, "reset to its default")
	assert.NotContains(t, body, "overridden")
}

func TestUserFormTogglesFlags(t *testing.T) {
	url := testServer(t)
	admin := register(t, url)
	openRegistration(t, url, admin.Token)

	var second tokenResponse
	status := call(t, http.MethodPost, url+"/api/user/register", "", map[string]string{
		"email":    "dev@example.com",
		"username": "dev",
		"password": "correct-horse",
	}, &second)
	assert.Equal(t, http.StatusCreated, status)

	operator := newBrowser(t, url)
	operator.login("ci@example.com", "correct-horse")

	_, body := operator.post("/admin/user", "/admin/user/"+second.UserID, urlpkg.Values{
		"field": {"admin"},
		"value": {"true"},
	})
	assert.Contains(t, body, "dev: admin is now true")

	_, body = operator.post("/admin/user", "/admin/user/"+second.UserID, urlpkg.Values{
		"field": {"active"},
		"value": {"false"},
	})
	assert.Contains(t, body, "dev: active is now false")

	// The door stops the deactivated account.
	status = call(t, http.MethodGet, url+"/api/user/whoami", second.Token, nil, nil)
	assert.Equal(t, http.StatusForbidden, status)
}

// TestUserFormRefusesToRemoveTheLastAdmin checks both halves: the page
// says so before the click, and the handler refuses it anyway.
func TestUserFormRefusesToRemoveTheLastAdmin(t *testing.T) {
	url := testServer(t)
	admin := register(t, url)

	operator := newBrowser(t, url)
	operator.login("ci@example.com", "correct-horse")

	status, body := operator.get("/admin/user")
	assert.Equal(t, http.StatusOK, status)
	assert.Contains(t, body, "the last active admin")
	assert.Contains(t, body, "disabled title=")

	// A disabled button is a courtesy; the control lives in the API.
	_, body = operator.post("/admin/user", "/admin/user/"+admin.UserID, urlpkg.Values{
		"field": {"admin"},
		"value": {"false"},
	})
	assert.Contains(t, body, "refusing to remove the last active admin")

	var whoami map[string]any
	status = call(t, http.MethodGet, url+"/api/user/whoami", admin.Token, nil, &whoami)
	assert.Equal(t, http.StatusOK, status)
	assert.Equal(t, true, whoami["is_admin"])
}

func TestSSHKeyFormLifecycle(t *testing.T) {
	url := testServer(t)
	register(t, url)

	operator := newBrowser(t, url)
	operator.login("ci@example.com", "correct-horse")

	private := testSSHKey(t)

	status, body := operator.post("/admin/ssh-key", "/admin/ssh-key", urlpkg.Values{
		"name":        {"github"},
		"host":        {"github.com"},
		"private_key": {private},
	})
	assert.Equal(t, http.StatusOK, status)
	assert.Contains(t, body, "SHA256:")
	assert.Contains(t, body, "ssh-ed25519")

	// The page renders the key the form just submitted, and none of what
	// it renders is the private half. (The empty textarea's placeholder
	// mentions the PEM header, so the check looks at the body of the key
	// and skips its banner.)
	base64Lines := 0
	for _, line := range strings.Split(strings.TrimSpace(private), "\n") {
		if len(line) > 20 && !strings.HasPrefix(line, "-----") {
			assert.NotContains(t, body, line)
			base64Lines++
		}
	}
	assert.NotZero(t, base64Lines, "the fixture key has no body to look for")

	keyID := attributeValue(t, body, `action="/admin/ssh-key/`)

	_, body = operator.post("/admin/ssh-key", "/admin/ssh-key/"+keyID, urlpkg.Values{"action": {"deactivate"}})
	assert.Contains(t, body, "Key deactivated")
	assert.Contains(t, body, "disabled</span>")

	_, body = operator.post("/admin/ssh-key", "/admin/ssh-key/"+keyID, urlpkg.Values{"action": {"activate"}})
	assert.Contains(t, body, "Key activated")

	_, body = operator.post("/admin/ssh-key", "/admin/ssh-key/"+keyID, urlpkg.Values{"action": {"remove"}})
	assert.Contains(t, body, "Key removed")
	assert.Contains(t, body, "No deploy keys")

	// The form refuses a key that does not parse.
	_, body = operator.post("/admin/ssh-key", "/admin/ssh-key", urlpkg.Values{
		"name":        {"broken"},
		"private_key": {"not a key at all"},
	})
	assert.Contains(t, body, "not a usable ssh private key")
}

func TestTriggerFormQueuesAJob(t *testing.T) {
	url := testServer(t)
	admin := register(t, url)

	first := dispatch(t, url, admin.Token, "atkins test:build", "")

	operator := newBrowser(t, url)
	operator.login("ci@example.com", "correct-horse")

	response := operator.hop(http.MethodPost, "/admin/repository/"+first.RepositoryID+"/trigger", urlpkg.Values{
		"job": {"analyze"},
		// An agent cd's into this, so the handler has to drop the escape
		// here, the same as /api/dispatch.
		"working_directory": {"../../etc"},
		"csrf_token":        {operator.csrf("/admin/repository")},
	})
	assert.Equal(t, http.StatusSeeOther, response.StatusCode)

	// Straight to the job page: watching it run is the next thing
	// anybody wants.
	location := response.Header.Get("Location")
	assert.True(t, strings.HasPrefix(location, "/job/"), location)

	var job map[string]any
	status := call(t, http.MethodGet, url+"/api"+location, admin.Token, nil, &job)
	assert.Equal(t, http.StatusOK, status)
	assert.Equal(t, "atkins analyze", job["command"])
	assert.Equal(t, model.JobStatusPending, job["status"])
	assert.Equal(t, "", job["working_directory"])
}

// TestTriggerFormObeysTheAllowlist: the page hides the button for a
// refused repository, and the server still refuses a post sent anyway.
func TestTriggerFormObeysTheAllowlist(t *testing.T) {
	url := testServer(t)
	admin := register(t, url)

	first := dispatch(t, url, admin.Token, "atkins test:build", "")
	setSetting(t, url, admin.Token, model.SettingRepositoryPolicy, model.PolicyAllowlist)

	operator := newBrowser(t, url)
	operator.login("ci@example.com", "correct-horse")

	status, body := operator.get("/admin/repository")
	assert.Equal(t, http.StatusOK, status)
	assert.Contains(t, body, "the allowlist refuses this repository")
	assert.NotContains(t, body, "/admin/repository/"+first.RepositoryID+"/trigger")

	_, body = operator.post("/admin/repository", "/admin/repository/"+first.RepositoryID+"/trigger", urlpkg.Values{
		"job": {"analyze"},
	})
	assert.Contains(t, body, "not on the allowlist")
}

// TestRepositoryPageEscapesAJobCommand: a command is whatever a client
// dispatched, and this page renders it.
func TestRepositoryPageEscapesAJobCommand(t *testing.T) {
	url := testServer(t)
	admin := register(t, url)

	dispatch(t, url, admin.Token, `atkins <script>alert(1)</script>`, "")

	operator := newBrowser(t, url)
	operator.login("ci@example.com", "correct-horse")

	status, body := operator.get("/admin/repository")
	assert.Equal(t, http.StatusOK, status)
	assert.NotContains(t, body, "<script>alert(1)</script>")
	assert.Contains(t, body, "&lt;script&gt;")
}

// TestJobPagesStayOpen guards the one thing the admin gate must not
// cover: atkins prints a job URL and a person pastes it into a browser.
//
// "Open" means the page requires no *session*, and not that anything
// goes. A private instance still requires the view token from the
// printed URL; what must never happen is a job page redirecting a
// visitor to /login.
func TestJobPagesStayOpen(t *testing.T) {
	url := testServer(t)
	admin := register(t, url)

	job := dispatch(t, url, admin.Token, "atkins test:build", "")

	anonymous := newBrowser(t, url)

	status, _ := anonymous.get("/job/" + job.JobID + "?t=" + job.ViewToken)
	assert.Equal(t, http.StatusOK, status)

	// Without the token the page refuses on its own terms: a 403 about
	// the link, and no redirect into the admin login.
	status, body := anonymous.get("/job/" + job.JobID)
	assert.Equal(t, http.StatusForbidden, status)
	assert.NotContains(t, body, "/login")

	// The front page answers and does not refuse: a health check probes
	// it, and a private instance simply lists nothing.
	status, _ = anonymous.get("/")
	assert.Equal(t, http.StatusOK, status)
}

// loginToken returns an admin API token for assertions that are easier
// to make over JSON than by reading a page.
func loginToken(t *testing.T, base string) string {
	t.Helper()

	var token tokenResponse
	status := call(t, http.MethodPost, base+"/api/user/login", "", map[string]string{
		"email":    "ci@example.com",
		"password": "correct-horse",
	}, &token)
	assert.Equal(t, http.StatusOK, status)

	return token.Token
}

// attributeValue pulls the id out of the first form action matching a
// prefix; these tests find the row they just created that way.
func attributeValue(t *testing.T, body, prefix string) string {
	t.Helper()

	index := strings.Index(body, prefix)
	assert.GreaterOrEqual(t, index, 0, prefix)

	rest := body[index+len(prefix):]
	end := strings.IndexAny(rest, `"/`)
	assert.Greater(t, end, 0)

	return rest[:end]
}

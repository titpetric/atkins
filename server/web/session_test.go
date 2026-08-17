package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/titpetric/atkins/server/model"
)

// signedHandlers returns Handlers that can sign cookies.
func signedHandlers(t *testing.T) *Handlers {
	t.Helper()

	handlers, err := NewHandlers(Options{SigningKey: "test-signing-key"})
	require.NoError(t, err)

	return handlers
}

func TestCookieValueRoundTrips(t *testing.T) {
	handlers := signedHandlers(t)

	id, err := handlers.sessionID(handlers.cookieValue("01ARZ3NDEKTSV4RRFFQ69G5FAV"))
	require.NoError(t, err)
	assert.Equal(t, "01ARZ3NDEKTSV4RRFFQ69G5FAV", id)
}

// TestCookieValueRefusesAnUnsignedID is the point of signing it. A ULID
// is time-ordered and partly guessable, so a cookie that were only the
// session id would let somebody walk onto a session by writing one down.
func TestCookieValueRefusesAnUnsignedID(t *testing.T) {
	handlers := signedHandlers(t)

	for _, value := range []string{
		"",
		"01ARZ3NDEKTSV4RRFFQ69G5FAV",
		"01ARZ3NDEKTSV4RRFFQ69G5FAV.",
		"01ARZ3NDEKTSV4RRFFQ69G5FAV.not-a-signature",
		".signature",
	} {
		_, err := handlers.sessionID(value)
		assert.Error(t, err, value)
	}

	// A signature for one session does not carry another.
	forged := "01ARZ3NDEKTSV4RRFFQ69G5FAW." + strings.SplitN(handlers.cookieValue("01ARZ3NDEKTSV4RRFFQ69G5FAV"), ".", 2)[1]
	_, err := handlers.sessionID(forged)
	assert.Error(t, err)
}

// TestSigningKeyIsRequired: HMAC under an empty key is a signature
// anybody can compute, so a server with no key must trust no cookie.
func TestSigningKeyIsRequired(t *testing.T) {
	handlers, err := NewHandlers(Options{})
	require.NoError(t, err)

	request := httptest.NewRequest(http.MethodGet, "/admin/user", nil)
	request.AddCookie(&http.Cookie{Name: sessionCookie, Value: handlers.cookieValue("session-1")})

	_, err = handlers.authenticate(request)
	assert.ErrorIs(t, err, errNoSession)
}

// TestCSRFTokenIsScopedToItsSession stops a cookie value being replayed
// as a CSRF token, and one session's token being used against another.
func TestCSRFTokenIsScopedToItsSession(t *testing.T) {
	handlers := signedHandlers(t)

	cookie := handlers.sign("cookie", "session-1")
	csrf := handlers.sign("csrf", "session-1")

	assert.NotEqual(t, cookie, csrf)
	assert.NotEqual(t, csrf, handlers.sign("csrf", "session-2"))
}

func TestAdminRedirectsAnAnonymousVisitorToLogin(t *testing.T) {
	handlers := signedHandlers(t)

	recorder := httptest.NewRecorder()
	handler := handlers.admin(func(http.ResponseWriter, *http.Request, *session) {
		t.Fatal("the page ran without a session")
	})
	handler(recorder, httptest.NewRequest(http.MethodGet, "/admin/user?page=2", nil))

	assert.Equal(t, http.StatusSeeOther, recorder.Code)
	// Where they were going is preserved, so signing in lands them there.
	assert.Contains(t, recorder.Header().Get("Location"), "next=%2Fadmin%2Fuser%3Fpage%3D2")
}

func TestSubmitRefusesACrossOriginPost(t *testing.T) {
	handlers := signedHandlers(t)

	request := httptest.NewRequest(http.MethodPost, "/admin/allowlist", strings.NewReader("pattern=**"))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("Origin", "https://evil.example.com")

	assert.Error(t, handlers.parseForm(httptest.NewRecorder(), request))
}

func TestParseFormAcceptsSameOriginAndNoOrigin(t *testing.T) {
	handlers := signedHandlers(t)

	form := func(origin string) *http.Request {
		request := httptest.NewRequest(http.MethodPost, "/admin/allowlist", strings.NewReader("pattern=**"))
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		if origin != "" {
			request.Header.Set("Origin", origin)
		}
		return request
	}

	// curl sends no Origin at all, and a request without one is not a
	// cross-site request.
	assert.NoError(t, handlers.parseForm(httptest.NewRecorder(), form("")))
	assert.NoError(t, handlers.parseForm(httptest.NewRecorder(), form("http://example.com")))
}

func TestSafeNextStaysOnThisServer(t *testing.T) {
	assert.Equal(t, "/admin/user", safeNext("/admin/user"))
	assert.Equal(t, "/admin/project", safeNext(""))

	// An open redirect on a login page is how a phishing link borrows a
	// domain's good name.
	assert.Equal(t, "/admin/project", safeNext("//evil.example.com/"))
	assert.Equal(t, "/admin/project", safeNext("https://evil.example.com/"))
}

func TestIsTLS(t *testing.T) {
	plain := httptest.NewRequest(http.MethodGet, "/", nil)
	assert.False(t, isTLS(plain))

	// A proxy that terminated TLS says so, and the cookie should be
	// Secure even though this hop is not.
	proxied := httptest.NewRequest(http.MethodGet, "/", nil)
	proxied.Header.Set("X-Forwarded-Proto", "https")
	assert.True(t, isTLS(proxied))
}

func TestSetSessionCookieIsHardened(t *testing.T) {
	handlers := signedHandlers(t)

	recorder := httptest.NewRecorder()
	handlers.setSessionCookie(recorder, httptest.NewRequest(http.MethodGet, "/", nil), &model.Session{ID: "session-1"})

	cookies := recorder.Result().Cookies()
	require.Len(t, cookies, 1)

	cookie := cookies[0]
	assert.Equal(t, sessionCookie, cookie.Name)
	assert.True(t, cookie.HttpOnly)
	assert.Equal(t, http.SameSiteLaxMode, cookie.SameSite)
	// Not over plain http: a Secure cookie is never sent back, which
	// would make the local instance impossible to log into.
	assert.False(t, cookie.Secure)

	secure := httptest.NewRecorder()
	tls := httptest.NewRequest(http.MethodGet, "/", nil)
	tls.Header.Set("X-Forwarded-Proto", "https")
	handlers.setSessionCookie(secure, tls, &model.Session{ID: "session-1"})
	require.Len(t, secure.Result().Cookies(), 1)
	assert.True(t, secure.Result().Cookies()[0].Secure)
}

package web

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/titpetric/atkins/server/model"
	"github.com/titpetric/atkins/server/storage"
)

// sessionCookie carries a signed reference to a row in the session
// table. It is not a second credential: the session it names is the one
// `atkins --login` creates, and revoking it logs the browser out of the
// admin pages exactly as `atkins --logout` logs the CLI out.
const sessionCookie = "atkins_session"

// csrfField is the hidden input every state-changing form carries.
const csrfField = "csrf_token"

// maxFormBytes bounds a submitted form. A deploy key is the largest
// thing posted here and it is measured in kilobytes.
const maxFormBytes = 1 << 20

// errNoSession means the request carried no usable session cookie.
var errNoSession = errors.New("no session")

// session is the authenticated state a page handler runs with.
type session struct {
	User    *model.User
	Session *model.Session

	// CSRF is the token this session's forms must echo back.
	CSRF string
}

// sign returns the HMAC of value under the server signing key, scoped
// by purpose so a cookie value can never be replayed as a CSRF token.
func (h *Handlers) sign(purpose, value string) string {
	mac := hmac.New(sha256.New, []byte(h.signingKey))
	mac.Write([]byte(purpose))
	mac.Write([]byte{0})
	mac.Write([]byte(value))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

// cookieValue is the session id followed by its signature.
//
// The id alone would not do. A ULID is time-ordered and partly
// guessable, so an unsigned cookie would let somebody walk onto another
// operator's session; the signature makes the cookie unforgeable
// without the signing key, and keeps the session row itself the only
// source of truth about whether the session is still live.
func (h *Handlers) cookieValue(sessionID string) string {
	return sessionID + "." + h.sign("cookie", sessionID)
}

// sessionID validates a cookie value and returns the session it names.
func (h *Handlers) sessionID(value string) (string, error) {
	id, signature, ok := strings.Cut(value, ".")
	if !ok || id == "" {
		return "", errNoSession
	}
	if subtle.ConstantTimeCompare([]byte(signature), []byte(h.sign("cookie", id))) != 1 {
		return "", errNoSession
	}
	return id, nil
}

// authenticate resolves the session cookie into a user.
//
// A missing signing key fails closed: HMAC under an empty key is a
// signature anybody can compute, so a server without one has no way to
// tell a real cookie from a made-up one.
func (h *Handlers) authenticate(r *http.Request) (*session, error) {
	if h.signingKey == "" || h.sessions == nil || h.users == nil {
		return nil, errNoSession
	}

	cookie, err := r.Cookie(sessionCookie)
	if err != nil {
		return nil, errNoSession
	}

	id, err := h.sessionID(cookie.Value)
	if err != nil {
		return nil, err
	}

	stored, err := h.sessions.Get(r.Context(), id)
	if err != nil {
		return nil, errNoSession
	}
	if stored.RevokedAt != nil {
		return nil, errNoSession
	}
	if stored.ExpiresAt != nil && time.Now().After(*stored.ExpiresAt) {
		return nil, errNoSession
	}

	user, err := h.users.Get(r.Context(), stored.UserID)
	if err != nil || !user.IsActive {
		return nil, errNoSession
	}

	return &session{User: user, Session: stored, CSRF: h.sign("csrf", stored.ID)}, nil
}

// setSessionCookie writes the cookie for a freshly created session.
//
// Secure is set only over TLS: a cookie marked Secure is never sent
// back over http, which would make the local instance on :3200
// impossible to log into.
func (h *Handlers) setSessionCookie(w http.ResponseWriter, r *http.Request, stored *model.Session) {
	expires := time.Now().Add(storage.DefaultSessionTTL)
	if stored.ExpiresAt != nil {
		expires = *stored.ExpiresAt
	}

	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    h.cookieValue(stored.ID),
		Path:     "/",
		Expires:  expires,
		MaxAge:   int(time.Until(expires).Seconds()),
		HttpOnly: true,
		Secure:   isTLS(r),
		SameSite: http.SameSiteLaxMode,
	})
}

// clearSessionCookie expires the cookie in the browser.
func (h *Handlers) clearSessionCookie(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   isTLS(r),
		SameSite: http.SameSiteLaxMode,
	})
}

// isTLS reports whether the request reached the server over TLS,
// including through a proxy that terminated it.
func isTLS(r *http.Request) bool {
	if r.TLS != nil {
		return true
	}
	return strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
}

// pageHandler is a page that runs with an authenticated admin session.
type pageHandler func(http.ResponseWriter, *http.Request, *session)

// admin gates a page on an admin session.
//
// An unauthenticated visitor is sent to the login form rather than
// shown a 401 they can do nothing with. An authenticated non-admin gets
// a 403: they are somebody, just not somebody with these pages.
func (h *Handlers) admin(next pageHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		current, err := h.authenticate(r)
		if err != nil {
			http.Redirect(w, r, "/login?next="+url.QueryEscape(r.URL.RequestURI()), http.StatusSeeOther)
			return
		}
		if !current.User.IsAdmin {
			h.fail(w, r, http.StatusForbidden, errors.New("these pages are for administrators"))
			return
		}

		next(w, r, current)
	}
}

// submit gates a form post on an admin session, a same-origin request
// and a matching CSRF token.
//
// Three layers, because each covers what the others miss. SameSite=Lax
// stops a cross-site form post in any browser that honours it; the
// origin check stops one from a browser that doesn't; and the token
// stops a same-site injection that reaches the form but cannot read the
// session's HMAC.
func (h *Handlers) submit(next pageHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		current, err := h.authenticate(r)
		if err != nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		if !current.User.IsAdmin {
			h.fail(w, r, http.StatusForbidden, errors.New("these pages are for administrators"))
			return
		}

		if err := h.parseForm(w, r); err != nil {
			h.fail(w, r, http.StatusBadRequest, err)
			return
		}
		if subtle.ConstantTimeCompare([]byte(r.PostFormValue(csrfField)), []byte(current.CSRF)) != 1 {
			h.fail(w, r, http.StatusForbidden, errors.New("this form has expired; reload the page and try again"))
			return
		}

		next(w, r, current)
	}
}

// parseForm reads a bounded form body and refuses a cross-origin post.
//
// The origin check is "reject a mismatch" rather than "require a
// match": curl and older browsers send no Origin at all, and a request
// with no Origin is not a cross-site one.
func (h *Handlers) parseForm(w http.ResponseWriter, r *http.Request) error {
	if origin := r.Header.Get("Origin"); origin != "" {
		parsed, err := url.Parse(origin)
		if err != nil || !strings.EqualFold(parsed.Host, r.Host) {
			return errors.New("cross-origin form submission refused")
		}
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxFormBytes)
	return r.ParseForm()
}

// LoginForm renders the login page.
func (h *Handlers) LoginForm(w http.ResponseWriter, r *http.Request) {
	// Already signed in: nothing to ask for.
	if current, err := h.authenticate(r); err == nil && current.User.IsAdmin {
		http.Redirect(w, r, "/admin/repository", http.StatusSeeOther)
		return
	}

	h.render(w, r, "login.html", &loginPage{
		Next:  safeNext(r.URL.Query().Get("next")),
		Error: r.URL.Query().Get("error"),
	})
}

// loginPage is the view model for the login form.
type loginPage struct {
	// Next is where to go after a successful login. It is always a
	// path on this server; see safeNext.
	Next  string
	Error string
}

// Login exchanges an email and password for a session cookie.
func (h *Handlers) Login(w http.ResponseWriter, r *http.Request) {
	if h.signingKey == "" || h.users == nil || h.sessions == nil {
		h.fail(w, r, http.StatusServiceUnavailable, errors.New("this server has no signing key configured"))
		return
	}

	if err := h.parseForm(w, r); err != nil {
		h.fail(w, r, http.StatusBadRequest, err)
		return
	}

	next := safeNext(r.PostFormValue("next"))

	user, err := h.users.Authenticate(r.Context(), r.PostFormValue("email"), r.PostFormValue("password"))
	if err != nil {
		// Both failure modes are reported as one: the login page is
		// reachable by anyone, and it should not answer "does this
		// address have an account here".
		h.render(w, r, "login.html", &loginPage{
			Next:  next,
			Error: "That email and password do not match an account.",
		})
		return
	}

	stored, err := h.sessions.Create(r.Context(), user.ID, storage.SessionRequest{
		Hostname:   "browser",
		UserAgent:  r.UserAgent(),
		RemoteAddr: r.RemoteAddr,
	})
	if err != nil {
		h.fail(w, r, http.StatusInternalServerError, err)
		return
	}

	h.setSessionCookie(w, r, stored)
	http.Redirect(w, r, next, http.StatusSeeOther)
}

// Logout revokes the session the cookie names and clears it.
func (h *Handlers) Logout(w http.ResponseWriter, r *http.Request) {
	current, err := h.authenticate(r)
	if err == nil {
		if err := h.parseForm(w, r); err != nil {
			h.fail(w, r, http.StatusBadRequest, err)
			return
		}
		if subtle.ConstantTimeCompare([]byte(r.PostFormValue(csrfField)), []byte(current.CSRF)) != 1 {
			h.fail(w, r, http.StatusForbidden, errors.New("this form has expired; reload the page and try again"))
			return
		}

		// Revoking the row is what makes this a logout rather than a
		// cookie deletion: an access token minted from the same session
		// stops working too.
		if err := h.sessions.Revoke(r.Context(), current.Session.ID); err != nil {
			h.fail(w, r, http.StatusInternalServerError, err)
			return
		}
	}

	h.clearSessionCookie(w, r)
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

// safeNext confines a post-login redirect to this server. An open
// redirect on a login page is how a phishing link borrows a domain's
// good name.
func safeNext(next string) string {
	if next == "" || !strings.HasPrefix(next, "/") || strings.HasPrefix(next, "//") {
		return "/admin/repository"
	}
	return next
}

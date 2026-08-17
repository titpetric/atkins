package web

import (
	"errors"
	"net/http"
	"strings"

	"github.com/titpetric/atkins/server/model"
	"github.com/titpetric/atkins/server/storage"
)

// A fresh instance has nobody who can sign into it, and until somebody
// can, every page it serves is a page saying so. `atkins --register
// https://host` has always been the way through that, which is fine for
// whoever installed the server and useless to anyone who was handed a
// URL.
//
// /setup is the same bootstrap through the front door. It exists only
// while the user table is empty, it creates the account the same storage
// call registration does — so the first account becomes the admin by the
// same rule, not by a second one written here — and it signs the browser
// in rather than leaving somebody at a login form with credentials they
// have just invented.
//
// It closes behind itself. Once an account exists the page is gone: the
// bootstrap is open because there is nobody to ask, and the moment there
// is somebody, asking them is exactly what a new account should require.

// setupPage is the view model for the first-run form.
type setupPage struct {
	// Values the visitor already typed, echoed back so a rejected form
	// does not ask for everything a second time. The password is never
	// among them.
	Username string
	Email    string
	FullName string

	Error string
}

// minPasswordLength is the shortest password setup accepts. It matches
// the API's rule, which is the point: this is the same registration.
const minPasswordLength = 8

// needsSetup reports whether the instance has no human accounts yet.
//
// Agents do not count, the same way they do not count for registration:
// an agent enrolling before any person must not close the door on the
// person who was going to open it.
func (h *Handlers) needsSetup(r *http.Request) bool {
	if h.users == nil {
		return false
	}

	count, err := h.users.Count(r.Context())
	if err != nil {
		// A database that cannot be counted is not an instance to invite
		// somebody to claim.
		return false
	}

	return count == 0
}

// SetupForm renders the first-run form, or sends a visitor on when there
// is nothing left to set up.
func (h *Handlers) SetupForm(w http.ResponseWriter, r *http.Request) {
	if !h.needsSetup(r) {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	h.render(w, r, setupView(&setupPage{}))
}

// Setup creates the first account and signs the browser in as it.
//
// The emptiness of the user table is checked again here rather than
// trusted from the form: two people opening /setup on a new instance
// should produce one admin and one "somebody got there first", not two
// admins because both pages were rendered before either was submitted.
func (h *Handlers) Setup(w http.ResponseWriter, r *http.Request) {
	if h.signingKey == "" || h.users == nil || h.sessions == nil {
		h.fail(w, r, http.StatusServiceUnavailable, errors.New("this server has no signing key configured"))
		return
	}

	if err := h.parseForm(w, r); err != nil {
		h.fail(w, r, http.StatusBadRequest, err)
		return
	}

	if !h.needsSetup(r) {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	page := &setupPage{
		Username: strings.TrimSpace(r.PostFormValue("username")),
		Email:    strings.TrimSpace(r.PostFormValue("email")),
		FullName: strings.TrimSpace(r.PostFormValue("full_name")),
	}
	password := r.PostFormValue("password")

	if err := validateSetup(page, password, r.PostFormValue("password_confirmation")); err != nil {
		page.Error = err.Error()
		h.render(w, r, setupView(page))
		return
	}

	user, err := h.users.Create(r.Context(), storage.CreateRequest{
		Email:    page.Email,
		Username: page.Username,
		FullName: page.FullName,
		Password: password,
	})
	if err != nil {
		switch {
		case errors.Is(err, model.ErrEmailTaken), errors.Is(err, model.ErrUsernameTaken):
			page.Error = err.Error()
		default:
			page.Error = "That account could not be created: " + err.Error()
		}
		h.render(w, r, setupView(page))
		return
	}

	session, err := h.sessions.Create(r.Context(), user.ID, storage.SessionRequest{
		Hostname:   "browser",
		UserAgent:  r.UserAgent(),
		RemoteAddr: r.RemoteAddr,
	})
	if err != nil {
		// The account exists; only the sign-in failed. Sending them to
		// the login form is a working way forward rather than a dead end.
		h.fail(w, r, http.StatusInternalServerError, err)
		return
	}

	h.setSessionCookie(w, r, session)

	// Straight to the projects page. Somebody who has just claimed an
	// empty instance wants to put something in it.
	http.Redirect(w, r, "/admin/project", http.StatusSeeOther)
}

// validateSetup checks the form the way a person reads it: one problem
// at a time, named, and in the order the fields appear.
func validateSetup(page *setupPage, password, confirmation string) error {
	switch {
	case page.Username == "":
		return errors.New("A username is required.")
	case page.Email == "":
		return errors.New("An email address is required.")
	case len(password) < minPasswordLength:
		return errors.New("The password must be at least 8 characters.")
	case password != confirmation:
		return errors.New("The two passwords do not match.")
	}

	return nil
}

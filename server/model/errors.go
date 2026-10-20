package model

import (
	"errors"
)

// The storage layer returns these errors. Handlers map them onto status
// codes; the API reports anything not listed here as a 5xx.
var (
	// ErrInvalidCredentials reports an email/password pair that does
	// not match an active user. "No such user" and "wrong password"
	// share this one error so nobody can probe the login endpoint for
	// account names.
	ErrInvalidCredentials = errors.New("invalid credentials")

	// ErrUserInactive reports a user that exists but is inactive. Login
	// refuses that user.
	ErrUserInactive = errors.New("user is not active")

	// ErrEmailTaken reports a registration with an existing email.
	ErrEmailTaken = errors.New("email already registered")

	// ErrUsernameTaken reports a registration with an existing username.
	ErrUsernameTaken = errors.New("username already taken")

	// ErrRegistrationClosed reports a registration attempt while the
	// registration setting is off and at least one user already exists.
	ErrRegistrationClosed = errors.New("registration is closed")

	// ErrSessionExpired reports a refresh token past its expiry. The
	// client has to log in again.
	ErrSessionExpired = errors.New("session expired")

	// ErrSessionRevoked reports a refresh token that belongs to a
	// logged-out session.
	ErrSessionRevoked = errors.New("session revoked")

	// ErrInvalidRepository reports a dispatch request that carries no
	// usable git remote.
	ErrInvalidRepository = errors.New("invalid repository")

	// ErrMaxDepthExceeded reports a job dispatching a child past the
	// nesting limit from the server settings. It's the backstop against
	// a pipeline that dispatches itself forever.
	ErrMaxDepthExceeded = errors.New("maximum job depth exceeded")

	// ErrRepositoryNotAllowed reports a repository that matches no rule
	// while the server runs an allowlist policy.
	ErrRepositoryNotAllowed = errors.New("repository is not on the allowlist")

	// ErrInvalidPattern reports an empty allowlist pattern.
	ErrInvalidPattern = errors.New("pattern is required")

	// ErrRuleNotFound reports an allowlist rule that does not exist, or
	// that a delete already removed.
	ErrRuleNotFound = errors.New("repository rule not found")

	// ErrSSHKeyNotFound reports an ssh key that does not exist.
	ErrSSHKeyNotFound = errors.New("ssh key not found")

	// ErrInvalidSSHKey reports a submitted key that is not a usable
	// private key.
	ErrInvalidSSHKey = errors.New("not a usable ssh private key")

	// ErrInvalidArtefactPath reports an upload that names a file which
	// could not sit inside the job's directory.
	ErrInvalidArtefactPath = errors.New("artefact path must be relative to the job directory")

	// ErrArtefactTooLarge reports an upload that runs past
	// artefact.max_size. The store discards the bytes written so far.
	ErrArtefactTooLarge = errors.New("artefact is larger than artefact.max_size")

	// ErrTooManyArtefacts reports a job that already stores
	// artefact.max_count files. This is the per-job backstop against a
	// pipeline that uploads in a loop.
	ErrTooManyArtefacts = errors.New("job has reached artefact.max_count")

	// ErrChecksumMismatch reports received bytes that do not hash to the
	// checksum the agent declared: something truncated or altered the
	// upload in flight.
	ErrChecksumMismatch = errors.New("artefact checksum does not match the uploaded bytes")

	// ErrArtefactNotFound reports an artefact that does not exist,
	// belongs to another job, or that retention already swept.
	ErrArtefactNotFound = errors.New("artefact not found")

	// ErrForbidden reports a signed-in user who lacks the permission
	// for the action.
	ErrForbidden = errors.New("insufficient permissions")

	// ErrLastAdmin reports a flag change that would leave the instance
	// with no active admin, and so with no way back in.
	ErrLastAdmin = errors.New("refusing to remove the last active admin; promote someone else first")

	// ErrInvalidAgentToken reports an agent enrolment that presents the
	// wrong token, or a server with no enrolment token set.
	ErrInvalidAgentToken = errors.New("invalid agent enrolment token")
)

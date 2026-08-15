package model

import "errors"

// Errors returned by the storage layer. Handlers map these onto status
// codes; anything not listed here is a 5xx as far as the API is concerned.
var (
	// ErrInvalidCredentials is returned when the email/password pair
	// does not match an active user. It is deliberately the same error
	// for "no such user" and "wrong password" so the login endpoint
	// cannot be used to enumerate accounts.
	ErrInvalidCredentials = errors.New("invalid credentials")

	// ErrUserInactive is returned when a user exists but has been
	// deactivated. Login is refused.
	ErrUserInactive = errors.New("user is not active")

	// ErrEmailTaken is returned when registering an existing email.
	ErrEmailTaken = errors.New("email already registered")

	// ErrUsernameTaken is returned when registering an existing username.
	ErrUsernameTaken = errors.New("username already taken")

	// ErrRegistrationClosed is returned when registration is disabled
	// and at least one user already exists.
	ErrRegistrationClosed = errors.New("registration is closed")

	// ErrSessionExpired is returned when a refresh token is past its
	// expiry. The client has to log in again.
	ErrSessionExpired = errors.New("session expired")

	// ErrSessionRevoked is returned when a refresh token belongs to a
	// session that was logged out.
	ErrSessionRevoked = errors.New("session revoked")

	// ErrInvalidRepository is returned when a dispatch request carries
	// no usable git remote.
	ErrInvalidRepository = errors.New("invalid repository")

	// ErrMaxDepthExceeded is returned when a job tries to dispatch a
	// child past the configured nesting limit. It's the backstop
	// against a pipeline that dispatches itself forever.
	ErrMaxDepthExceeded = errors.New("maximum job depth exceeded")

	// ErrRepositoryNotAllowed is returned when the server runs an
	// allowlist policy and no rule matches the repository.
	ErrRepositoryNotAllowed = errors.New("repository is not on the allowlist")

	// ErrInvalidPattern is returned for an empty allowlist pattern.
	ErrInvalidPattern = errors.New("pattern is required")

	// ErrRuleNotFound is returned when an allowlist rule does not
	// exist, or has already been deleted.
	ErrRuleNotFound = errors.New("repository rule not found")

	// ErrSSHKeyNotFound is returned when an ssh key does not exist.
	ErrSSHKeyNotFound = errors.New("ssh key not found")

	// ErrInvalidSSHKey is returned when a submitted key is not a
	// usable private key.
	ErrInvalidSSHKey = errors.New("not a usable ssh private key")

	// ErrInvalidArtefactPath is returned when an upload names a file
	// that could not be inside the job's directory.
	ErrInvalidArtefactPath = errors.New("artefact path must be relative to the job directory")

	// ErrArtefactTooLarge is returned when an upload runs past
	// artefact.max_size. The bytes written so far are discarded.
	ErrArtefactTooLarge = errors.New("artefact is larger than artefact.max_size")

	// ErrTooManyArtefacts is returned when a job has already stored
	// artefact.max_count files. It is the per-job backstop against a
	// pipeline that uploads in a loop.
	ErrTooManyArtefacts = errors.New("job has reached artefact.max_count")

	// ErrChecksumMismatch is returned when the bytes received do not
	// hash to the checksum the agent declared, which means the upload
	// was truncated or altered in flight.
	ErrChecksumMismatch = errors.New("artefact checksum does not match the uploaded bytes")

	// ErrArtefactNotFound is returned when an artefact does not exist,
	// belongs to another job, or has been swept by retention.
	ErrArtefactNotFound = errors.New("artefact not found")

	// ErrForbidden is returned when a user is authenticated but not
	// permitted to perform the action.
	ErrForbidden = errors.New("insufficient permissions")

	// ErrInvalidAgentToken is returned when agent enrolment presents
	// the wrong token, or the server has no enrolment token set.
	ErrInvalidAgentToken = errors.New("invalid agent enrolment token")
)

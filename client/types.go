package client

import "io"

// The payload types below mirror github.com/titpetric/atkins/server/api.
// They are re-declared rather than imported so the atkins CLI links
// against net/http alone; importing the server package would pull the
// platform, chi and sqlx trees into every atkins build.

// LoginRequest is the body of POST /api/user/login.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Hostname string `json:"hostname"`
}

// RegisterRequest is the body of POST /api/user/register.
type RegisterRequest struct {
	Email    string `json:"email"`
	Username string `json:"username"`
	FullName string `json:"full_name"`
	Password string `json:"password"`
}

// RefreshRequest is the body of POST /api/user/refreshToken.
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// LogoutRequest is the body of POST /api/user/logout.
type LogoutRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// TokenResponse is returned by login, register and refresh.
type TokenResponse struct {
	UserID       string `json:"user_id"`
	Username     string `json:"username"`
	Token        string `json:"token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresAt    int64  `json:"expires_at"`
}

// RepositoryPayload describes the checkout a run happens in.
//
// Ref is what the agent checks out: a branch, a tag, a commit sha or a
// fully qualified refname. The client fills it with the commit it is
// sitting on, because a run dispatched from a laptop should build that
// commit rather than whatever the branch has moved to by the time an
// agent picks the job up.
type RepositoryPayload struct {
	RemoteURL     string `json:"remote_url"`
	Ref           string `json:"ref,omitempty"`
	DefaultBranch string `json:"default_branch"`
}

// DispatchRequest is the body of POST /api/dispatch.
type DispatchRequest struct {
	Repository       RepositoryPayload `json:"repository"`
	WorkingDirectory string            `json:"working_directory"`
	Command          string            `json:"command"`
	ParentID         string            `json:"parent_id,omitempty"`
	Labels           []string          `json:"labels,omitempty"`
	Params           map[string]any    `json:"params,omitempty"`
	Artefacts        []string          `json:"artefacts,omitempty"`
}

// DispatchResponse is returned by POST /api/dispatch.
type DispatchResponse struct {
	JobID          string `json:"job_id"`
	ParentID       string `json:"parent_id,omitempty"`
	RootID         string `json:"root_id"`
	Depth          int64  `json:"depth"`
	RepositoryID   string `json:"repository_id"`
	RepositorySlug string `json:"repository_slug"`
	Status         string `json:"status"`

	// ViewToken opens the job page in a browser without a session. A
	// server that keeps jobs private returns one; a public one returns
	// nothing and the plain job URL opens.
	ViewToken string `json:"view_token,omitempty"`
}

// EnrolRequest is the body of POST /api/agent/enrol.
type EnrolRequest struct {
	Token   string   `json:"token"`
	AgentID string   `json:"agent_id"`
	Labels  []string `json:"labels,omitempty"`
}

// PolicyResponse is what an agent may run.
type PolicyResponse struct {
	Policy   string   `json:"policy"`
	Patterns []string `json:"patterns"`
}

// AgentSSHKey is a deploy key handed to an agent.
type AgentSSHKey struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Host        string `json:"host"`
	PrivateKey  string `json:"private_key"`
	KnownHosts  string `json:"known_hosts"`
	Fingerprint string `json:"fingerprint"`
}

// Repository policy values, mirroring server/model.
const (
	PolicyOpen      = "open"
	PolicyAllowlist = "allowlist"
)

// ClaimRequest is the body of POST /api/job/claim.
type ClaimRequest struct {
	AgentID string   `json:"agent_id"`
	Labels  []string `json:"labels,omitempty"`
}

// ClaimResponse is returned when an agent leases a job. The repository
// travels with the job so an agent can start cloning without a second
// round trip.
type ClaimResponse struct {
	Job        *Job        `json:"job"`
	Repository *Repository `json:"repository"`
}

// Repository is the checkout an agent has to reproduce.
type Repository struct {
	ID            string `json:"id"`
	Slug          string `json:"slug"`
	RemoteURL     string `json:"remote_url"`
	DefaultBranch string `json:"default_branch"`
}

// Job is the subset of a queued job an agent needs to run it. It
// mirrors the columns of server/model.Job that matter on this side.
type Job struct {
	ID               string `json:"id"`
	ParentID         string `json:"parent_id"`
	RootID           string `json:"root_id"`
	RepositoryID     string `json:"repository_id"`
	WorkingDirectory string `json:"working_directory"`
	Command          string `json:"command"`
	Ref              string `json:"ref"`
	CommitSHA        string `json:"commit_sha"`
	CloneDepth       int64  `json:"clone_depth"`
	Labels           string `json:"labels"`
	Params           string `json:"params"`
	Status           string `json:"status"`

	// ArtefactPaths are the comma separated globs the agent collects
	// after the command exits.
	ArtefactPaths string `json:"artefact_paths"`

	// Interactive means the command reads a terminal. The agent gives it
	// a pty and pumps the job's input queue into it instead of running
	// it with no stdin at all.
	Interactive bool `json:"interactive"`
}

// JobCheckoutRequest is the body of POST /api/job/{jobID}/checkout: what
// the agent actually put in the work tree.
type JobCheckoutRequest struct {
	Ref       string `json:"ref"`
	CommitSHA string `json:"commit_sha"`
}

// Artefact mirrors api.ArtefactView: one file a job produced.
type Artefact struct {
	ID          string `json:"id"`
	JobID       string `json:"job_id"`
	Path        string `json:"path"`
	Size        int64  `json:"size"`
	ContentType string `json:"content_type"`
	Checksum    string `json:"checksum"`
	CreatedAt   string `json:"created_at"`
	URL         string `json:"url"`
}

// ArtefactUpload is one file being pushed to the server.
type ArtefactUpload struct {
	// Path is the name the pipeline gave the file, relative to the
	// directory the job ran in.
	Path string

	// ContentType is the media type, guessed from the extension.
	ContentType string

	// Checksum is the SHA256 the agent computed while reading the
	// file. The server compares it against what arrives, so a
	// truncated upload is refused rather than stored.
	Checksum string

	// Content is the file. It is streamed rather than buffered: an
	// artefact is as large as the server's limit allows.
	Content io.Reader
}

// HeaderArtefactChecksum carries ArtefactUpload.Checksum, mirroring
// server/api.
const HeaderArtefactChecksum = "X-Atkins-Checksum"

// JobLogRequest is the body of POST /api/job/{jobID}/log.
type JobLogRequest struct {
	Stream  string `json:"stream"`
	Content string `json:"content"`
}

// Log stream names, mirroring server/storage.
const (
	StreamOutput = "output"
	StreamError  = "error"
)

// JobInputResponse is the body of GET /api/job/{jobID}/input: whatever
// has been typed at an interactive job since the agent last collected.
//
// The bytes are base64 because they are bytes — arrow keys, control
// characters, whatever a keyboard produced — and none of that survives
// being called a JSON string.
type JobInputResponse struct {
	Input string `json:"input,omitempty"`
}

// JobStatusRequest is the body of POST /api/job/{jobID}/status.
type JobStatusRequest struct {
	Status   string `json:"status"`
	ExitCode int64  `json:"exit_code"`
	Error    string `json:"error,omitempty"`
}

// Job status values reported by the CLI. They mirror the server's
// terminal statuses in server/model.
const (
	StatusPassed    = "passed"
	StatusFailed    = "failed"
	StatusTimeout   = "timeout"
	StatusCancelled = "cancelled"
)

// errorResponse is the platform error envelope.
type errorResponse struct {
	Error struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// Package client talks to an atkins CI/CD server.
//
// It covers the whole client side of the feature: `atkins --login
// https://domain` and `atkins --register` obtain a credential, and every
// later atkins run posts to /api/dispatch so the server knows which
// repository, which directory in its work tree, and which command ran.
//
// The package deliberately depends on net/http alone. Everything it
// sends and receives is declared in types.go rather than imported from
// server/api, so building the CLI does not pull in the server's
// database and routing dependencies.
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// UserAgent is sent with every request. The main package overwrites it
// with the build version, which is what a server log needs to tell an
// old client from a new one.
var UserAgent = "atkins"

// Client is an HTTP client for one atkins server.
type Client struct {
	server string
	http   *http.Client

	// store and credential are set for authenticated clients. The
	// store is written back when a refresh rotates the tokens.
	store      *Store
	credential *Credential
}

// DefaultTimeout bounds a single API call. Dispatch happens on the hot
// path of every atkins run, so it must never hang a build.
const DefaultTimeout = 10 * time.Second

// UploadTimeout bounds one artefact transfer. It is generous because an
// artefact is measured in megabytes rather than in fields, and nothing
// is waiting on it: the job has already finished by the time it runs.
const UploadTimeout = 5 * time.Minute

// New returns an unauthenticated client for a server. It is what the
// login and register flows use before a credential exists.
func New(server string) *Client {
	return &Client{
		server: NormalizeServer(server),
		http:   &http.Client{Timeout: configuredTimeout()},
	}
}

// Open returns an authenticated client for a server, or for the stored
// default server when server is empty.
//
// It returns ErrNotLoggedIn when there is no credential, which callers
// on the dispatch path treat as "this machine isn't attached to a CI
// server" rather than as a failure.
func Open(server string) (*Client, error) {
	store, err := LoadStore()
	if err != nil {
		return nil, err
	}

	credential, ok := store.Get(server)
	if !ok {
		return nil, ErrNotLoggedIn
	}

	return &Client{
		server:     credential.Server,
		http:       &http.Client{Timeout: configuredTimeout()},
		store:      store,
		credential: credential,
	}, nil
}

// Server returns the server URL the client talks to.
func (c *Client) Server() string {
	return c.server
}

// Username returns the logged-in username, if any.
func (c *Client) Username() string {
	if c.credential == nil {
		return ""
	}
	return c.credential.Username
}

// Login exchanges email and password for tokens and stores them.
func (c *Client) Login(ctx context.Context, req LoginRequest) (*Credential, error) {
	var token TokenResponse
	if err := c.do(ctx, http.MethodPost, "/api/user/login", req, &token, false); err != nil {
		return nil, err
	}
	return c.persist(token)
}

// Register creates an account and stores the credential it returns.
func (c *Client) Register(ctx context.Context, req RegisterRequest) (*Credential, error) {
	var token TokenResponse
	if err := c.do(ctx, http.MethodPost, "/api/user/register", req, &token, false); err != nil {
		return nil, err
	}
	return c.persist(token)
}

// Logout revokes the stored session and forgets the credential.
//
// The local credential is dropped even when the server call fails: a
// user asking to log out of an unreachable server should still end up
// logged out on this machine.
func (c *Client) Logout(ctx context.Context) error {
	if c.credential == nil {
		return ErrNotLoggedIn
	}

	req := LogoutRequest{RefreshToken: c.credential.RefreshToken}
	err := c.do(ctx, http.MethodPost, "/api/user/logout", req, nil, true)

	if c.store != nil {
		c.store.Remove(c.server)
		if saveErr := c.store.Save(); saveErr != nil {
			return saveErr
		}
	}
	c.credential = nil

	return err
}

// Refresh exchanges the refresh token for a new access token and
// persists the rotated pair.
func (c *Client) Refresh(ctx context.Context) error {
	if c.credential == nil {
		return ErrNotLoggedIn
	}

	var token TokenResponse
	req := RefreshRequest{RefreshToken: c.credential.RefreshToken}
	if err := c.do(ctx, http.MethodPost, "/api/user/refreshToken", req, &token, false); err != nil {
		return err
	}

	_, err := c.persist(token)
	return err
}

// Dispatch records a job for the current run and returns its ID.
func (c *Client) Dispatch(ctx context.Context, req DispatchRequest) (*DispatchResponse, error) {
	var response DispatchResponse
	if err := c.do(ctx, http.MethodPost, "/api/dispatch", req, &response, true); err != nil {
		return nil, err
	}
	return &response, nil
}

// ReportStatus settles a job into a terminal state.
func (c *Client) ReportStatus(ctx context.Context, jobID string, req JobStatusRequest) error {
	return c.do(ctx, http.MethodPost, "/api/job/"+jobID+"/status", req, nil, true)
}

// Claim leases the oldest pending job this agent can run.
//
// A nil job with a nil error means the queue held nothing for us, which
// is the normal outcome of a poll and not a condition worth logging.
func (c *Client) Claim(ctx context.Context, agentID string, labels []string) (*ClaimResponse, error) {
	var response ClaimResponse

	status, err := c.doStatus(ctx, http.MethodPost, "/api/job/claim", ClaimRequest{
		AgentID: agentID,
		Labels:  labels,
	}, &response, true)
	if err != nil {
		return nil, err
	}
	if status == http.StatusNoContent || response.Job == nil {
		return nil, nil
	}

	return &response, nil
}

// ReportCheckout records the ref and commit an agent checked out.
func (c *Client) ReportCheckout(ctx context.Context, jobID string, req JobCheckoutRequest) error {
	return c.do(ctx, http.MethodPost, "/api/job/"+jobID+"/checkout", req, nil, true)
}

// Heartbeat extends this agent's lease on a job.
func (c *Client) Heartbeat(ctx context.Context, jobID, agentID string) error {
	return c.do(ctx, http.MethodPost, "/api/job/"+jobID+"/heartbeat", ClaimRequest{AgentID: agentID}, nil, true)
}

// AppendLog records a chunk of job output.
func (c *Client) AppendLog(ctx context.Context, jobID, stream, content string) error {
	return c.do(ctx, http.MethodPost, "/api/job/"+jobID+"/log", JobLogRequest{
		Stream:  stream,
		Content: content,
	}, nil, true)
}

// UploadArtefact pushes one file a job produced.
//
// The body is the file itself. A multipart envelope would buy nothing
// here: there is one part, and the two fields around it fit in a query
// parameter and a header.
func (c *Client) UploadArtefact(ctx context.Context, jobID string, upload ArtefactUpload) (*Artefact, error) {
	authorization, err := c.authorization(ctx)
	if err != nil {
		return nil, err
	}

	target := c.server + "/api/job/" + jobID + "/artefact?path=" + url.QueryEscape(upload.Path)

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, target, upload.Content)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", contentTypeOrDefault(upload.ContentType))
	request.Header.Set("Accept", "application/json")
	request.Header.Set("User-Agent", UserAgent)
	request.Header.Set("Authorization", authorization)
	if upload.Checksum != "" {
		request.Header.Set(HeaderArtefactChecksum, upload.Checksum)
	}

	// Not c.http: its timeout is sized for the dispatch call on the hot
	// path of every run, and a 32MB artefact on a slow link is not that
	// request.
	response, err := (&http.Client{Timeout: UploadTimeout}).Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode >= http.StatusBadRequest {
		return nil, decodeAPIError(response)
	}

	var artefact Artefact
	if err := json.NewDecoder(response.Body).Decode(&artefact); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &artefact, nil
}

// Artefacts lists the files a job produced.
func (c *Client) Artefacts(ctx context.Context, jobID string) ([]Artefact, error) {
	var artefacts []Artefact
	if err := c.do(ctx, http.MethodGet, "/api/job/"+jobID+"/artefact", nil, &artefacts, true); err != nil {
		return nil, err
	}
	return artefacts, nil
}

// contentTypeOrDefault falls back to the media type the server would
// have picked anyway.
func contentTypeOrDefault(value string) string {
	if strings.TrimSpace(value) == "" {
		return "application/octet-stream"
	}
	return value
}

// JobURL is where a job is watched in a browser. It is the one thing
// atkins prints when it hands a run to a server.
func (c *Client) JobURL(jobID string) string {
	return c.server + "/job/" + jobID
}

// Enrol trades the shared agent token for agent credentials.
//
// The credentials are persisted like any login, so a restarted agent
// with the same identity resumes without re-enrolling.
func (c *Client) Enrol(ctx context.Context, req EnrolRequest) (*Credential, error) {
	var token TokenResponse
	if err := c.do(ctx, http.MethodPost, "/api/agent/enrol", req, &token, false); err != nil {
		return nil, err
	}
	return c.persist(token)
}

// Policy returns the repository policy the agent must enforce.
func (c *Client) Policy(ctx context.Context) (*PolicyResponse, error) {
	var policy PolicyResponse
	if err := c.do(ctx, http.MethodGet, "/api/agent/policy", nil, &policy, true); err != nil {
		return nil, err
	}
	return &policy, nil
}

// SSHKeys returns the deploy keys the agent should install.
func (c *Client) SSHKeys(ctx context.Context) ([]AgentSSHKey, error) {
	var keys []AgentSSHKey
	if err := c.do(ctx, http.MethodGet, "/api/agent/ssh-key", nil, &keys, true); err != nil {
		return nil, err
	}
	return keys, nil
}

// persist stores a token response as the credential for this server.
func (c *Client) persist(token TokenResponse) (*Credential, error) {
	credential := &Credential{
		Server:       c.server,
		UserID:       token.UserID,
		Username:     token.Username,
		Token:        token.Token,
		RefreshToken: token.RefreshToken,
		ExpiresAt:    token.ExpiresAt,
	}

	if c.store == nil {
		store, err := LoadStore()
		if err != nil {
			return nil, err
		}
		c.store = store
	}

	c.store.Set(credential)
	if err := c.store.Save(); err != nil {
		return nil, err
	}
	c.credential = credential

	return credential, nil
}

// APIError is a non-2xx response from the server.
type APIError struct {
	StatusCode int
	Message    string
}

// Error implements error.
func (e *APIError) Error() string {
	if e.Message == "" {
		return fmt.Sprintf("server returned %d", e.StatusCode)
	}
	return e.Message
}

// authorization returns the Authorization header for an authenticated
// call, refreshing the access token first when it is close to expiring.
//
// Every authenticated request goes through here, including the ones
// that don't carry JSON, so a long-running agent uploading artefacts
// renews on exactly the same terms as one appending log lines.
func (c *Client) authorization(ctx context.Context) (string, error) {
	if c.credential == nil {
		return "", ErrNotLoggedIn
	}
	if c.credential.Expired() {
		if err := c.Refresh(ctx); err != nil {
			return "", fmt.Errorf("refresh token: %w", err)
		}
	}
	return "Bearer " + c.credential.Token, nil
}

// do performs a JSON request, discarding the response status.
func (c *Client) do(ctx context.Context, method, path string, payload, target any, authenticated bool) error {
	_, err := c.doStatus(ctx, method, path, payload, target, authenticated)
	return err
}

// doStatus performs a JSON request and reports the response status.
// When authenticated is set, the access token is attached and refreshed
// first if it is close to expiring.
func (c *Client) doStatus(ctx context.Context, method, path string, payload, target any, authenticated bool) (int, error) {
	if c.server == "" {
		return 0, errors.New("no server configured")
	}

	var authorization string
	if authenticated {
		header, err := c.authorization(ctx)
		if err != nil {
			return 0, err
		}
		authorization = header
	}

	var body io.Reader
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return 0, err
		}
		body = bytes.NewReader(encoded)
	}

	request, err := http.NewRequestWithContext(ctx, method, c.server+path, body)
	if err != nil {
		return 0, err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")
	request.Header.Set("User-Agent", UserAgent)
	if authorization != "" {
		request.Header.Set("Authorization", authorization)
	}

	response, err := c.http.Do(request)
	if err != nil {
		return 0, err
	}
	defer response.Body.Close()

	if response.StatusCode >= http.StatusBadRequest {
		return response.StatusCode, decodeAPIError(response)
	}

	if target == nil || response.StatusCode == http.StatusNoContent {
		_, _ = io.Copy(io.Discard, response.Body)
		return response.StatusCode, nil
	}

	if err := json.NewDecoder(response.Body).Decode(target); err != nil {
		return response.StatusCode, fmt.Errorf("decode response: %w", err)
	}

	return response.StatusCode, nil
}

// decodeAPIError turns an error response into an *APIError, falling back
// to the raw body when it isn't the platform error envelope.
func decodeAPIError(response *http.Response) error {
	contents, _ := io.ReadAll(io.LimitReader(response.Body, 4096))

	var envelope errorResponse
	if err := json.Unmarshal(contents, &envelope); err == nil && envelope.Error.Message != "" {
		return &APIError{StatusCode: response.StatusCode, Message: envelope.Error.Message}
	}

	message := strings.TrimSpace(string(contents))
	if len(message) > 200 {
		message = message[:200]
	}

	return &APIError{StatusCode: response.StatusCode, Message: message}
}

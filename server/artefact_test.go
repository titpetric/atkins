package server_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/titpetric/atkins/server"
	"github.com/titpetric/atkins/server/api"
	"github.com/titpetric/atkins/server/model"
)

// artefactView mirrors api.ArtefactView.
type artefactView struct {
	ID          string `json:"id"`
	JobID       string `json:"job_id"`
	Path        string `json:"path"`
	Size        int64  `json:"size"`
	ContentType string `json:"content_type"`
	Checksum    string `json:"checksum"`
	URL         string `json:"url"`
}

// artefactUpload is one file being pushed to a job.
type artefactUpload struct {
	Path        string
	ContentType string
	Checksum    string
	Content     []byte
}

// uploadArtefact posts a file to a job, the way the agent does: the
// body is the file, and everything else is a query parameter or a
// header.
func uploadArtefact(t *testing.T, server, token, jobID string, upload artefactUpload, target any) int {
	t.Helper()

	endpoint := server + "/api/job/" + jobID + "/artefact?path=" + url.QueryEscape(upload.Path)

	request, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(upload.Content))
	require.NoError(t, err)

	if upload.ContentType != "" {
		request.Header.Set("Content-Type", upload.ContentType)
	}
	if upload.Checksum != "" {
		request.Header.Set(api.HeaderArtefactChecksum, upload.Checksum)
	}
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}

	response, err := http.DefaultClient.Do(request)
	require.NoError(t, err)
	defer response.Body.Close()

	if target != nil && response.StatusCode < http.StatusBadRequest {
		require.NoError(t, json.NewDecoder(response.Body).Decode(target))
	} else {
		_, _ = io.Copy(io.Discard, response.Body)
	}

	return response.StatusCode
}

// fetch performs a GET and returns the status, the body and the
// response headers.
func fetch(t *testing.T, target, token string) (int, []byte, http.Header) {
	t.Helper()

	request, err := http.NewRequest(http.MethodGet, target, nil)
	require.NoError(t, err)
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}

	response, err := http.DefaultClient.Do(request)
	require.NoError(t, err)
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	require.NoError(t, err)

	return response.StatusCode, body, response.Header
}

// checksum is the SHA256 an agent sends with an upload.
func checksum(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}

// artefacts lists what a job produced.
func artefacts(t *testing.T, server, token, jobID string) []artefactView {
	t.Helper()

	var listed []artefactView
	status := call(t, http.MethodGet, server+"/api/job/"+jobID+"/artefact", token, nil, &listed)
	require.Equal(t, http.StatusOK, status)

	return listed
}

func TestArtefactRoundTrip(t *testing.T) {
	url := testServer(t)
	admin := register(t, url)
	agent := enrol(t, url, "agent-1")

	job := dispatch(t, url, admin.Token, "atkins scan", "")
	content := []byte(`{"findings": 3}`)

	var stored artefactView
	status := uploadArtefact(t, url, agent.Token, job.JobID, artefactUpload{
		Path:        "reports/scan.json",
		ContentType: "application/json",
		Checksum:    checksum(content),
		Content:     content,
	}, &stored)
	require.Equal(t, http.StatusCreated, status)

	assert.Equal(t, "reports/scan.json", stored.Path)
	assert.Equal(t, int64(len(content)), stored.Size)
	assert.Equal(t, "application/json", stored.ContentType)
	// The server hashes what it received rather than trusting the
	// header, so this is evidence the bytes arrived intact.
	assert.Equal(t, checksum(content), stored.Checksum)

	listed := artefacts(t, url, admin.Token, job.JobID)
	require.Len(t, listed, 1)
	assert.Equal(t, stored.ID, listed[0].ID)
	// The location of the bytes on the server is not part of the view.
	assert.NotContains(t, listed[0].URL, "storage_key")

	status, body, header := fetch(t, url+listed[0].URL, admin.Token)
	require.Equal(t, http.StatusOK, status)
	assert.Equal(t, content, body)
	assert.Equal(t, "application/json", header.Get("Content-Type"))
	// A job's own output must never render as a page in the server's
	// origin.
	assert.Contains(t, header.Get("Content-Disposition"), "attachment")
	assert.Equal(t, "nosniff", header.Get("X-Content-Type-Options"))
}

func TestArtefactAppearsOnTheJobPage(t *testing.T) {
	url := testServer(t)
	admin := register(t, url)
	agent := enrol(t, url, "agent-1")

	job := dispatch(t, url, admin.Token, "atkins scan", "")
	content := []byte("coverage: 81%\n")

	var stored artefactView
	status := uploadArtefact(t, url, agent.Token, job.JobID, artefactUpload{
		Path:    "coverage.txt",
		Content: content,
	}, &stored)
	require.Equal(t, http.StatusCreated, status)

	status, page, _ := fetch(t, url+"/job/"+job.JobID+"?t="+job.ViewToken, "")
	require.Equal(t, http.StatusOK, status)
	assert.Contains(t, string(page), "coverage.txt")

	// The link on the page has to work in a browser, which carries no
	// bearer token: it is governed by the same view token the page
	// itself is, so the link the page renders carries one.
	download := "/job/" + job.JobID + "/artefact/" + stored.ID
	assert.Contains(t, string(page), download)

	status, body, _ := fetch(t, url+download+"?t="+job.ViewToken, "")
	require.Equal(t, http.StatusOK, status)
	assert.Equal(t, content, body)

	// And no wider than the page: without the token the bytes are not
	// served, or a private job's artefacts would be public.
	status, _, _ = fetch(t, url+download, "")
	assert.Equal(t, http.StatusForbidden, status)
}

func TestArtefactUploadIsAgentOnly(t *testing.T) {
	url := testServer(t)
	admin := register(t, url)
	job := dispatch(t, url, admin.Token, "atkins scan", "")

	// An ordinary human, not an agent and not an admin.
	setSetting(t, url, admin.Token, model.SettingRegistrationOpen, "true")

	var human tokenResponse
	status := call(t, http.MethodPost, url+"/api/user/register", "", map[string]string{
		"email":    "dev@example.com",
		"username": "dev",
		"password": "correct-horse",
	}, &human)
	require.Equal(t, http.StatusCreated, status)

	status = uploadArtefact(t, url, human.Token, job.JobID, artefactUpload{
		Path:    "planted.json",
		Content: []byte("{}"),
	}, nil)
	assert.Equal(t, http.StatusForbidden, status)

	status = uploadArtefact(t, url, "", job.JobID, artefactUpload{
		Path:    "planted.json",
		Content: []byte("{}"),
	}, nil)
	assert.Equal(t, http.StatusUnauthorized, status)

	// Neither attempt left anything behind.
	assert.Empty(t, artefacts(t, url, admin.Token, job.JobID))
}

func TestArtefactUploadForAnUnknownJob(t *testing.T) {
	url := testServer(t)
	register(t, url)
	agent := enrol(t, url, "agent-1")

	status := uploadArtefact(t, url, agent.Token, "01ARZ3NDEKTSV4RRFFQ69G5FAV", artefactUpload{
		Path:    "scan.json",
		Content: []byte("{}"),
	}, nil)
	assert.Equal(t, http.StatusNotFound, status)
}

func TestArtefactRejectsPathsOutsideTheJob(t *testing.T) {
	url := testServer(t)
	admin := register(t, url)
	agent := enrol(t, url, "agent-1")

	job := dispatch(t, url, admin.Token, "atkins scan", "")

	// A path is a name within the job, not a location on the server.
	for _, path := range []string{"../../etc/passwd", "/etc/passwd", "reports/../../escape", "", "..", `..\..\windows`} {
		status := uploadArtefact(t, url, agent.Token, job.JobID, artefactUpload{
			Path:    path,
			Content: []byte("planted"),
		}, nil)
		assert.Equal(t, http.StatusBadRequest, status, "path %q", path)
	}

	assert.Empty(t, artefacts(t, url, admin.Token, job.JobID))
}

func TestArtefactRejectsOversize(t *testing.T) {
	url := testServer(t)
	admin := register(t, url)
	agent := enrol(t, url, "agent-1")

	setSetting(t, url, admin.Token, model.SettingArtefactMaxSize, "64B")

	job := dispatch(t, url, admin.Token, "atkins scan", "")

	status := uploadArtefact(t, url, agent.Token, job.JobID, artefactUpload{
		Path:    "core.dump",
		Content: bytes.Repeat([]byte("x"), 65),
	}, nil)
	assert.Equal(t, http.StatusRequestEntityTooLarge, status)

	// Refused, and not half stored: the row only exists once the bytes
	// are complete.
	assert.Empty(t, artefacts(t, url, admin.Token, job.JobID))

	// Exactly at the limit is still allowed.
	status = uploadArtefact(t, url, agent.Token, job.JobID, artefactUpload{
		Path:    "small.txt",
		Content: bytes.Repeat([]byte("x"), 64),
	}, nil)
	assert.Equal(t, http.StatusCreated, status)
}

func TestArtefactRejectsTooMany(t *testing.T) {
	url := testServer(t)
	admin := register(t, url)
	agent := enrol(t, url, "agent-1")

	setSetting(t, url, admin.Token, model.SettingArtefactMaxCount, "2")

	job := dispatch(t, url, admin.Token, "atkins scan", "")

	for _, path := range []string{"one.json", "two.json"} {
		status := uploadArtefact(t, url, agent.Token, job.JobID, artefactUpload{
			Path:    path,
			Content: []byte("{}"),
		}, nil)
		require.Equal(t, http.StatusCreated, status, path)
	}

	status := uploadArtefact(t, url, agent.Token, job.JobID, artefactUpload{
		Path:    "three.json",
		Content: []byte("{}"),
	}, nil)
	assert.Equal(t, http.StatusUnprocessableEntity, status)

	// Replacing one of the two it already has is not "one more".
	status = uploadArtefact(t, url, agent.Token, job.JobID, artefactUpload{
		Path:    "two.json",
		Content: []byte(`{"replaced": true}`),
	}, nil)
	assert.Equal(t, http.StatusCreated, status)

	assert.Len(t, artefacts(t, url, admin.Token, job.JobID), 2)
}

func TestArtefactRejectsAChecksumMismatch(t *testing.T) {
	url := testServer(t)
	admin := register(t, url)
	agent := enrol(t, url, "agent-1")

	job := dispatch(t, url, admin.Token, "atkins scan", "")

	// What the agent hashed is not what the server received: the upload
	// was truncated or altered in flight.
	status := uploadArtefact(t, url, agent.Token, job.JobID, artefactUpload{
		Path:     "scan.json",
		Checksum: checksum([]byte("the whole file")),
		Content:  []byte("half the f"),
	}, nil)
	assert.Equal(t, http.StatusBadRequest, status)

	assert.Empty(t, artefacts(t, url, admin.Token, job.JobID))
}

func TestArtefactReplacesTheSamePath(t *testing.T) {
	url := testServer(t)
	admin := register(t, url)
	agent := enrol(t, url, "agent-1")

	job := dispatch(t, url, admin.Token, "atkins scan", "")

	var first artefactView
	status := uploadArtefact(t, url, agent.Token, job.JobID, artefactUpload{
		Path:    "scan.json",
		Content: []byte(`{"run": 1}`),
	}, &first)
	require.Equal(t, http.StatusCreated, status)

	var second artefactView
	status = uploadArtefact(t, url, agent.Token, job.JobID, artefactUpload{
		Path:    "scan.json",
		Content: []byte(`{"run": 2}`),
	}, &second)
	require.Equal(t, http.StatusCreated, status)

	// One scan.json, holding the newer bytes.
	listed := artefacts(t, url, admin.Token, job.JobID)
	require.Len(t, listed, 1)
	assert.Equal(t, second.ID, listed[0].ID)

	status, body, _ := fetch(t, url+second.URL, admin.Token)
	require.Equal(t, http.StatusOK, status)
	assert.JSONEq(t, `{"run": 2}`, string(body))

	// The superseded artefact is gone rather than merely hidden.
	status, _, _ = fetch(t, url+first.URL, admin.Token)
	assert.Equal(t, http.StatusNotFound, status)
}

func TestArtefactDownloadNeedsAuthentication(t *testing.T) {
	url := testServer(t)
	admin := register(t, url)
	agent := enrol(t, url, "agent-1")

	job := dispatch(t, url, admin.Token, "atkins scan", "")

	var stored artefactView
	status := uploadArtefact(t, url, agent.Token, job.JobID, artefactUpload{
		Path:    "scan.json",
		Content: []byte("{}"),
	}, &stored)
	require.Equal(t, http.StatusCreated, status)

	status, _, _ = fetch(t, url+stored.URL, "")
	assert.Equal(t, http.StatusUnauthorized, status)

	status, _, _ = fetch(t, url+"/api/job/"+job.JobID+"/artefact", "")
	assert.Equal(t, http.StatusUnauthorized, status)
}

func TestArtefactsAreScopedLikeTheJob(t *testing.T) {
	url := testServer(t)
	admin := register(t, url)
	agent := enrol(t, url, "agent-1")
	dev := registerUser(t, url, admin.Token, "dev@example.com", "dev")

	job := dispatch(t, url, admin.Token, "atkins scan", "")

	var stored artefactView
	status := uploadArtefact(t, url, agent.Token, job.JobID, artefactUpload{
		Path:    "scan.json",
		Content: []byte(`{"findings": 3}`),
	}, &stored)
	require.Equal(t, http.StatusCreated, status)

	// An account that has never seen the job is told the same thing the
	// job endpoint tells it: nothing here. Authentication alone is not
	// the bar, or the artefact is the way around a scoped ledger.
	status, _, _ = fetch(t, url+"/api/job/"+job.JobID+"/artefact", dev.Token)
	assert.Equal(t, http.StatusNotFound, status)

	status, body, _ := fetch(t, url+stored.URL, dev.Token)
	assert.Equal(t, http.StatusNotFound, status)
	assert.NotContains(t, string(body), "findings")

	// The person who dispatched it reads it as before.
	status, body, _ = fetch(t, url+stored.URL, admin.Token)
	require.Equal(t, http.StatusOK, status)
	assert.JSONEq(t, `{"findings": 3}`, string(body))

	// And a public instance is shared again, artefacts included.
	setSetting(t, url, admin.Token, model.SettingJobVisibility, model.VisibilityPublic)

	require.Len(t, artefacts(t, url, dev.Token, job.JobID), 1)

	status, _, _ = fetch(t, url+stored.URL, dev.Token)
	assert.Equal(t, http.StatusOK, status)
}

func TestArtefactBelongsToOneJob(t *testing.T) {
	url := testServer(t)
	admin := register(t, url)
	agent := enrol(t, url, "agent-1")

	first := dispatch(t, url, admin.Token, "atkins scan", "")
	second := dispatch(t, url, admin.Token, "atkins scan", "")

	var stored artefactView
	status := uploadArtefact(t, url, agent.Token, first.JobID, artefactUpload{
		Path:    "scan.json",
		Content: []byte("{}"),
	}, &stored)
	require.Equal(t, http.StatusCreated, status)

	// An artefact ID from one job does not resolve under another.
	status, _, _ = fetch(t, url+"/api/job/"+second.JobID+"/artefact/"+stored.ID, admin.Token)
	assert.Equal(t, http.StatusNotFound, status)
}

func TestDispatchNormalizesArtefactPatterns(t *testing.T) {
	url := testServer(t)
	admin := register(t, url)

	var dispatched dispatchResponse
	status := call(t, http.MethodPost, url+"/api/dispatch", admin.Token, map[string]any{
		"repository": map[string]string{"remote_url": "git@github.com:titpetric/atkins.git"},
		"command":    "atkins scan",
		// The traversal is dropped rather than stored for an agent to
		// deal with later.
		"artefacts": []string{"reports/*.json", "../../etc/passwd", " ./coverage.out "},
	}, &dispatched)
	require.Equal(t, http.StatusCreated, status)

	var job map[string]any
	status = call(t, http.MethodGet, url+"/api/job/"+dispatched.JobID, admin.Token, nil, &job)
	require.Equal(t, http.StatusOK, status)
	assert.Equal(t, "reports/*.json,coverage.out", job["artefact_paths"])
}

func TestArtefactSizeSettingIsValidated(t *testing.T) {
	url := testServer(t)
	admin := register(t, url)

	status := call(t, http.MethodPost, url+"/api/admin/setting/"+model.SettingArtefactMaxSize, admin.Token,
		map[string]string{"value": "a handful"}, nil)
	assert.Equal(t, http.StatusBadRequest, status)

	setSetting(t, url, admin.Token, model.SettingArtefactMaxSize, "8MB")

	var settings []map[string]any
	status = call(t, http.MethodGet, url+"/api/admin/setting", admin.Token, nil, &settings)
	require.Equal(t, http.StatusOK, status)

	var found bool
	for _, setting := range settings {
		if setting["name"] == model.SettingArtefactMaxSize {
			found = true
			assert.Equal(t, "8MB", setting["value"])
			assert.Equal(t, string(model.KindBytes), setting["kind"])
		}
	}
	assert.True(t, found, "artefact.max_size is missing from the registry listing")
}

func TestArtefactRetentionSweepsTheBytes(t *testing.T) {
	var root string

	url := testServerWith(t, func(opts *server.Options) {
		root = filepath.Join(t.TempDir(), "artefacts")
		opts.ArtefactDir = root
		// The sweep shares the lease reclaim ticker.
		opts.ReclaimInterval = 50 * time.Millisecond
	})

	admin := register(t, url)
	agent := enrol(t, url, "agent-1")

	setSetting(t, url, admin.Token, model.SettingArtefactRetention, "1ms")

	job := dispatch(t, url, admin.Token, "atkins scan", "")

	var stored artefactView
	status := uploadArtefact(t, url, agent.Token, job.JobID, artefactUpload{
		Path:    "scan.json",
		Content: []byte(`{"findings": 3}`),
	}, &stored)
	require.Equal(t, http.StatusCreated, status)

	require.FileExists(t, filepath.Join(root, job.JobID, stored.ID))

	// The sweep is on a ticker, so wait for it rather than for a
	// fixed sleep to be long enough.
	require.Eventually(t, func() bool {
		return len(artefacts(t, url, admin.Token, job.JobID)) == 0
	}, 5*time.Second, 25*time.Millisecond)

	// Retention is about the disk: the bytes are what has to go.
	assert.NoFileExists(t, filepath.Join(root, job.JobID, stored.ID))

	status, _, _ = fetch(t, url+stored.URL, admin.Token)
	assert.Equal(t, http.StatusNotFound, status)
}

func TestArtefactEmptyListForAJobWithNone(t *testing.T) {
	url := testServer(t)
	admin := register(t, url)

	job := dispatch(t, url, admin.Token, "atkins scan", "")

	// An empty list rather than null: a caller should be able to range
	// over it without a nil check.
	status, body, _ := fetch(t, url+"/api/job/"+job.JobID+"/artefact", admin.Token)
	require.Equal(t, http.StatusOK, status)
	assert.Equal(t, "[]", strings.TrimSpace(string(body)))
}

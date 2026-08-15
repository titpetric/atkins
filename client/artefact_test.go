package client_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/titpetric/atkins/client"
)

// artefactServer captures one artefact upload, body and all.
//
// It is separate from fakeServer because that one decodes every request
// as JSON, and an artefact request body is a file.
type artefactServer struct {
	*httptest.Server

	Path        string
	Query       string
	ContentType string
	Checksum    string
	Auth        string
	Content     []byte

	// status is what the upload endpoint answers with.
	status int
}

func newArtefactServer(t *testing.T) *artefactServer {
	t.Helper()

	fake := &artefactServer{status: http.StatusCreated}

	fake.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/user/login" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(tokens("access"))
			return
		}

		if r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode([]client.Artefact{{ID: "artefact-1", Path: "scan.json", Size: 2}})
			return
		}

		body, _ := io.ReadAll(r.Body)

		fake.Path = r.URL.Path
		fake.Query = r.URL.RawQuery
		fake.ContentType = r.Header.Get("Content-Type")
		fake.Checksum = r.Header.Get(client.HeaderArtefactChecksum)
		fake.Auth = r.Header.Get("Authorization")
		fake.Content = body

		if fake.status >= http.StatusBadRequest {
			http.Error(w, `{"error":{"code":413,"message":"artefact is larger than artefact.max_size"}}`, fake.status)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(client.Artefact{
			ID:   "artefact-1",
			Path: r.URL.Query().Get("path"),
			Size: int64(len(body)),
		})
	}))
	t.Cleanup(fake.Close)

	t.Setenv("ATKINS_CREDENTIALS", filepath.Join(t.TempDir(), "credentials.json"))

	return fake
}

// loggedIn returns a client with a stored credential.
func loggedIn(t *testing.T, server string) *client.Client {
	t.Helper()

	c := client.New(server)
	_, err := c.Login(t.Context(), client.LoginRequest{Email: "ci@example.com"})
	require.NoError(t, err)

	return c
}

func TestUploadArtefactSendsTheFile(t *testing.T) {
	fake := newArtefactServer(t)
	c := loggedIn(t, fake.URL)

	artefact, err := c.UploadArtefact(t.Context(), "job-1", client.ArtefactUpload{
		Path:        "reports/scan.json",
		ContentType: "application/json",
		Checksum:    "abc123",
		Content:     strings.NewReader(`{"ok":true}`),
	})
	require.NoError(t, err)
	assert.Equal(t, "artefact-1", artefact.ID)

	assert.Equal(t, "/api/job/job-1/artefact", fake.Path)
	// The name is a query parameter, and it is escaped: a path with a
	// space or a plus in it has to survive the trip.
	assert.Equal(t, "path=reports%2Fscan.json", fake.Query)
	assert.Equal(t, "application/json", fake.ContentType)
	assert.Equal(t, "abc123", fake.Checksum)
	assert.Equal(t, "Bearer access", fake.Auth)
	assert.JSONEq(t, `{"ok":true}`, string(fake.Content))
}

func TestUploadArtefactDefaultsTheContentType(t *testing.T) {
	fake := newArtefactServer(t)
	c := loggedIn(t, fake.URL)

	_, err := c.UploadArtefact(t.Context(), "job-1", client.ArtefactUpload{
		Path:    "core.dump",
		Content: strings.NewReader("binary"),
	})
	require.NoError(t, err)

	assert.Equal(t, "application/octet-stream", fake.ContentType)
}

func TestUploadArtefactReportsARefusal(t *testing.T) {
	fake := newArtefactServer(t)
	fake.status = http.StatusRequestEntityTooLarge

	c := loggedIn(t, fake.URL)

	_, err := c.UploadArtefact(t.Context(), "job-1", client.ArtefactUpload{
		Path:    "core.dump",
		Content: strings.NewReader("too much"),
	})
	require.Error(t, err)

	var apiErr *client.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, http.StatusRequestEntityTooLarge, apiErr.StatusCode)
	assert.Contains(t, apiErr.Error(), "artefact.max_size")
}

func TestUploadArtefactWithoutCredentials(t *testing.T) {
	fake := newArtefactServer(t)

	_, err := client.New(fake.URL).UploadArtefact(t.Context(), "job-1", client.ArtefactUpload{
		Path:    "scan.json",
		Content: strings.NewReader("{}"),
	})
	assert.ErrorIs(t, err, client.ErrNotLoggedIn)
}

func TestArtefactsLists(t *testing.T) {
	fake := newArtefactServer(t)
	c := loggedIn(t, fake.URL)

	listed, err := c.Artefacts(t.Context(), "job-1")
	require.NoError(t, err)
	require.Len(t, listed, 1)
	assert.Equal(t, "scan.json", listed[0].Path)
}

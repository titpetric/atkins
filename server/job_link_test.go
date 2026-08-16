package server_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/titpetric/atkins/server/model"
)

// jobView is the part of the API's job payload this file is about.
type jobLinkView struct {
	ID        string `json:"id"`
	ViewToken string `json:"view_token"`
	URL       string `json:"url"`
}

// readJob reads one job as a caller who may see it.
func readJob(t *testing.T, url, token, jobID string) jobLinkView {
	t.Helper()

	var job jobLinkView
	status := call(t, http.MethodGet, url+"/api/job/"+jobID, token, nil, &job)
	require.Equal(t, http.StatusOK, status)

	return job
}

func TestJobPayloadCarriesTheViewToken(t *testing.T) {
	url := testServer(t)
	admin := register(t, url)

	dispatched := dispatch(t, url, admin.Token, "atkins build", "")
	require.NotEmpty(t, dispatched.ViewToken)

	// The token the dispatch printed is rebuildable from the job itself,
	// which is the whole point: a lost terminal is not a lost job.
	job := readJob(t, url, admin.Token, dispatched.JobID)
	assert.Equal(t, dispatched.ViewToken, job.ViewToken)
	assert.Equal(t, "/job/"+dispatched.JobID+"?t="+dispatched.ViewToken, job.URL)

	// And the listing carries it too, so a caller that found a job here
	// is not left without a way to open it.
	var listed []jobLinkView
	status := call(t, http.MethodGet, url+"/api/job", admin.Token, nil, &listed)
	require.Equal(t, http.StatusOK, status)
	require.Len(t, listed, 1)
	assert.Equal(t, dispatched.ViewToken, listed[0].ViewToken)

	// The link works in a browser with no session, which is what the
	// token is for.
	status, _, _ = fetch(t, url+job.URL, "")
	assert.Equal(t, http.StatusOK, status)
}

func TestJobPayloadHasNoTokenOnAPublicInstance(t *testing.T) {
	url := testServer(t)
	admin := register(t, url)

	setSetting(t, url, admin.Token, model.SettingJobVisibility, model.VisibilityPublic)

	dispatched := dispatch(t, url, admin.Token, "atkins build", "")

	// A public instance guards nothing with the token, and a secret in a
	// URL that guards nothing is just a longer URL.
	job := readJob(t, url, admin.Token, dispatched.JobID)
	assert.Empty(t, job.ViewToken)
	assert.Equal(t, "/job/"+dispatched.JobID, job.URL)

	status, _, _ := fetch(t, url+job.URL, "")
	assert.Equal(t, http.StatusOK, status)
}

func TestJobPayloadIsStillScoped(t *testing.T) {
	url := testServer(t)
	admin := register(t, url)
	dev := registerUser(t, url, admin.Token, "dev@example.com", "dev")

	theirs := dispatch(t, url, admin.Token, "atkins build", "")

	// Handing out the token is safe because the caller had to pass the
	// job scope to get here. Somebody who may not read the job still
	// gets a 404 rather than a link to it.
	status := call(t, http.MethodGet, url+"/api/job/"+theirs.JobID, dev.Token, nil, nil)
	assert.Equal(t, http.StatusNotFound, status)
}

func TestIndexListsTheSignedInUsersJobs(t *testing.T) {
	url := testServer(t)
	admin := register(t, url)
	openRegistration(t, url, admin.Token)

	dev := registerUser(t, url, admin.Token, "dev@example.com", "dev")

	mine := dispatch(t, url, dev.Token, "atkins mine", "")
	theirs := dispatch(t, url, admin.Token, "atkins theirs", "")

	// With no session there is nothing to scope a listing by, so the
	// page says where to sign in rather than listing the instance.
	anonymous := newBrowser(t, url)
	status, body := anonymous.get("/")
	require.Equal(t, http.StatusOK, status)
	assert.Contains(t, body, "private on this server")
	assert.NotContains(t, body, "atkins mine")

	signed := newBrowser(t, url)
	signed.login("dev@example.com", "correct-horse")

	status, body = signed.get("/")
	require.Equal(t, http.StatusOK, status)
	assert.Contains(t, body, "atkins mine")
	assert.NotContains(t, body, "atkins theirs")

	// The link on the row carries the job's own view token, so the page
	// it points at opens.
	assert.Contains(t, body, "/job/"+mine.JobID+"?t="+mine.ViewToken)

	status, _, _ = fetch(t, url+"/job/"+mine.JobID+"?t="+mine.ViewToken, "")
	assert.Equal(t, http.StatusOK, status)

	// An admin sees the instance, which is the rule the API already
	// applies to a listing.
	operator := newBrowser(t, url)
	operator.login("ci@example.com", "correct-horse")

	status, body = operator.get("/")
	require.Equal(t, http.StatusOK, status)
	assert.Contains(t, body, "atkins mine")
	assert.Contains(t, body, "atkins theirs")
	assert.Equal(t, 1, strings.Count(body, "/job/"+theirs.JobID+"?t="))
}

package server_test

import (
	"net/http"
	urlpkg "net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// jobView mirrors the fields of api.JobView these tests read.
type jobView struct {
	ID          string `json:"id"`
	Command     string `json:"command"`
	Status      string `json:"status"`
	Interactive bool   `json:"interactive"`
	ViewToken   string `json:"view_token"`
}

// enrolAgent trades the shared token for an agent's access token.
func enrolAgent(t *testing.T, url, agentID string) string {
	t.Helper()

	var enrolled tokenResponse
	status := call(t, http.MethodPost, url+"/api/agent/enrol", "", map[string]any{
		"token":    testAgentToken,
		"agent_id": agentID,
	}, &enrolled)
	require.Equal(t, http.StatusOK, status)

	return enrolled.Token
}

// claimJob leases the oldest pending job for an agent, so a test can
// exercise the endpoints that need a job to be running.
func claimJob(t *testing.T, url, token, agentID string) *claimResponse {
	t.Helper()

	var claimed claimResponse
	status := call(t, http.MethodPost, url+"/api/job/claim", token, map[string]any{
		"agent_id": agentID,
	}, &claimed)
	require.Equal(t, http.StatusOK, status)
	require.NotNil(t, claimed.Job)

	return &claimed
}

// operatorOn signs a browser in as the instance's admin.
func operatorOn(t *testing.T, url string) *browser {
	t.Helper()

	operator := newBrowser(t, url)
	operator.login("ci@example.com", "correct-horse")

	return operator
}

// addProject fills in the add form and returns the project's id, read
// out of the redirect it lands on.
func addProject(t *testing.T, operator *browser, form urlpkg.Values) string {
	t.Helper()

	form.Set("csrf_token", operator.csrf("/admin/project"))

	response := operator.hop(http.MethodPost, "/admin/project", form)
	require.Equal(t, http.StatusSeeOther, response.StatusCode)

	location := response.Header.Get("Location")
	require.Contains(t, location, "/admin/project/")

	id, _, _ := strings.Cut(strings.TrimPrefix(location, "/admin/project/"), "?")
	require.NotEmpty(t, id)

	return id
}

// A project is a repository somebody named, so adding one is the same
// row a dispatch would have created — with the name and the defaults it
// could not have known.
func TestAddingAProjectRecordsItAndQueuesTheListing(t *testing.T) {
	url := testServer(t)
	register(t, url)
	operator := operatorOn(t, url)

	id := addProject(t, operator, urlpkg.Values{
		"remote_url": {"https://github.com/titpetric/atkins.git"},
		"name":       {"Atkins"},
		"command":    {"atkins test:simple"},
		"ref":        {"main"},
	})

	status, body := operator.get("/admin/project/" + id)
	require.Equal(t, http.StatusOK, status)
	assert.Contains(t, body, "Atkins")
	assert.Contains(t, body, "github.com/titpetric/atkins")

	// Nothing has run it yet, so the page says what it is waiting for
	// rather than showing an empty menu.
	assert.Contains(t, body, "Reading the pipeline")

	// The listing job is queued and claimable, which is what makes it
	// the agent's checkout the tree is read from rather than the
	// server's guess at one.
	admin := loginToken(t, url)

	var jobs []jobView
	code := call(t, http.MethodGet, url+"/api/job", admin, nil, &jobs)
	require.Equal(t, http.StatusOK, code)
	require.Len(t, jobs, 1)
	assert.Contains(t, jobs[0].Command, "atkins --list --json")
	assert.Equal(t, "pending", jobs[0].Status)
}

// The slug is the identity. Adding a remote the server has already seen
// names the row that exists rather than making a second one.
func TestAddingAProjectTwiceNamesTheSameRepository(t *testing.T) {
	url := testServer(t)
	admin := register(t, url)
	operator := operatorOn(t, url)

	// A dispatch got there first, under the ssh spelling of the remote.
	dispatch(t, url, admin.Token, "atkins test:build", "")

	id := addProject(t, operator, urlpkg.Values{
		"remote_url": {"https://github.com/titpetric/atkins"},
		"name":       {"Atkins"},
	})

	var repositories []map[string]any
	code := call(t, http.MethodGet, url+"/api/repository", admin.Token, nil, &repositories)
	require.Equal(t, http.StatusOK, code)
	require.Len(t, repositories, 1)
	assert.Equal(t, id, repositories[0]["id"])
	assert.Equal(t, "Atkins", repositories[0]["name"])
}

func TestAddingAProjectNeedsACloneAddress(t *testing.T) {
	url := testServer(t)
	register(t, url)
	operator := operatorOn(t, url)

	form := urlpkg.Values{"name": {"Nameless"}, "csrf_token": {operator.csrf("/admin/project")}}

	response := operator.hop(http.MethodPost, "/admin/project", form)
	require.Equal(t, http.StatusSeeOther, response.StatusCode)
	assert.Contains(t, response.Header.Get("Location"), "clone+address")
}

// Running is a menu of the project's own jobs. A name the listing does
// not carry is not a job, whatever the form says.
func TestRunningRefusesAJobThePipelineDoesNotDeclare(t *testing.T) {
	url := testServer(t)
	register(t, url)
	operator := operatorOn(t, url)

	id := addProject(t, operator, urlpkg.Values{
		"remote_url": {"https://github.com/titpetric/atkins.git"},
	})

	response := operator.hop(http.MethodPost, "/admin/project/"+id+"/run", urlpkg.Values{
		"job":        {"rm -rf /"},
		"csrf_token": {operator.csrf("/admin/project/" + id)},
	})
	require.Equal(t, http.StatusSeeOther, response.StatusCode)
	assert.Contains(t, response.Header.Get("Location"), "pipeline+has+not+been+read")
}

// projectListing is what an agent uploads for a project with one
// ordinary job, one nested one, and a shell.
const projectListing = `[{"desc":"Demo","cmds":[
	{"id":"default","desc":"Run everything","cmd":"atkins default"},
	{"id":"shell","desc":"A shell on the agent","cmd":"atkins shell","interactive":true},
	{"id":"test:simple","desc":"Tests","cmd":"atkins test:simple"}
]}]`

// The whole path: a project is added, an agent runs the listing job and
// uploads what it produced, and the page reads the tree back out of that
// artefact.
func TestProjectReadsItsTreeFromTheListingArtefact(t *testing.T) {
	url := testServer(t)
	register(t, url)
	operator := operatorOn(t, url)

	id := addProject(t, operator, urlpkg.Values{
		"remote_url": {"https://github.com/titpetric/atkins.git"},
		"name":       {"Demo"},
	})

	agent := enrolAgent(t, url, "agent-1")
	claimed := claimJob(t, url, agent, "agent-1")
	require.Contains(t, claimed.Job.Command, "atkins --list --json")

	require.Equal(t, http.StatusCreated, uploadArtefact(t, url, agent, claimed.Job.ID, artefactUpload{
		Path:    "pipeline.json",
		Content: []byte(projectListing),
	}, nil))

	status := call(t, http.MethodPost, url+"/api/job/"+claimed.Job.ID+"/status", agent, map[string]any{
		"status":    "passed",
		"exit_code": 0,
	}, nil)
	require.Equal(t, http.StatusOK, status)

	code, body := operator.get("/admin/project/" + id)
	require.Equal(t, http.StatusOK, code)

	// The tree, nested the way the names are: `simple` under `test`.
	assert.Contains(t, body, `value="default"`)
	assert.Contains(t, body, `value="test:simple"`)
	assert.Contains(t, body, "A shell on the agent")

	// Only the job that declared it is marked as taking a keyboard.
	assert.Contains(t, body, "interactive")

	// Reading it once caches it, so the page does not re-open the
	// artefact — and still works after retention has swept it.
	assert.Contains(t, body, "Read it again")
}

// Running dispatches the command the listing named, and an interactive
// job is dispatched interactive — the pipeline decides, not the form.
func TestRunningDispatchesWhatTheListingNamed(t *testing.T) {
	url := testServer(t)
	admin := register(t, url)
	operator := operatorOn(t, url)

	id := addProject(t, operator, urlpkg.Values{
		"remote_url": {"https://github.com/titpetric/atkins.git"},
	})

	agent := enrolAgent(t, url, "agent-1")
	claimed := claimJob(t, url, agent, "agent-1")
	require.Equal(t, http.StatusCreated, uploadArtefact(t, url, agent, claimed.Job.ID, artefactUpload{
		Path:    "pipeline.json",
		Content: []byte(projectListing),
	}, nil))
	require.Equal(t, http.StatusOK, call(t, http.MethodPost, url+"/api/job/"+claimed.Job.ID+"/status", agent,
		map[string]any{"status": "passed", "exit_code": 0}, nil))

	// Loading the page caches the tree, which is what the run reads.
	code, _ := operator.get("/admin/project/" + id)
	require.Equal(t, http.StatusOK, code)

	response := operator.hop(http.MethodPost, "/admin/project/"+id+"/run", urlpkg.Values{
		"job":        {"shell"},
		"csrf_token": {operator.csrf("/admin/project/" + id)},
	})
	require.Equal(t, http.StatusSeeOther, response.StatusCode)

	// Straight to the terminal: the next thing anybody wants after
	// pressing run is to watch it.
	assert.Contains(t, response.Header.Get("Location"), "/terminal")

	var jobs []jobView
	require.Equal(t, http.StatusOK, call(t, http.MethodGet, url+"/api/job?limit=1", admin.Token, nil, &jobs))
	require.Len(t, jobs, 1)
	assert.Equal(t, "atkins shell", jobs[0].Command)
	assert.True(t, jobs[0].Interactive, "a job the pipeline declared interactive was dispatched without a keyboard")

	// And a job that did not declare it does not get one.
	response = operator.hop(http.MethodPost, "/admin/project/"+id+"/run", urlpkg.Values{
		"job":        {"test:simple"},
		"csrf_token": {operator.csrf("/admin/project/" + id)},
	})
	require.Equal(t, http.StatusSeeOther, response.StatusCode)

	require.Equal(t, http.StatusOK, call(t, http.MethodGet, url+"/api/job?limit=1", admin.Token, nil, &jobs))
	require.Len(t, jobs, 1)
	assert.Equal(t, "atkins test:simple", jobs[0].Command)
	assert.False(t, jobs[0].Interactive)
}

func TestProjectPagesNeedAnAdminSession(t *testing.T) {
	url := testServer(t)
	register(t, url)

	anonymous := newBrowser(t, url)
	for _, path := range []string{"/admin/project", "/admin/project/whatever"} {
		response := anonymous.hop(http.MethodGet, path, nil)
		assert.Equal(t, http.StatusSeeOther, response.StatusCode, path)
		assert.Contains(t, response.Header.Get("Location"), "/login", path)
	}
}

// The terminal is a second window onto a job, so it is gated on the same
// view token the job page is — no wider and no narrower.
func TestTerminalIsGatedOnTheJobsViewToken(t *testing.T) {
	url := testServer(t)
	admin := register(t, url)
	job := dispatch(t, url, admin.Token, "atkins test:build", "")

	visitor := newBrowser(t, url)

	// The stream is not fetched with a token here: this job has not
	// settled, so the request would be held open for output — which is
	// the whole point of it, and not something a test should sit in.
	// TestStreamReplaysAndEndsForASettledJob covers the other side.
	for _, path := range []string{
		"/job/" + job.JobID + "/terminal",
		"/job/" + job.JobID + "/stream",
	} {
		status, body := visitor.get(path)
		assert.Equal(t, http.StatusForbidden, status, path)
		assert.Contains(t, body, "access token", path)
	}

	// Keystrokes go through the same gate, so a stranger holding a job
	// id cannot type at somebody's shell.
	typed, err := visitor.client.Post(
		url+"/job/"+job.JobID+"/input", "text/plain", strings.NewReader("ls\r"))
	require.NoError(t, err)
	defer typed.Body.Close()
	assert.Equal(t, http.StatusForbidden, typed.StatusCode)

	status, body := visitor.get("/job/" + job.JobID + "/terminal?t=" + job.ViewToken)
	assert.Equal(t, http.StatusOK, status)
	assert.Contains(t, body, "/assets/xterm.js")
	assert.Contains(t, body, "read only")
}

// A settled job's stream replays what was stored and then says it is
// over, rather than holding the request open for output that will never
// come.
func TestStreamReplaysAndEndsForASettledJob(t *testing.T) {
	url := testServer(t)
	admin := register(t, url)
	job := dispatch(t, url, admin.Token, "atkins test:build", "")

	agent := enrolAgent(t, url, "agent-1")
	claimJob(t, url, agent, "agent-1")

	status := call(t, http.MethodPost, url+"/api/job/"+job.JobID+"/log", agent, map[string]string{
		"stream":  "output",
		"content": "hello from the build\n",
	}, nil)
	require.Equal(t, http.StatusNoContent, status)

	status = call(t, http.MethodPost, url+"/api/job/"+job.JobID+"/status", agent, map[string]any{
		"status":    "passed",
		"exit_code": 0,
	}, nil)
	require.Equal(t, http.StatusOK, status)

	visitor := newBrowser(t, url)
	code, body := visitor.get("/job/" + job.JobID + "/stream?t=" + job.ViewToken)

	require.Equal(t, http.StatusOK, code)
	assert.Contains(t, body, "event: output")
	assert.Contains(t, body, "hello from the build")
	assert.Contains(t, body, "event: end")

	// The field names are the wire format the terminal page is written
	// against, so they are worth asserting rather than inheriting from
	// whatever the struct's fields happen to be called.
	assert.Contains(t, body, `"seq":`)
	assert.Contains(t, body, `"content":`)
}

// A job that never asked for stdin does not get a keyboard, whatever is
// posted at it.
func TestInputIsRefusedForANonInteractiveJob(t *testing.T) {
	url := testServer(t)
	admin := register(t, url)
	job := dispatch(t, url, admin.Token, "atkins test:build", "")

	agent := enrolAgent(t, url, "agent-1")
	claimJob(t, url, agent, "agent-1")

	visitor := newBrowser(t, url)

	response, err := visitor.client.Post(
		url+"/job/"+job.JobID+"/input?t="+job.ViewToken, "text/plain", strings.NewReader("ls\r"))
	require.NoError(t, err)
	defer response.Body.Close()
	assert.Equal(t, http.StatusNoContent, response.StatusCode)

	// Nothing was queued, so the agent's collection waits out its poll
	// and comes back empty.
	var collected map[string]string
	status := call(t, http.MethodGet,
		url+"/api/job/"+job.JobID+"/input?agent_id=agent-1", agent, nil, &collected)
	require.Equal(t, http.StatusOK, status)
	assert.Empty(t, collected["input"])
}

// Only the agent holding the lease collects, and only while the job is
// running: input is meant for a process, and the lease says which agent
// has one.
func TestInputCollectionNeedsTheLease(t *testing.T) {
	url := testServer(t)
	admin := register(t, url)
	job := dispatch(t, url, admin.Token, "atkins test:build", "")

	first := enrolAgent(t, url, "agent-1")
	claimJob(t, url, first, "agent-1")

	// Right agent, wrong name: the lease is recorded against the id the
	// job was claimed with.
	status := call(t, http.MethodGet,
		url+"/api/job/"+job.JobID+"/input?agent_id=agent-2", first, nil, nil)
	assert.Equal(t, http.StatusConflict, status)

	// A human, however administrative, is not an agent.
	status = call(t, http.MethodGet,
		url+"/api/job/"+job.JobID+"/input?agent_id=agent-1", admin.Token, nil, nil)
	assert.Equal(t, http.StatusForbidden, status)
}

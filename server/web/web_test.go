package web

import (
	"strings"
	"testing"
	"time"

	"github.com/a-h/templ"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/titpetric/atkins/server/model"
)

// render runs a component the way a handler does and returns the page.
//
// Components are plain functions, so a render test needs no Handlers
// and no storage — which is most of what the templ port bought.
func render(t *testing.T, page templ.Component) string {
	t.Helper()

	var out strings.Builder
	require.NoError(t, page.Render(t.Context(), &out))

	return out.String()
}

func TestPagesRender(t *testing.T) {
	// A page that cannot render is now a compile error rather than a
	// start-up one, so what is left to check is that each one produces
	// a document at all.
	pages := map[string]templ.Component{
		"index": indexView(IndexPage{}, Links{}),
		"job":   jobView(&JobPage{Job: &model.Job{ID: "job-1"}}, Links{}),
		"error": errorView(404, "no such job"),
		"login": loginView(&loginPage{}),
	}

	for name, page := range pages {
		t.Run(name, func(t *testing.T) {
			rendered := render(t, page)
			assert.Contains(t, rendered, "<!doctype html>")
			assert.Contains(t, rendered, "</html>")
		})
	}
}

func TestJobPageRendersEveryStatus(t *testing.T) {
	started := time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)
	finished := started.Add(90 * time.Second)

	for _, status := range []string{
		model.JobStatusPending,
		model.JobStatusRunning,
		model.JobStatusPassed,
		model.JobStatusFailed,
		model.JobStatusTimeout,
		model.JobStatusCancelled,
	} {
		t.Run(status, func(t *testing.T) {
			job := &model.Job{
				ID:      "01ARZ3NDEKTSV4RRFFQ69G5FAV",
				Command: "atkins test:build",
				Status:  status,
			}
			job.SetCreatedAt(started)
			job.SetStartedAt(started)
			job.SetFinishedAt(finished)

			rendered := render(t, jobView(&JobPage{
				Job:        job,
				Repository: &model.Repository{Slug: "github.com/titpetric/atkins"},
				Log:        []model.JobLog{{Content: "hello\n"}},
			}, Links{}))

			assert.Contains(t, rendered, "badge-"+status)
			assert.Contains(t, rendered, "atkins test:build")
			assert.Contains(t, rendered, "github.com/titpetric/atkins")
			assert.Contains(t, rendered, "hello")
		})
	}
}

func TestJobPageEscapesOutput(t *testing.T) {
	job := &model.Job{ID: "job-1", Status: model.JobStatusFailed, Command: "atkins"}

	rendered := render(t, jobView(&JobPage{
		Job: job,
		Log: []model.JobLog{{Content: "<script>alert(1)</script>"}},
	}, Links{}))

	// Job output is whatever a build printed, and it is rendered into a
	// page anyone with the URL can open.
	assert.NotContains(t, rendered, "<script>alert(1)</script>")
	assert.Contains(t, rendered, "&lt;script&gt;")
}

func TestJobPageListsArtefacts(t *testing.T) {
	job := &model.Job{ID: "01ARZ3NDEKTSV4RRFFQ69G5FAV", Status: model.JobStatusPassed, Command: "atkins scan"}

	rendered := render(t, jobView(&JobPage{
		Job: job,
		Artefacts: []model.JobArtefact{{
			ID:          "01J000000000000000000ART1",
			JobID:       job.ID,
			Path:        "reports/scan.json",
			Size:        2048,
			ContentType: "application/json",
		}},
	}, Links{}))

	assert.Contains(t, rendered, "reports/scan.json")
	assert.Contains(t, rendered, "2.0 KB")
	// The link a browser can follow: no bearer token to offer, so it
	// is the page's own route rather than the API's.
	assert.Contains(t, rendered, "/job/"+job.ID+"/artefact/01J000000000000000000ART1")
}

func TestFilesize(t *testing.T) {
	assert.Equal(t, "0 B", filesize(0))
	assert.Equal(t, "512 B", filesize(512))
	assert.Equal(t, "1.0 KB", filesize(1024))
	assert.Equal(t, "1.5 MB", filesize(1536*1024))
	assert.Equal(t, "2.0 GB", filesize(2*1024*1024*1024))
}

func TestStamp(t *testing.T) {
	assert.Equal(t, "—", stamp(nil))

	when := time.Date(2026, 8, 15, 12, 34, 56, 0, time.UTC)
	assert.Equal(t, "2026-08-15 12:34:56 UTC", stamp(&when))
}

func TestDuration(t *testing.T) {
	job := &model.Job{}
	assert.Equal(t, "—", duration(job))

	started := time.Now().Add(-90 * time.Second)
	job.SetStartedAt(started)
	job.SetFinishedAt(started.Add(90 * time.Second))
	assert.Equal(t, "1m30s", duration(job))

	// A running job is measured against now, so the page shows it
	// climbing rather than showing nothing.
	unfinished := &model.Job{}
	unfinished.SetStartedAt(time.Now().Add(-2 * time.Second))
	assert.NotEqual(t, "—", duration(unfinished))
}

func TestChildrenOf(t *testing.T) {
	jobs := []model.Job{
		{ID: "root"},
		{ID: "child-1", ParentID: "root"},
		{ID: "child-2", ParentID: "root"},
		{ID: "grandchild", ParentID: "child-1"},
	}

	children := childrenOf(jobs, "root")
	require.Len(t, children, 2)
	assert.Equal(t, "child-1", children[0].ID)
	assert.Equal(t, "child-2", children[1].ID)

	assert.Empty(t, childrenOf(jobs, "grandchild"))
}

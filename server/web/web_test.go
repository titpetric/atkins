package web

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/titpetric/atkins/server/model"
)

func TestTemplatesParse(t *testing.T) {
	// A template that doesn't parse would only surface when somebody
	// opened the URL atkins had already printed.
	handlers, err := NewHandlers(Options{})
	require.NoError(t, err)

	for _, name := range []string{"index.html", "job.html", "error.html"} {
		assert.NotNil(t, handlers.templates.Lookup(name), name)
	}
}

func TestJobPageRendersEveryStatus(t *testing.T) {
	handlers, err := NewHandlers(Options{})
	require.NoError(t, err)

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

			var out strings.Builder
			err := handlers.templates.ExecuteTemplate(&out, "job.html", &JobPage{
				Job:        job,
				Repository: &model.Repository{Slug: "github.com/titpetric/atkins"},
				Log:        []model.JobLog{{Content: "hello\n"}},
			})
			require.NoError(t, err)

			rendered := out.String()
			assert.Contains(t, rendered, "badge-"+status)
			assert.Contains(t, rendered, "atkins test:build")
			assert.Contains(t, rendered, "github.com/titpetric/atkins")
			assert.Contains(t, rendered, "hello")
		})
	}
}

func TestJobPageEscapesOutput(t *testing.T) {
	handlers, err := NewHandlers(Options{})
	require.NoError(t, err)

	job := &model.Job{ID: "job-1", Status: model.JobStatusFailed, Command: "atkins"}

	var out strings.Builder
	err = handlers.templates.ExecuteTemplate(&out, "job.html", &JobPage{
		Job: job,
		Log: []model.JobLog{{Content: "<script>alert(1)</script>"}},
	})
	require.NoError(t, err)

	// Job output is whatever a build printed, and it is rendered into a
	// page anyone with the URL can open.
	assert.NotContains(t, out.String(), "<script>alert(1)</script>")
	assert.Contains(t, out.String(), "&lt;script&gt;")
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

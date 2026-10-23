package runner_test

import (
	"testing"

	"github.com/titpetric/atkins/internal/assert"
	"github.com/titpetric/atkins/model"
	"github.com/titpetric/atkins/runner"
)

// TestRunPipeline_InactiveJob checks running a job whose when: block
// does not match fails, also through a task: reference.
func TestRunPipeline_InactiveJob(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	build := &model.Job{Name: "build", Inactive: true, Steps: []*model.Step{{Run: "true"}}}
	all := &model.Job{Name: "all", Steps: []*model.Step{{Task: "build"}}}
	pipeline := &model.Pipeline{
		Name: "gated",
		Dir:  dir,
		Jobs: map[string]*model.Job{"build": build, "all": all},
	}

	for _, job := range []string{"build", "all"} {
		err := runner.RunPipeline(t.Context(), pipeline, runner.PipelineOptions{
			Jobs:         []string{job},
			Silent:       true,
			AllPipelines: []*model.Pipeline{pipeline},
		})
		assert.ErrorContains(t, err, "does not apply here")
	}
}

package runner_test

import (
	"testing"
	"time"

	"github.com/titpetric/atkins/internal/assert"
	"github.com/titpetric/atkins/model"
	"github.com/titpetric/atkins/runner"
	"github.com/titpetric/atkins/treeview"
)

// jobContext returns the execution context a single job runs in.
func jobContext(job *model.Job) *runner.ExecutionContext {
	builder := treeview.NewBuilder("test")
	return &runner.ExecutionContext{
		Variables:  runner.NewContextVariables(nil),
		Env:        make(map[string]string),
		Job:        job,
		CurrentJob: builder.AddJob(job, nil, job.Name),
		Display:    treeview.NewSilentDisplay(),
		Builder:    builder,
	}
}

func TestNewExecutor(t *testing.T) {
	t.Parallel()

	job := &model.Job{Name: "pass", Steps: []*model.Step{{Run: "true"}}}
	assert.NoError(t, runner.NewExecutor().ExecuteJob(t.Context(), jobContext(job)))
}

func TestNewExecutorWithOptions(t *testing.T) {
	t.Parallel()

	t.Run("nil options run with the defaults", func(t *testing.T) {
		t.Parallel()
		job := &model.Job{Name: "pass", Steps: []*model.Step{{Run: "true"}}}
		assert.NoError(t, runner.NewExecutorWithOptions(nil).ExecuteJob(t.Context(), jobContext(job)))
	})

	t.Run("the default timeout ends a job without its own", func(t *testing.T) {
		t.Parallel()
		job := &model.Job{Name: "slow", Steps: []*model.Step{{Run: "sleep 5"}}}
		executor := runner.NewExecutorWithOptions(&runner.Options{DefaultTimeout: 50 * time.Millisecond})

		start := time.Now()
		assert.Error(t, executor.ExecuteJob(t.Context(), jobContext(job)))
		assert.True(t, time.Since(start) < 4*time.Second, "the job outlived its timeout")
	})
}

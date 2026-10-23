package runner_test

import (
	"sync"
	"testing"

	"github.com/titpetric/atkins/internal/assert"
	"github.com/titpetric/atkins/model"
	"github.com/titpetric/atkins/runner"
)

// progressRun runs a pipeline and returns the statuses its observer saw per job.
func progressRun(t *testing.T, jobs map[string]*model.Job, run string) (map[string][]runner.JobProgressStatus, error) {
	t.Helper()

	var (
		mu  sync.Mutex
		got = map[string][]runner.JobProgressStatus{}
	)
	var observer runner.ProgressObserver = runner.ProgressObserverFunc(func(ev runner.JobProgressEvent) {
		mu.Lock()
		defer mu.Unlock()
		got[ev.JobName] = append(got[ev.JobName], ev.Status)
		if ev.Status == runner.JobProgressFailed {
			assert.Error(t, ev.Err)
		}
	})

	pipeline := &model.Pipeline{Name: "progress", Dir: t.TempDir(), Jobs: jobs}
	err := runner.RunPipeline(t.Context(), pipeline, runner.PipelineOptions{
		Jobs:         []string{run},
		Silent:       true,
		AllPipelines: []*model.Pipeline{pipeline},
		Progress:     observer,
	})
	return got, err
}

func TestJobProgressStatus(t *testing.T) {
	t.Parallel()

	got, err := progressRun(t, map[string]*model.Job{
		"pass": {Name: "pass", Steps: []*model.Step{{Run: "true"}}},
	}, "pass")
	assert.NoError(t, err)
	assert.Equal(t, []runner.JobProgressStatus{runner.JobProgressRunning, runner.JobProgressPassed}, got["pass"])
}

func TestJobProgressEvent(t *testing.T) {
	t.Parallel()

	got, err := progressRun(t, map[string]*model.Job{
		"fail": {Name: "fail", Steps: []*model.Step{{Run: "false"}}},
		"skip": {Name: "skip", If: model.Conditionals{"false"}, Steps: []*model.Step{{Run: "true"}}},
		"all":  {Name: "all", DependsOn: []string{"skip", "fail"}, Steps: []*model.Step{{Run: "true"}}},
	}, "all")
	assert.Error(t, err)
	assert.Equal(t, []runner.JobProgressStatus{runner.JobProgressRunning, runner.JobProgressSkipped}, got["skip"])
	assert.Equal(t, []runner.JobProgressStatus{runner.JobProgressRunning, runner.JobProgressFailed}, got["fail"])
	assert.Equal(t, 0, len(got["all"]), "a job after a failed dependency never starts")
}

func TestProgressObserverFunc_OnJobProgress(t *testing.T) {
	t.Parallel()

	var got runner.JobProgressEvent
	runner.ProgressObserverFunc(func(ev runner.JobProgressEvent) { got = ev }).
		OnJobProgress(runner.JobProgressEvent{JobName: "build", Status: runner.JobProgressPassed})
	assert.Equal(t, "build", got.JobName)
	assert.Equal(t, runner.JobProgressPassed, got.Status)
}

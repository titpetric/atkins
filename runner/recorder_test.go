package runner_test

import (
	"context"
	"errors"
	"testing"

	"github.com/titpetric/atkins/eventlog"
	"github.com/titpetric/atkins/internal/assert"
	"github.com/titpetric/atkins/model"
	"github.com/titpetric/atkins/runner"
)

// fakeRecorder is a runner.Recorder that keeps the documents it receives,
// and optionally fails. A test can check both that the document arrives
// and that a failure to store it never reaches the caller.
type fakeRecorder struct {
	log  *eventlog.Log
	fail error
}

func (f *fakeRecorder) Record(_ context.Context, log *eventlog.Log) error {
	f.log = log
	return f.fail
}

// twoStepPipeline is a job whose two steps each run a command. The
// document has a tree to walk and events to count.
func twoStepPipeline() *model.Pipeline {
	return &model.Pipeline{
		Name: "recorded",
		Jobs: map[string]*model.Job{
			"work": {
				Name:     "work",
				Desc:     "Do two things",
				Passthru: true,
				Steps: []*model.Step{
					{Run: "echo first"},
					{Run: "echo second"},
				},
			},
		},
	}
}

// TestRecorder covers the hook the command line hands RunPipeline so
// every run leaves a row behind.
func TestRecorder(t *testing.T) {
	t.Run("receives the finished run", func(t *testing.T) {
		pipeline := twoStepPipeline()
		recorder := &fakeRecorder{}

		assert.NoError(t, runner.RunPipeline(t.Context(), pipeline, runner.PipelineOptions{
			Jobs:         []string{"work"},
			PipelineFile: "atkins.yml",
			Silent:       true,
			AllPipelines: []*model.Pipeline{pipeline},
			Recorder:     recorder,
		}))

		assert.NotNil(t, recorder.log)
		assert.Equal(t, "recorded", recorder.log.Metadata.Pipeline)
		assert.Equal(t, "atkins.yml", recorder.log.Metadata.File)
		assert.NotEmpty(t, recorder.log.Metadata.RunID)

		// The tree the report reads: the pipeline, the job under it, and
		// the steps under that.
		assert.NotNil(t, recorder.log.State)
		assert.Len(t, recorder.log.State.Children, 1)
		assert.Equal(t, "work - Do two things", recorder.log.State.Children[0].Name)
		assert.Len(t, recorder.log.State.Children[0].Children, 2)

		assert.NotNil(t, recorder.log.Summary)
		assert.Equal(t, eventlog.ResultPass, recorder.log.Summary.Result)
		assert.Equal(t, 2, recorder.log.Summary.TotalSteps)

		// The commands the statistics group by.
		var commands []string
		for _, event := range recorder.log.Events {
			if event.Command != "" {
				commands = append(commands, event.Command)
			}
		}
		assert.Equal(t, []string{"echo first", "echo second"}, commands)
	})

	t.Run("records a failed run too", func(t *testing.T) {
		pipeline := &model.Pipeline{
			Name: "failing",
			Jobs: map[string]*model.Job{
				"work": {Name: "work", Passthru: true, Steps: []*model.Step{{Run: "exit 3"}}},
			},
		}
		recorder := &fakeRecorder{}

		assert.Error(t, runner.RunPipeline(t.Context(), pipeline, runner.PipelineOptions{
			Jobs:         []string{"work"},
			Silent:       true,
			AllPipelines: []*model.Pipeline{pipeline},
			Recorder:     recorder,
		}))

		// A run that failed is the one most worth having a row for.
		assert.NotNil(t, recorder.log)
		assert.Equal(t, eventlog.ResultFail, recorder.log.Summary.Result)
	})

	t.Run("a recorder that fails does not fail the run", func(t *testing.T) {
		pipeline := twoStepPipeline()
		recorder := &fakeRecorder{fail: errors.New("database is locked")}

		// The pipeline has already happened and its exit code is already
		// decided; a row the recorder fails to write does not change it.
		assert.NoError(t, runner.RunPipeline(t.Context(), pipeline, runner.PipelineOptions{
			Jobs:         []string{"work"},
			Silent:       true,
			AllPipelines: []*model.Pipeline{pipeline},
			Recorder:     recorder,
		}))
		assert.NotNil(t, recorder.log)
	})

	t.Run("no recorder records nothing", func(t *testing.T) {
		pipeline := twoStepPipeline()

		assert.NoError(t, runner.RunPipeline(t.Context(), pipeline, runner.PipelineOptions{
			Jobs:         []string{"work"},
			Silent:       true,
			AllPipelines: []*model.Pipeline{pipeline},
		}))
	})
}

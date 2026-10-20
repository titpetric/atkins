package runner_test

import (
	"bytes"
	"testing"

	"github.com/titpetric/atkins/internal/assert"
	"github.com/titpetric/atkins/model"
	"github.com/titpetric/atkins/runner"
)

// TestTranscript covers the writer a caller hands RunPipeline to record
// the run somewhere else, as a locally-run CI job logs it.
func TestTranscript(t *testing.T) {
	t.Run("receives the finished tree", func(t *testing.T) {
		pipeline := &model.Pipeline{
			Name: "transcript",
			Jobs: map[string]*model.Job{
				"greet": {
					Name:     "greet",
					Desc:     "Say hello",
					Passthru: true,
					Steps:    []*model.Step{{Run: "echo hello"}},
				},
			},
		}

		var transcript bytes.Buffer
		err := runner.RunPipeline(t.Context(), pipeline, runner.PipelineOptions{
			Jobs:         []string{"greet"},
			Silent:       true,
			AllPipelines: []*model.Pipeline{pipeline},
			Transcript:   &transcript,
		})
		assert.NoError(t, err)

		assert.Contains(t, transcript.String(), "transcript")
		assert.Contains(t, transcript.String(), "greet")
		assert.Contains(t, transcript.String(), "Say hello")
	})

	t.Run("receives a failed run too", func(t *testing.T) {
		pipeline := &model.Pipeline{
			Name: "transcript",
			Jobs: map[string]*model.Job{
				"boom": {
					Name:  "boom",
					Steps: []*model.Step{{Run: "exit 3"}},
				},
			},
		}

		// A record that stops existing when the run fails is the record
		// nobody needed in the first place.
		var transcript bytes.Buffer
		err := runner.RunPipeline(t.Context(), pipeline, runner.PipelineOptions{
			Jobs:         []string{"boom"},
			Silent:       true,
			AllPipelines: []*model.Pipeline{pipeline},
			Transcript:   &transcript,
		})
		assert.Error(t, err)

		assert.Contains(t, transcript.String(), "boom")
	})

	t.Run("is optional", func(t *testing.T) {
		pipeline := &model.Pipeline{
			Name: "transcript",
			Jobs: map[string]*model.Job{
				"noop": {Name: "noop", Steps: []*model.Step{{Run: "true"}}},
			},
		}

		err := runner.RunPipeline(t.Context(), pipeline, runner.PipelineOptions{
			Jobs:         []string{"noop"},
			Silent:       true,
			AllPipelines: []*model.Pipeline{pipeline},
		})
		assert.NoError(t, err)
	})
}

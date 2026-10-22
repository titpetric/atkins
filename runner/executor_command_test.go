package runner

import (
	"bytes"
	"testing"

	"github.com/titpetric/atkins/internal/assert"
	"github.com/titpetric/atkins/model"
)

func TestShouldUseTTY(t *testing.T) {
	tests := []struct {
		name     string
		step     *model.Step
		job      *model.Job
		terminal bool
		want     bool
	}{
		{"explicit step tty, no terminal", &model.Step{TTY: true}, nil, false, true},
		{"explicit job tty, no terminal", &model.Step{}, &model.Job{TTY: true}, false, true},
		{"step passthru on a terminal", &model.Step{Passthru: true}, nil, true, true},
		{"job passthru on a terminal", &model.Step{}, &model.Job{Passthru: true}, true, true},
		{"step passthru without a terminal", &model.Step{Passthru: true}, nil, false, false},
		{"job passthru without a terminal", &model.Step{}, &model.Job{Passthru: true}, false, false},
		{"neither, on a terminal", &model.Step{}, &model.Job{}, true, false},
		{"neither, no terminal", &model.Step{}, nil, false, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, shouldUseTTY(tc.step, tc.job, tc.terminal))
		})
	}
}

// TestPassthruTTYFollowsTerminalDetection runs a job whose command
// reports whether it got a pseudo-terminal. Under go test stdout is a
// pipe, so a passthru job runs without one, and an explicit tty keeps
// forcing the allocation.
func TestPassthruTTYFollowsTerminalDetection(t *testing.T) {
	run := func(t *testing.T, job *model.Job) string {
		t.Helper()
		job.Name = "probe"
		job.Steps = []*model.Step{{Run: "test -t 1 && printf attached || printf detached"}}
		pipeline := &model.Pipeline{
			Name: "tty",
			Jobs: map[string]*model.Job{"probe": job},
		}

		var transcript bytes.Buffer
		err := RunPipeline(t.Context(), pipeline, PipelineOptions{
			Jobs:         []string{"probe"},
			Silent:       true,
			AllPipelines: []*model.Pipeline{pipeline},
			Transcript:   &transcript,
		})
		assert.NoError(t, err)
		return transcript.String()
	}

	t.Run("passthru without a terminal gets no PTY", func(t *testing.T) {
		assert.Contains(t, run(t, &model.Job{Passthru: true}), "detached")
	})

	t.Run("explicit tty forces a PTY", func(t *testing.T) {
		assert.Contains(t, run(t, &model.Job{Passthru: true, TTY: true}), "attached")
	})
}

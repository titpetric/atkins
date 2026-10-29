package runner_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/titpetric/atkins/internal/assert"
	"github.com/titpetric/atkins/model"
	"github.com/titpetric/atkins/runner"
)

// counter is a command that appends a line to a file and prints a word.
// The file is the record of how many times the command actually ran,
// which is the thing a test of this cannot take on trust.
func counter(path string) string {
	return "sh -c 'echo ran >> " + path + "; echo once'"
}

// ranCount returns how many times the counter command ran.
func ranCount(t *testing.T, path string) int {
	t.Helper()

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return 0
	}
	assert.NoError(t, err)

	return len(strings.Fields(string(data)))
}

// TestSubstitutionRunsOncePerStep covers the contract that a $(...)
// substitution executes once for the step that holds it.
//
// The tree renders a step with its label interpolated, and the step then
// runs with its command interpolated. Both read the same text. Without
// a memo the shell command inside it runs twice: once for a label nobody
// asked to have side effects, and once for the step itself.
func TestSubstitutionRunsOncePerStep(t *testing.T) {
	t.Run("a step runs its substitution once", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "count")
		pipeline := &model.Pipeline{
			Name: "substitution",
			Jobs: map[string]*model.Job{
				"work": {
					Name:     "work",
					Passthru: true,
					Steps:    []*model.Step{{Run: "echo value $(" + counter(path) + ")"}},
				},
			},
		}

		assert.NoError(t, runner.RunPipeline(t.Context(), pipeline, runner.PipelineOptions{
			Jobs:         []string{"work"},
			Silent:       true,
			AllPipelines: []*model.Pipeline{pipeline},
		}))

		assert.Equal(t, 1, ranCount(t, path))
	})

	t.Run("two steps each run their own", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "count")
		pipeline := &model.Pipeline{
			Name: "substitution",
			Jobs: map[string]*model.Job{
				"work": {
					Name:     "work",
					Passthru: true,
					Steps: []*model.Step{
						{Run: "echo first $(" + counter(path) + ")"},
						{Run: "echo second $(" + counter(path) + ")"},
					},
				},
			},
		}

		assert.NoError(t, runner.RunPipeline(t.Context(), pipeline, runner.PipelineOptions{
			Jobs:         []string{"work"},
			Silent:       true,
			AllPipelines: []*model.Pipeline{pipeline},
		}))

		// The memo is for one step. A later step with the same command
		// runs it again, because its output may have changed.
		assert.Equal(t, 2, ranCount(t, path))
	})

	t.Run("each loop iteration runs its own", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "count")
		pipeline := &model.Pipeline{
			Name: "substitution",
			Jobs: map[string]*model.Job{
				"work": {
					Name:     "work",
					Passthru: true,
					Steps: []*model.Step{{
						For: model.Iterators{"item in [1, 2, 3]"},
						Run: "echo ${{item}} $(" + counter(path) + ")",
					}},
				},
			},
		}

		assert.NoError(t, runner.RunPipeline(t.Context(), pipeline, runner.PipelineOptions{
			Jobs:         []string{"work"},
			Silent:       true,
			AllPipelines: []*model.Pipeline{pipeline},
		}))

		assert.Equal(t, 3, ranCount(t, path))
	})

	t.Run("the substitution still reaches the command", func(t *testing.T) {
		out := filepath.Join(t.TempDir(), "out")
		pipeline := &model.Pipeline{
			Name: "substitution",
			Jobs: map[string]*model.Job{
				"work": {
					Name:     "work",
					Passthru: true,
					Steps:    []*model.Step{{Run: "echo value-$(echo resolved) > " + out}},
				},
			},
		}

		assert.NoError(t, runner.RunPipeline(t.Context(), pipeline, runner.PipelineOptions{
			Jobs:         []string{"work"},
			Silent:       true,
			AllPipelines: []*model.Pipeline{pipeline},
		}))

		data, err := os.ReadFile(out)
		assert.NoError(t, err)
		assert.Equal(t, "value-resolved", strings.TrimSpace(string(data)))
	})
}

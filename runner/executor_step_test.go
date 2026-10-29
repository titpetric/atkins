package runner_test

import (
	"os"
	"testing"

	"github.com/titpetric/atkins/internal/assert"
	"github.com/titpetric/atkins/model"
	"github.com/titpetric/atkins/runner"
	"github.com/titpetric/atkins/treeview"
)

func TestStepWithMultipleCmds(t *testing.T) {
	t.Run("step with multiple cmds should display count", func(t *testing.T) {
		step := &model.Step{
			Name: "test multiple commands",
			Cmds: []string{"echo cmd1", "echo cmd2", "echo cmd3"},
		}

		// String() should show the command count
		result := step.String()
		assert.Equal(t, "cmds: <3 commands>", result)
		assert.NotContains(t, result, "&&")
		assert.NotContains(t, result, "echo cmd1")
	})
}

func TestSkippedStepDisplaysInterpolatedCondition(t *testing.T) {
	repoPath := t.TempDir()
	step := &model.Step{
		Run: "echo should not run",
		If:  model.Conditionals{"$(test ! -d ${{repo.path}})"},
	}
	job := &model.Job{
		Name:  "setup",
		Steps: []*model.Step{step},
	}

	display := treeview.NewSilentDisplay()
	builder := treeview.NewBuilder("test")
	jobNode := builder.AddJob(job, nil, job.Name)
	ctx := &runner.ExecutionContext{
		Variables: runner.NewContextVariables(map[string]any{
			"repo": map[string]any{"path": repoPath},
		}),
		Env:        make(map[string]string),
		Dir:        repoPath,
		Job:        job,
		CurrentJob: jobNode,
		Display:    display,
		Builder:    builder,
	}

	err := runner.NewExecutor().ExecuteJob(t.Context(), ctx)
	assert.NoError(t, err)

	stepNode := jobNode.GetChildren()[0].Node
	assert.Equal(t, treeview.StatusSkipped, stepNode.GetStatus())
	assert.Equal(t, "$(test ! -d "+repoPath+")", stepNode.GetIf())
}

func TestEchoStep_RunsOnce(t *testing.T) {
	// Echo steps have their stdout promoted to the tree label. That output
	// comes from the single run of the command; re-running it to read the
	// output would apply the redirect twice.
	tmpDir := t.TempDir()
	pipeline := &model.Pipeline{
		Name: "echo once",
		Dir:  tmpDir,
		Jobs: map[string]*model.Job{
			"echoer": {
				Name:  "echoer",
				Steps: []*model.Step{{Run: "echo appended >> log"}},
			},
		},
	}

	err := runner.RunPipeline(t.Context(), pipeline, runner.PipelineOptions{
		Jobs:         []string{"echoer"},
		Silent:       true,
		AllPipelines: []*model.Pipeline{pipeline},
	})
	assert.NoError(t, err)

	log, readErr := os.ReadFile(tmpDir + "/log")
	assert.NoError(t, readErr)
	assert.Equal(t, "appended\n", string(log))
}

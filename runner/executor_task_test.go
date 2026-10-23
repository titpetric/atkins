package runner_test

import (
	"os"
	"testing"

	"github.com/titpetric/atkins/internal/assert"
	"github.com/titpetric/atkins/model"
	"github.com/titpetric/atkins/runner"
)

func TestTaskLoopWithLazyCallSiteVar(t *testing.T) {
	tmpDir := t.TempDir()
	counterFile := tmpDir + "/evaluations"
	outputFile := tmpDir + "/output"

	task := &model.Job{
		Name:     "format-one",
		Requires: []string{"file"},
		Steps: []*model.Step{
			{Run: `printf '%s\n' "${{ file }}" >> "` + outputFile + `"`},
		},
	}
	caller := &model.Job{
		Name: "fmt",
		Steps: []*model.Step{
			{
				Task: "format-one",
				For:  model.Iterators{"file in ${{ files }}"},
				Decl: &model.Decl{Vars: map[string]any{
					"files": `$(printf x >> "` + counterFile + `"; printf 'one.md\ntwo.md\n')`,
				}},
			},
		},
	}
	pipeline := &model.Pipeline{
		Name: "task loop",
		Dir:  tmpDir,
		Jobs: map[string]*model.Job{
			"fmt":        caller,
			"format-one": task,
		},
	}

	err := runner.RunPipeline(t.Context(), pipeline, runner.PipelineOptions{
		Jobs:         []string{"fmt"},
		Silent:       true,
		AllPipelines: []*model.Pipeline{pipeline},
	})
	assert.NoError(t, err)

	output, err := os.ReadFile(outputFile)
	assert.NoError(t, err)
	assert.Equal(t, "one.md\ntwo.md\n", string(output))

	evaluations, err := os.ReadFile(counterFile)
	assert.NoError(t, err)
	assert.Equal(t, "x", string(evaluations), "loop source var should be evaluated once")
}

func TestDeferredTaskLoopExpandsOnce(t *testing.T) {
	tmpDir := t.TempDir()
	outputFile := tmpDir + "/output"

	task := &model.Job{
		Name:     "record",
		Requires: []string{"item"},
		Steps: []*model.Step{
			{Run: `printf '%s\n' "${{ item }}" >> "` + outputFile + `"`},
		},
	}
	caller := &model.Job{
		Name: "caller",
		Steps: []*model.Step{
			{
				Task:     "record",
				For:      model.Iterators{"item in items"},
				Deferred: true,
				Decl: &model.Decl{Vars: map[string]any{
					"items": []any{"one", "two"},
				}},
			},
		},
	}
	pipeline := &model.Pipeline{
		Name: "deferred task loop",
		Dir:  tmpDir,
		Jobs: map[string]*model.Job{
			"caller": caller,
			"record": task,
		},
	}

	err := runner.RunPipeline(t.Context(), pipeline, runner.PipelineOptions{
		Jobs:         []string{"caller"},
		Silent:       true,
		AllPipelines: []*model.Pipeline{pipeline},
	})
	assert.NoError(t, err)

	output, err := os.ReadFile(outputFile)
	assert.NoError(t, err)
	assert.Equal(t, "one\ntwo\n", string(output))
}

func TestTaskInvocationWithForLoop(t *testing.T) {
	t.Run("expand for loop with task variables", func(t *testing.T) {
		// Create a step that invokes a task with a for loop
		step := &model.Step{
			Name: "build all components",
			Task: "build_component",
			For:  model.Iterators{"component in components"},
		}

		// Create execution context
		ctx := &runner.ExecutionContext{
			Variables: runner.NewContextVariables(map[string]any{
				"components": []any{"src/main", "src/utils", "tests/"},
			}),
			Step: step,
			Env:  make(map[string]string),
		}

		// Expand the for loop
		iterations, err := runner.ExpandFor(ctx, func(cmd string) (string, error) {
			return "", nil
		})
		assert.NoError(t, err)

		assert.Equal(t, 3, len(iterations))

		// Verify each iteration has the component variable
		expectedComponents := []string{"src/main", "src/utils", "tests/"}
		for i, iter := range iterations {
			assert.Equal(t, expectedComponents[i], iter.Variables.Get("component"), "Iteration %d component mismatch", i)
		}
	})

	t.Run("task requires variable from for loop", func(t *testing.T) {
		// Create a task that requires the loop variable
		task := &model.Job{
			Name:     "build_component",
			Requires: []string{"component"},
		}

		// Simulate iteration context with loop variable
		ctx := &runner.ExecutionContext{
			Variables: runner.NewContextVariables(map[string]any{
				"component": "src/main",
			}),
		}

		// Should pass validation
		err := runner.ValidateJobRequirements(ctx, task)
		assert.NoError(t, err)
	})

	t.Run("task requires variable missing from for loop context", func(t *testing.T) {
		// Create a task that requires a variable
		task := &model.Job{
			Name:     "build_component",
			Requires: []string{"component"},
		}

		// Iteration context without the required variable
		ctx := &runner.ExecutionContext{
			Variables: runner.NewContextVariables(nil),
		}

		// Should fail validation
		err := runner.ValidateJobRequirements(ctx, task)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "component")
	})
}

func TestInvokedTask_SkipsWhenSubshellConditionFalse(t *testing.T) {
	tmpDir := t.TempDir()
	marker := tmpDir + "/should-not-exist"

	conditionalTask := &model.Job{
		Name: "conditional",
		If:   model.Conditionals{"$(test -f missing-file)"},
		Steps: []*model.Step{
			{Run: "touch " + marker},
		},
	}
	caller := &model.Job{
		Name: "caller",
		Steps: []*model.Step{
			{Task: "conditional"},
		},
	}
	pipeline := &model.Pipeline{
		Name: "conditional task",
		Dir:  tmpDir,
		Jobs: map[string]*model.Job{
			"caller":      caller,
			"conditional": conditionalTask,
		},
	}

	err := runner.RunPipeline(t.Context(), pipeline, runner.PipelineOptions{
		Jobs:         []string{"caller"},
		Silent:       true,
		AllPipelines: []*model.Pipeline{pipeline},
	})
	assert.NoError(t, err)
	_, statErr := os.Stat(marker)
	assert.ErrorIs(t, statErr, os.ErrNotExist)
}

func TestTaskStep_MultipleTaskNames(t *testing.T) {
	newPipeline := func(dir, task string) *model.Pipeline {
		return &model.Pipeline{
			Name: "multi task",
			Dir:  dir,
			Jobs: map[string]*model.Job{
				"caller": {
					Name:  "caller",
					Steps: []*model.Step{{Task: task}},
				},
				"one":  {Name: "one", Steps: []*model.Step{{Run: "printf 'one\\n' >> log"}}},
				"two":  {Name: "two", Steps: []*model.Step{{Run: "printf 'two\\n' >> log"}}},
				"boom": {Name: "boom", Steps: []*model.Step{{Run: "exit 3"}}},
			},
		}
	}

	run := func(t *testing.T, task string) (string, error) {
		t.Helper()

		tmpDir := t.TempDir()
		pipeline := newPipeline(tmpDir, task)
		err := runner.RunPipeline(t.Context(), pipeline, runner.PipelineOptions{
			Jobs:         []string{"caller"},
			Silent:       true,
			AllPipelines: []*model.Pipeline{pipeline},
		})

		log, readErr := os.ReadFile(tmpDir + "/log")
		if readErr != nil {
			return "", err
		}
		return string(log), err
	}

	t.Run("runs each named task in order", func(t *testing.T) {
		log, err := run(t, "one two one")
		assert.NoError(t, err)
		assert.Equal(t, "one\ntwo\none\n", log)
	})

	t.Run("stops at the first failing task", func(t *testing.T) {
		log, err := run(t, "one boom two")
		assert.Error(t, err)
		assert.Equal(t, "one\n", log)
	})
}

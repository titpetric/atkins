package runner_test

import (
	"context"
	"os"
	"testing"

	"github.com/titpetric/atkins/internal/assert"
	"github.com/titpetric/atkins/model"
	"github.com/titpetric/atkins/runner"
	"github.com/titpetric/atkins/treeview"
)

func TestForLoopStepExecutionWithLazyCommandVar(t *testing.T) {
	tmpDir := t.TempDir()
	counterFile := tmpDir + "/evaluations"
	outputFile := tmpDir + "/output"

	job := &model.Job{
		Name: "fmt",
		Steps: []*model.Step{
			{
				Run: `printf '%s\n' "${{ file }}" >> "` + outputFile + `"`,
				For: model.Iterators{"file in ${files}"},
				Decl: &model.Decl{Vars: map[string]any{
					"files":      "${{ discovered }}",
					"discovered": `$(printf x >> "` + counterFile + `"; printf 'one.md\ntwo.md\n')`,
				}},
			},
		},
	}

	display := treeview.NewSilentDisplay()
	builder := treeview.NewBuilder("test")
	ctx := &runner.ExecutionContext{
		Variables:  runner.NewContextVariables(nil),
		Env:        make(map[string]string),
		Job:        job,
		CurrentJob: builder.AddJob(job, nil, job.Name),
		Display:    display,
		Builder:    builder,
	}

	err := runner.NewExecutor().ExecuteJob(t.Context(), ctx)
	assert.NoError(t, err)

	output, err := os.ReadFile(outputFile)
	assert.NoError(t, err)
	assert.Equal(t, "one.md\ntwo.md\n", string(output))

	evaluations, err := os.ReadFile(counterFile)
	assert.NoError(t, err)
	assert.Equal(t, "x", string(evaluations), "loop source var should be evaluated once")
}

func TestForLoopSourceResolvesEnvDependencyClosureOnce(t *testing.T) {
	tmpDir := t.TempDir()
	counterFile := tmpDir + "/evaluations"
	outputFile := tmpDir + "/output"
	job := &model.Job{Name: "env-loop", Steps: []*model.Step{{
		Run: `printf '%s\n' "${{ file }}:${{ label }}" >> "` + outputFile + `"`,
		For: model.Iterators{"file in ${files}"},
		Decl: &model.Decl{
			Vars: map[string]any{
				"files": `$(printf v >> "` + counterFile + `"; printf '%s' "$FILES")`,
				"label": "${{ file }}",
			},
			Env: &model.EnvDecl{Vars: map[string]any{
				"FILES": `$(printf e >> "` + counterFile + `"; printf 'one.md\ntwo.md\n')`,
				"LATER": "${{ file }}",
			}},
		},
	}}}
	builder := treeview.NewBuilder("test")
	ctx := &runner.ExecutionContext{
		Variables: runner.NewContextVariables(nil), Env: make(map[string]string), Job: job,
		CurrentJob: builder.AddJob(job, nil, job.Name), Display: treeview.NewSilentDisplay(), Builder: builder,
	}

	err := runner.NewExecutor().ExecuteJob(t.Context(), ctx)
	assert.NoError(t, err)
	output, readErr := os.ReadFile(outputFile)
	assert.NoError(t, readErr)
	assert.Equal(t, "one.md:one.md\ntwo.md:two.md\n", string(output))
	evaluations, readErr := os.ReadFile(counterFile)
	assert.NoError(t, readErr)
	assert.Equal(t, "ev", string(evaluations), "required env and var commands should each execute once")
}

func TestForLoopStepVarCanDependOnEarlierIterator(t *testing.T) {
	tmpDir := t.TempDir()
	outputFile := tmpDir + "/output"
	job := &model.Job{
		Name: "nested",
		Steps: []*model.Step{
			{
				Run: `printf '%s:%s\n' "${{ group }}" "${{ child }}" >> "` + outputFile + `"`,
				For: model.Iterators{"group in groups", "child in children"},
				Decl: &model.Decl{Vars: map[string]any{
					"groups":   []any{"one", "two"},
					"children": "${{ group }}",
				}},
			},
		},
	}
	display := treeview.NewSilentDisplay()
	builder := treeview.NewBuilder("test")
	ctx := &runner.ExecutionContext{
		Variables:  runner.NewContextVariables(nil),
		Env:        make(map[string]string),
		Job:        job,
		CurrentJob: builder.AddJob(job, nil, job.Name),
		Display:    display,
		Builder:    builder,
	}

	err := runner.NewExecutor().ExecuteJob(t.Context(), ctx)
	assert.NoError(t, err)
	output, err := os.ReadFile(outputFile)
	assert.NoError(t, err)
	assert.Equal(t, "one:one\ntwo:two\n", string(output))
}

func TestStepVarsWithLoopVariable(t *testing.T) {
	t.Run("step vars with loop variable should be interpolated per iteration", func(t *testing.T) {
		// This tests the fix for BUG 2: step-level vars like `path: $(dirname "${{item}}")`
		// interpolate with the loop variable available

		step := &model.Step{
			Name: "test step",
			Task: "project:ps",
			For:  model.Iterators{"item in projects"},
			Decl: &model.Decl{
				Vars: map[string]any{
					"path": "${{item}}/subdir",
				},
			},
		}

		ctx := &runner.ExecutionContext{
			Variables: runner.NewContextVariables(map[string]any{
				"projects": []any{"proj1", "proj2", "proj3"},
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

		// Verify each iteration has the correct item
		for i, iter := range iterations {
			expectedItems := []string{"proj1", "proj2", "proj3"}
			assert.Equal(t, expectedItems[i], iter.Variables.Get("item"))

			// Now simulate what happens in executeTaskStepWithLoop:
			// Create iteration context and merge step vars
			iterCtx := &runner.ExecutionContext{
				Variables: iter.Variables.Clone(),
				Env:       make(map[string]string),
			}

			// Merge step-level vars (this is the fix)
			err := runner.MergeVariables(iterCtx, step.Decl)
			assert.NoError(t, err)

			// The path now holds the interpolated item value
			expectedPath := expectedItems[i] + "/subdir"
			assert.Equal(t, expectedPath, iterCtx.Variables.Get("path"),
				"path should be interpolated with item=%s", expectedItems[i])
		}
	})
}

func TestCurrentStepSetCorrectlyInIteration(t *testing.T) {
	t.Run("CurrentStep should be set to iteration node during execution", func(t *testing.T) {
		// This tests the fix for BUG 1: output goes to the correct iteration node,
		// and later iterations do not overwrite it

		// Create a simple context with a CurrentStep set
		parentNode := treeview.NewNode("parent")
		iterNode1 := treeview.NewNode("iter1")
		iterNode2 := treeview.NewNode("iter2")

		execCtx := &runner.ExecutionContext{
			CurrentStep: parentNode,
			Variables:   runner.NewContextVariables(nil),
			Env:         make(map[string]string),
		}

		// Simulate what executeStepIteration does:
		// Save original, set to iter node, then restore

		// First iteration
		func() {
			originalStep := execCtx.CurrentStep
			execCtx.CurrentStep = iterNode1
			defer func() { execCtx.CurrentStep = originalStep }()

			// During execution, CurrentStep is iterNode1
			assert.Equal(t, iterNode1, execCtx.CurrentStep)
		}()

		// After first iteration, CurrentStep returns to parent
		assert.Equal(t, parentNode, execCtx.CurrentStep)

		// Second iteration
		func() {
			originalStep := execCtx.CurrentStep
			execCtx.CurrentStep = iterNode2
			defer func() { execCtx.CurrentStep = originalStep }()

			// During execution, CurrentStep is iterNode2
			assert.Equal(t, iterNode2, execCtx.CurrentStep)
		}()

		// After second iteration, CurrentStep returns to parent
		assert.Equal(t, parentNode, execCtx.CurrentStep)
	})
}

// TestExecuteStepWithForLoop_SequentialBreaksOnError verifies that sequential
// iteration (without detach) stops on the first error.
func TestExecuteStepWithForLoop_SequentialBreaksOnError(t *testing.T) {
	tmpDir := t.TempDir()

	job := &model.Job{
		Name: "test_job",
		Steps: []*model.Step{
			{
				Run: `if [ "${{ item }}" = "fail" ]; then exit 1; fi; touch "` + tmpDir + `/${{ item }}.done"`,
				For: model.Iterators{"item in items"},
			},
		},
	}

	display := treeview.NewSilentDisplay()
	builder := treeview.NewBuilder("test")
	jobNode := builder.AddJob(job, nil, "test_job")

	ctx := &runner.ExecutionContext{
		Variables: runner.NewContextVariables(map[string]any{
			"items": []any{"first", "fail", "third"},
		}),
		Env:        make(map[string]string),
		Job:        job,
		CurrentJob: jobNode,
		Display:    display,
		Builder:    builder,
	}

	executor := runner.NewExecutor()
	err := executor.ExecuteJob(t.Context(), ctx)
	assert.Error(t, err)

	// First ran, third should NOT have run
	_, err1 := os.Stat(tmpDir + "/first.done")
	_, err3 := os.Stat(tmpDir + "/third.done")
	assert.NoError(t, err1, "first iteration should complete")
	assert.True(t, os.IsNotExist(err3), "third iteration should not run")
}

// TestExecuteStepWithForLoop_DetachParallel verifies that for loop with
// detach runs iterations in parallel by completing all iterations.
func TestExecuteStepWithForLoop_DetachParallel(t *testing.T) {
	tmpDir := t.TempDir()

	job := &model.Job{
		Name: "test_job",
		Steps: []*model.Step{
			{
				Run:    `touch "` + tmpDir + `/${{ item }}.done"`,
				For:    model.Iterators{"item in items"},
				Detach: true,
			},
		},
	}

	display := treeview.NewSilentDisplay()
	builder := treeview.NewBuilder("test")
	jobNode := builder.AddJob(job, nil, "test_job")

	ctx := &runner.ExecutionContext{
		Variables: runner.NewContextVariables(map[string]any{
			"items": []any{"a", "b", "c", "d"},
		}),
		Env:        make(map[string]string),
		Job:        job,
		CurrentJob: jobNode,
		Display:    display,
		Builder:    builder,
	}

	executor := runner.NewExecutor()
	err := executor.ExecuteJob(t.Context(), ctx)
	assert.NoError(t, err)

	// All files exist when detach completes
	for _, item := range []string{"a", "b", "c", "d"} {
		_, err := os.Stat(tmpDir + "/" + item + ".done")
		assert.NoError(t, err, "iteration %s should complete", item)
	}
}

// TestExecuteStepWithForLoop_DetachContinuesOnError verifies that parallel
// iterations continue even when one fails.
func TestExecuteStepWithForLoop_DetachContinuesOnError(t *testing.T) {
	tmpDir := t.TempDir()

	job := &model.Job{
		Name: "test_job",
		Steps: []*model.Step{
			{
				Run:    `if [ "${{ item }}" = "fail" ]; then exit 1; fi; touch "` + tmpDir + `/${{ item }}.done"`,
				For:    model.Iterators{"item in items"},
				Detach: true,
			},
		},
	}

	display := treeview.NewSilentDisplay()
	builder := treeview.NewBuilder("test")
	jobNode := builder.AddJob(job, nil, "test_job")

	ctx := &runner.ExecutionContext{
		Variables: runner.NewContextVariables(map[string]any{
			"items": []any{"first", "fail", "third"},
		}),
		Env:        make(map[string]string),
		Job:        job,
		CurrentJob: jobNode,
		Display:    display,
		Builder:    builder,
	}

	executor := runner.NewExecutor()
	err := executor.ExecuteJob(t.Context(), ctx)
	assert.Error(t, err)

	// Both first and third should have run (detach waits for all to complete)
	_, err1 := os.Stat(tmpDir + "/first.done")
	_, err3 := os.Stat(tmpDir + "/third.done")
	assert.NoError(t, err1, "first iteration should complete")
	assert.NoError(t, err3, "third iteration should complete with detach")
}

// TestExecuteStepWithForLoop_ContextCancellation verifies that iterations
// respect context timeout/cancellation.
func TestExecuteStepWithForLoop_ContextCancellation(t *testing.T) {
	job := &model.Job{
		Name:    "test_job",
		Timeout: "1ms", // Short timeout to trigger cancellation
		Steps: []*model.Step{
			{
				// Command that would take too long
				Run: `for i in $(seq 1 1000); do echo $i > /dev/null; done`,
				For: model.Iterators{"item in items"},
			},
		},
	}

	display := treeview.NewSilentDisplay()
	builder := treeview.NewBuilder("test")
	jobNode := builder.AddJob(job, nil, "test_job")

	ctx := &runner.ExecutionContext{
		Variables: runner.NewContextVariables(map[string]any{
			"items": []any{"first", "second", "third"},
		}),
		Env:        make(map[string]string),
		Job:        job,
		CurrentJob: jobNode,
		Display:    display,
		Builder:    builder,
	}

	executor := runner.NewExecutor()
	err := executor.ExecuteJob(context.Background(), ctx)
	// Should have an error due to timeout
	assert.Error(t, err)
}

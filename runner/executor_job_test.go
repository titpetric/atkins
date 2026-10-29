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

func TestExecuteJob_SkipsWhenIfConditionFalse(t *testing.T) {
	t.Run("job with false if condition returns ErrJobSkipped", func(t *testing.T) {
		job := &model.Job{
			Name: "conditional_job",
			If:   model.Conditionals{"false"},
			Steps: []*model.Step{
				{Run: "echo should not run"},
			},
		}

		ctx := &runner.ExecutionContext{
			Variables: runner.NewContextVariables(nil),
			Env:       make(map[string]string),
			Job:       job,
		}

		executor := runner.NewExecutor()
		err := executor.ExecuteJob(t.Context(), ctx)
		assert.ErrorIs(t, err, runner.ErrJobSkipped)
	})

	t.Run("job with true if condition runs normally", func(t *testing.T) {
		job := &model.Job{
			Name: "conditional_job",
			If:   model.Conditionals{"true"},
			Steps: []*model.Step{
				{Run: "echo hello"},
			},
		}

		display := treeview.NewSilentDisplay()
		builder := treeview.NewBuilder("test")
		jobNode := builder.AddJob(job, nil, "conditional_job")

		ctx := &runner.ExecutionContext{
			Variables:  runner.NewContextVariables(nil),
			Env:        make(map[string]string),
			Job:        job,
			CurrentJob: jobNode,
			Display:    display,
			Builder:    builder,
		}

		executor := runner.NewExecutor()
		err := executor.ExecuteJob(t.Context(), ctx)
		assert.NoError(t, err)
	})

	t.Run("job with variable-based if condition skipped", func(t *testing.T) {
		job := &model.Job{
			Name: "notification",
			If:   model.Conditionals{"last_sent_match < last_inbox_match"},
			Decl: &model.Decl{
				Vars: map[string]any{
					"last_inbox_match": "2026-02-23T12:23:36Z",
					"last_sent_match":  "2026-03-02T20:12:59Z",
				},
			},
			Steps: []*model.Step{
				{Run: "echo Hello"},
			},
		}

		ctx := &runner.ExecutionContext{
			Variables: runner.NewContextVariables(nil),
			Env:       make(map[string]string),
			Job:       job,
		}

		executor := runner.NewExecutor()
		err := executor.ExecuteJob(t.Context(), ctx)
		assert.ErrorIs(t, err, runner.ErrJobSkipped)
	})
}

// TestExecuteJob_DirBeforeVars verifies that ExecuteJob evaluates job.Dir
// BEFORE merging variables. $(cmd) in vars runs in the correct directory.
func TestExecuteJob_DirBeforeVars(t *testing.T) {
	t.Run("subshell pwd uses job dir", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "test-workdir-*")
		assert.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		job := &model.Job{
			Name: "test_job",
			Dir:  tmpDir,
			Decl: &model.Decl{
				Vars: map[string]any{
					"current_dir": "$(pwd)",
				},
			},
			Steps: []*model.Step{
				{Run: "true"},
			},
		}

		display := treeview.NewSilentDisplay()
		builder := treeview.NewBuilder("test")
		jobNode := builder.AddJob(job, nil, "test_job")

		ctx := &runner.ExecutionContext{
			Variables:  runner.NewContextVariables(nil),
			Env:        make(map[string]string),
			Job:        job,
			CurrentJob: jobNode,
			Display:    display,
			Builder:    builder,
		}

		executor := runner.NewExecutor()
		err = executor.ExecuteJob(t.Context(), ctx)
		assert.NoError(t, err)
		assert.Equal(t, tmpDir, ctx.Variables.Get("current_dir"))
	})

	t.Run("subshell ls lists files from job dir", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "test-workdir-*")
		assert.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		testFile := tmpDir + "/marker.txt"
		assert.NoError(t, os.WriteFile(testFile, []byte("x"), 0o644))

		job := &model.Job{
			Name: "test_job",
			Dir:  tmpDir,
			Decl: &model.Decl{
				Vars: map[string]any{
					"files": "$(ls)",
				},
			},
			Steps: []*model.Step{
				{Run: "true"},
			},
		}

		display := treeview.NewSilentDisplay()
		builder := treeview.NewBuilder("test")
		jobNode := builder.AddJob(job, nil, "test_job")

		ctx := &runner.ExecutionContext{
			Variables:  runner.NewContextVariables(nil),
			Env:        make(map[string]string),
			Job:        job,
			CurrentJob: jobNode,
			Display:    display,
			Builder:    builder,
		}

		executor := runner.NewExecutor()
		err = executor.ExecuteJob(t.Context(), ctx)
		assert.NoError(t, err)
		assert.Contains(t, ctx.Variables.Get("files"), "marker.txt")
	})

	t.Run("dir from interpolated var with subshell", func(t *testing.T) {
		// Tests the case where dir depends on a var that comes from a subshell:
		// - dir: ${{workdir}}
		// - vars: { workdir: "$(echo /tmp)", files_in_wd: "$(ls)" }
		// The expected resolution order:
		// 1. Resolve workdir (dir depends on it)
		// 2. Resolve dir with workdir available
		// 3. Resolve files_in_wd with dir as cwd
		tmpDir, err := os.MkdirTemp("", "test-workdir-*")
		assert.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		assert.NoError(t, os.WriteFile(tmpDir+"/subshell_marker.txt", []byte("x"), 0o644))

		job := &model.Job{
			Name: "test_job",
			Dir:  "${{ workdir }}",
			Decl: &model.Decl{
				Vars: map[string]any{
					"workdir":     "$(echo " + tmpDir + ")",
					"files_in_wd": "$(ls)",
				},
			},
			Steps: []*model.Step{
				{Run: "true"},
			},
		}

		display := treeview.NewSilentDisplay()
		builder := treeview.NewBuilder("test")
		jobNode := builder.AddJob(job, nil, "test_job")

		ctx := &runner.ExecutionContext{
			Variables:  runner.NewContextVariables(nil),
			Env:        make(map[string]string),
			Job:        job,
			CurrentJob: jobNode,
			Display:    display,
			Builder:    builder,
		}

		executor := runner.NewExecutor()
		err = executor.ExecuteJob(t.Context(), ctx)

		assert.NoError(t, err)
		assert.Equal(t, tmpDir, ctx.Variables.Get("workdir"))
		assert.Contains(t, ctx.Variables.Get("files_in_wd"), "subshell_marker.txt")
	})
}

func TestExecuteJob_ForLoop(t *testing.T) {
	t.Run("job-level for loop runs steps for each iteration", func(t *testing.T) {
		job := &model.Job{
			Name: "loop_job",
			Desc: "Iteration ${{ item }}",
			For:  model.Iterators{"item in items"},
			Steps: []*model.Step{
				{Run: "echo ${{ item }}"},
			},
		}

		display := treeview.NewSilentDisplay()
		builder := treeview.NewBuilder("test")
		jobNode := builder.AddJob(job, nil, "loop_job")

		ctx := &runner.ExecutionContext{
			Variables: runner.NewContextVariables(map[string]any{
				"items": []any{"alpha", "beta", "gamma"},
			}),
			Env:        make(map[string]string),
			Job:        job,
			CurrentJob: jobNode,
			Display:    display,
			Builder:    builder,
		}

		executor := runner.NewExecutor()
		err := executor.ExecuteJob(context.Background(), ctx)
		assert.NoError(t, err)

		// Job node should have 3 iteration children (pre-built steps replaced)
		children := jobNode.GetChildren()
		assert.Len(t, children, 3)
		assert.Equal(t, "Iteration alpha", children[0].GetName())
		assert.Equal(t, "Iteration beta", children[1].GetName())
		assert.Equal(t, "Iteration gamma", children[2].GetName())

		for _, child := range children {
			assert.Equal(t, treeview.StatusPassed, child.GetStatus())
		}
	})

	t.Run("job-level for loop with dir referencing loop variable", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "test-jobloop-*")
		assert.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		// Create subdirectories
		for _, name := range []string{"aaa", "bbb"} {
			assert.NoError(t, os.MkdirAll(tmpDir+"/"+name, 0o755))
		}

		job := &model.Job{
			Name: "dir_loop",
			Desc: "${{ folder }}",
			Dir:  tmpDir + "/${{ folder }}",
			For:  model.Iterators{"folder in folders"},
			Steps: []*model.Step{
				{Run: "echo hello"},
			},
		}

		display := treeview.NewSilentDisplay()
		builder := treeview.NewBuilder("test")
		jobNode := builder.AddJob(job, nil, "dir_loop")

		ctx := &runner.ExecutionContext{
			Variables: runner.NewContextVariables(map[string]any{
				"folders": []any{"aaa", "bbb"},
			}),
			Env:        make(map[string]string),
			Job:        job,
			CurrentJob: jobNode,
			Display:    display,
			Builder:    builder,
		}

		executor := runner.NewExecutor()
		err = executor.ExecuteJob(context.Background(), ctx)
		assert.NoError(t, err)

		children := jobNode.GetChildren()
		assert.Len(t, children, 2)
		assert.Equal(t, "aaa", children[0].GetName())
		assert.Equal(t, "bbb", children[1].GetName())
	})

	t.Run("job-level for loop with if condition referencing loop variable", func(t *testing.T) {
		job := &model.Job{
			Name: "if_loop",
			For:  model.Iterators{"item in items"},
			If:   model.Conditionals{`item != "skip"`},
			Steps: []*model.Step{
				{Run: "echo ${{ item }}"},
			},
		}

		display := treeview.NewSilentDisplay()
		builder := treeview.NewBuilder("test")
		jobNode := builder.AddJob(job, nil, "if_loop")

		ctx := &runner.ExecutionContext{
			Variables: runner.NewContextVariables(map[string]any{
				"items": []any{"one", "skip", "three"},
			}),
			Env:        make(map[string]string),
			Job:        job,
			CurrentJob: jobNode,
			Display:    display,
			Builder:    builder,
		}

		executor := runner.NewExecutor()
		err := executor.ExecuteJob(context.Background(), ctx)
		assert.NoError(t, err)

		// The if condition drops the "skip" iteration, leaving 2 iterations
		children := jobNode.GetChildren()
		assert.Len(t, children, 2)
	})

	t.Run("job-level for loop with empty iterations", func(t *testing.T) {
		job := &model.Job{
			Name: "empty_loop",
			For:  model.Iterators{"item in items"},
			Steps: []*model.Step{
				{Run: "echo ${{ item }}"},
			},
		}

		display := treeview.NewSilentDisplay()
		builder := treeview.NewBuilder("test")
		jobNode := builder.AddJob(job, nil, "empty_loop")

		ctx := &runner.ExecutionContext{
			Variables: runner.NewContextVariables(map[string]any{
				"items": []any{},
			}),
			Env:        make(map[string]string),
			Job:        job,
			CurrentJob: jobNode,
			Display:    display,
			Builder:    builder,
		}

		executor := runner.NewExecutor()
		err := executor.ExecuteJob(context.Background(), ctx)
		assert.NoError(t, err)
	})
}

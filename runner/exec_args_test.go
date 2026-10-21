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

// TestExecStep_Args checks an exec: step runs its argv without a shell
// and a job with args: true appends the command line arguments as
// literal argv, while a job without it ignores them.
func TestExecStep_Args(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	out := filepath.Join(dir, "argv.out")
	script := filepath.Join(dir, "argv.sh")
	assert.NoError(t, os.WriteFile(script, []byte("#!/bin/sh\nfor a in \"$@\"; do printf '%s\\n' \"$a\"; done > \""+out+"\"\n"), 0o755))

	run := func(t *testing.T, takesArgs bool, args ...string) []string {
		t.Helper()
		assert.NoError(t, os.RemoveAll(out))

		job := &model.Job{
			Name:  "custom",
			Args:  takesArgs,
			Steps: []*model.Step{{Exec: []string{script, "compose", "${{ flag }}"}}},
		}
		pipeline := &model.Pipeline{
			Name: "exec",
			Dir:  dir,
			Decl: &model.Decl{Vars: map[string]any{"flag": "--ansi=never"}},
			Jobs: map[string]*model.Job{"custom": job},
		}

		err := runner.RunPipeline(t.Context(), pipeline, runner.PipelineOptions{
			Jobs:         []string{"custom"},
			Silent:       true,
			AllPipelines: []*model.Pipeline{pipeline},
			Args:         args,
		})
		assert.NoError(t, err)

		data, err := os.ReadFile(out)
		assert.NoError(t, err)
		return strings.Split(strings.TrimSuffix(string(data), "\n"), "\n")
	}

	t.Run("two argv", func(t *testing.T) {
		assert.Equal(t, []string{"compose", "--ansi=never", "ps", "-a"}, run(t, true, "ps", "-a"))
	})

	t.Run("shell syntax stays literal", func(t *testing.T) {
		got := run(t, true, "; rm -rf "+dir, "$(id)", "${{ flag }}", "*")
		assert.Equal(t, []string{"compose", "--ansi=never", "; rm -rf " + dir, "$(id)", "${{ flag }}", "*"}, got)
		_, err := os.Stat(dir)
		assert.NoError(t, err)
	})

	t.Run("no args: true", func(t *testing.T) {
		assert.Equal(t, []string{"compose", "--ansi=never"}, run(t, false, "ps"))
	})
}

// TestRunPipeline_InactiveJob checks running a job whose when: block
// does not match fails, also through a task: reference.
func TestRunPipeline_InactiveJob(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	build := &model.Job{Name: "build", Inactive: true, Steps: []*model.Step{{Run: "true"}}}
	all := &model.Job{Name: "all", Steps: []*model.Step{{Task: "build"}}}
	pipeline := &model.Pipeline{
		Name: "gated",
		Dir:  dir,
		Jobs: map[string]*model.Job{"build": build, "all": all},
	}

	for _, job := range []string{"build", "all"} {
		err := runner.RunPipeline(t.Context(), pipeline, runner.PipelineOptions{
			Jobs:         []string{job},
			Silent:       true,
			AllPipelines: []*model.Pipeline{pipeline},
		})
		assert.ErrorContains(t, err, "does not apply here")
	}
}

package runner

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/titpetric/atkins/internal/assert"
	"github.com/titpetric/atkins/model"
)

// TestTTYFollowsTerminalDetection runs a job whose command reports
// whether it got a pseudo-terminal. Under go test stdout is a pipe, so
// no step gets one, with or without passthru.
func TestTTYFollowsTerminalDetection(t *testing.T) {
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

	t.Run("plain step without a terminal gets no PTY", func(t *testing.T) {
		assert.Contains(t, run(t, &model.Job{}), "detached")
	})
}

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

		err := RunPipeline(t.Context(), pipeline, PipelineOptions{
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

// TestExecStep_Output checks an exec: step prints its output on success,
// and passthru: false keeps it quiet.
func TestExecStep_Output(t *testing.T) {
	t.Parallel()

	run := func(t *testing.T, step string) string {
		t.Helper()

		src := "name: exec\njobs:\n  custom:\n    args: true\n    steps:\n      - exec: [printf, \"%s-line\\n\"]\n" + step
		pipelines, err := LoadPipelineFromReader(strings.NewReader(src))
		assert.NoError(t, err)
		pipeline := pipelines[0]
		pipeline.Dir = t.TempDir()

		var transcript bytes.Buffer
		err = RunPipeline(t.Context(), pipeline, PipelineOptions{
			Jobs:         []string{"custom"},
			Silent:       true,
			AllPipelines: []*model.Pipeline{pipeline},
			Args:         []string{"exec-output"},
			Transcript:   &transcript,
		})
		assert.NoError(t, err)
		return transcript.String()
	}

	t.Run("prints output", func(t *testing.T) {
		t.Parallel()
		out := run(t, "")
		assert.Contains(t, out, "exec-output-line")
	})

	t.Run("passthru false", func(t *testing.T) {
		t.Parallel()
		out := run(t, "        passthru: false\n")
		assert.NotContains(t, out, "exec-output-line")
	})
}

func TestIsEchoCommand(t *testing.T) {
	tests := []struct {
		name     string
		cmd      string
		expected bool
	}{
		{
			name:     "simple echo command",
			cmd:      "echo hello",
			expected: true,
		},
		{
			name:     "echo with quoted string",
			cmd:      "echo 'hello world'",
			expected: true,
		},
		{
			name:     "echo with variable",
			cmd:      "echo ${{ name }}",
			expected: true,
		},
		{
			name:     "echo with leading spaces",
			cmd:      "   echo test",
			expected: true,
		},
		{
			name:     "echo with multiline",
			cmd:      "echo hello\necho world",
			expected: false,
		},
		{
			name:     "non-echo command",
			cmd:      "make build",
			expected: false,
		},
		{
			name:     "command with echo in middle",
			cmd:      "cmd | echo test",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsEchoCommand(tt.cmd)
			assert.Equal(t, tt.expected, result)
		})
	}
}

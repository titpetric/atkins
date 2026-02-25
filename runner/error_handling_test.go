package runner_test

import (
	"embed"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/titpetric/atkins/runner"
)

//go:embed all:testdata
var testdataFS embed.FS

func TestErrorHandling(t *testing.T) {
	tests := []struct {
		name     string
		fixture  string
		checkErr func(t *testing.T, err error)
	}{
		// ── Step execution errors ────────────────────────────────────────
		// Steps that echo output and exit 1 produce runner.ExecError.
		{
			name:    "step echoes to stdout and exits 1",
			fixture: "testdata/error-handling/step-stdout-exit1.yml",
			checkErr: func(t *testing.T, err error) {
				var execErr runner.ExecError
				require.True(t, errors.As(err, &execErr), "expected ExecError, got %T: %v", err, err)
				assert.Equal(t, 1, execErr.LastExitCode)
				assert.Contains(t, execErr.Output, "stdout error output")
			},
		},
		{
			name:    "step echoes to stderr and exits 1",
			fixture: "testdata/error-handling/step-stderr-exit1.yml",
			checkErr: func(t *testing.T, err error) {
				var execErr runner.ExecError
				require.True(t, errors.As(err, &execErr), "expected ExecError, got %T: %v", err, err)
				assert.Equal(t, 1, execErr.LastExitCode)
				assert.Contains(t, execErr.Output, "stderr error output")
			},
		},
		{
			name:    "step echoes to stdout and stderr then exits 1",
			fixture: "testdata/error-handling/step-mixed-exit1.yml",
			checkErr: func(t *testing.T, err error) {
				var execErr runner.ExecError
				require.True(t, errors.As(err, &execErr), "expected ExecError, got %T: %v", err, err)
				assert.Equal(t, 1, execErr.LastExitCode)
				// ExecError.Output prefers stderr over stdout when stderr is non-empty
				assert.Contains(t, execErr.Output, "stderr line")
			},
		},

		// ── cmd: and cmds: step formats ─────────────────────────────────
		{
			name:    "step with cmd field exits 1",
			fixture: "testdata/error-handling/step-cmd-exit1.yml",
			checkErr: func(t *testing.T, err error) {
				var execErr runner.ExecError
				require.True(t, errors.As(err, &execErr), "expected ExecError, got %T: %v", err, err)
				assert.Equal(t, 1, execErr.LastExitCode)
				assert.Contains(t, execErr.Output, "cmd error output")
			},
		},
		{
			name:    "step with cmds where second command fails",
			fixture: "testdata/error-handling/step-cmds-exit1.yml",
			checkErr: func(t *testing.T, err error) {
				// executeCommands continues on errors and returns the last error
				var execErr runner.ExecError
				require.True(t, errors.As(err, &execErr), "expected ExecError, got %T: %v", err, err)
				assert.Equal(t, 1, execErr.LastExitCode)
			},
		},

		// ── Multiline run: | script ─────────────────────────────────────
		{
			name:    "multiline script echoes then exits 1",
			fixture: "testdata/error-handling/step-multiline-exit1.yml",
			checkErr: func(t *testing.T, err error) {
				var execErr runner.ExecError
				require.True(t, errors.As(err, &execErr), "expected ExecError, got %T: %v", err, err)
				assert.Equal(t, 1, execErr.LastExitCode)
				assert.Contains(t, execErr.Output, "second line stderr")
			},
		},

		// ── Multi-step: first succeeds, second fails ────────────────────
		// Verifies stop-on-failure semantics: step three should not run.
		{
			name:    "second step fails after first succeeds",
			fixture: "testdata/error-handling/step-second-fails.yml",
			checkErr: func(t *testing.T, err error) {
				var execErr runner.ExecError
				require.True(t, errors.As(err, &execErr), "expected ExecError, got %T: %v", err, err)
				assert.Equal(t, 1, execErr.LastExitCode)
				assert.Contains(t, execErr.Output, "step two fails")
				// step three's output should NOT appear in the error
				assert.NotContains(t, execErr.Output, "step three")
			},
		},

		// ── Interpolation errors in vars ─────────────────────────────────
		// $(echo ...; exit 1) during variable resolution produces regular
		// errors (not ExecError) propagated from MergeVariables.
		{
			name:    "interpolation failure in pipeline-level var",
			fixture: "testdata/error-handling/interp-pipeline-var.yml",
			checkErr: func(t *testing.T, err error) {
				msg := err.Error()
				assert.Contains(t, msg, "command execution failed")
				assert.Contains(t, msg, "exit")
			},
		},
		{
			name:    "interpolation failure in job-level var",
			fixture: "testdata/error-handling/interp-job-var.yml",
			checkErr: func(t *testing.T, err error) {
				msg := err.Error()
				assert.Contains(t, msg, "command execution failed")
				assert.Contains(t, msg, "exit")
			},
		},
		{
			name:    "interpolation failure in step-level var",
			fixture: "testdata/error-handling/interp-step-var.yml",
			checkErr: func(t *testing.T, err error) {
				msg := err.Error()
				assert.Contains(t, msg, "command execution failed")
				assert.Contains(t, msg, "exit")
			},
		},
		{
			name:    "interpolation failure in run command",
			fixture: "testdata/error-handling/interp-in-run.yml",
			checkErr: func(t *testing.T, err error) {
				msg := err.Error()
				assert.Contains(t, msg, "command execution failed")
				assert.Contains(t, msg, "exit")
			},
		},

		// ── Interpolation errors in env.vars ─────────────────────────────
		// env.vars go through mergeEnv → processEnv → interpolateVariables,
		// a distinct codepath from vars.
		{
			name:    "interpolation failure in pipeline-level env var",
			fixture: "testdata/error-handling/interp-pipeline-env.yml",
			checkErr: func(t *testing.T, err error) {
				msg := err.Error()
				assert.Contains(t, msg, "command execution failed")
				assert.Contains(t, msg, "error processing environment")
			},
		},
		{
			name:    "interpolation failure in job-level env var",
			fixture: "testdata/error-handling/interp-job-env.yml",
			checkErr: func(t *testing.T, err error) {
				msg := err.Error()
				assert.Contains(t, msg, "command execution failed")
			},
		},
		{
			name:    "interpolation failure in step-level env var",
			fixture: "testdata/error-handling/interp-step-env.yml",
			checkErr: func(t *testing.T, err error) {
				msg := err.Error()
				assert.Contains(t, msg, "command execution failed")
			},
		},

		// ── Dir errors: interpolation and nonexistent ────────────────────
		{
			name:    "interpolation failure in pipeline dir",
			fixture: "testdata/error-handling/interp-pipeline-dir.yml",
			checkErr: func(t *testing.T, err error) {
				msg := err.Error()
				assert.Contains(t, msg, "failed to interpolate pipeline dir")
				assert.Contains(t, msg, "command execution failed")
			},
		},
		{
			name:    "interpolation failure in job dir",
			fixture: "testdata/error-handling/interp-job-dir.yml",
			checkErr: func(t *testing.T, err error) {
				msg := err.Error()
				assert.Contains(t, msg, "failed to interpolate job dir")
				assert.Contains(t, msg, "command execution failed")
			},
		},
		{
			name:    "interpolation failure in step dir",
			fixture: "testdata/error-handling/interp-step-dir.yml",
			checkErr: func(t *testing.T, err error) {
				msg := err.Error()
				assert.Contains(t, msg, "failed to interpolate step dir")
				assert.Contains(t, msg, "command execution failed")
			},
		},
		{
			name:    "nonexistent pipeline dir",
			fixture: "testdata/error-handling/pipeline-dir-nonexistent.yml",
			checkErr: func(t *testing.T, err error) {
				msg := err.Error()
				assert.Contains(t, msg, "pipeline dir")
				assert.Contains(t, msg, "no such file or directory")
			},
		},
		{
			name:    "nonexistent job dir",
			fixture: "testdata/error-handling/job-dir-nonexistent.yml",
			checkErr: func(t *testing.T, err error) {
				msg := err.Error()
				assert.Contains(t, msg, "job dir")
				assert.Contains(t, msg, "no such file or directory")
			},
		},
		{
			name:    "nonexistent step dir",
			fixture: "testdata/error-handling/step-dir-nonexistent.yml",
			checkErr: func(t *testing.T, err error) {
				msg := err.Error()
				assert.Contains(t, msg, "step dir")
				assert.Contains(t, msg, "no such file or directory")
			},
		},

		// ── Interpolation error in for loop source ───────────────────────
		// $(exit 1) in the for iteration source expression.
		{
			name:    "interpolation failure in for loop source",
			fixture: "testdata/error-handling/interp-for-source.yml",
			checkErr: func(t *testing.T, err error) {
				msg := err.Error()
				assert.Contains(t, msg, "failed to expand for loop")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := testdataFS.ReadFile(tt.fixture)
			require.NoError(t, err)

			pipelines, err := runner.LoadPipelineFromReader(strings.NewReader(string(data)))
			require.NoError(t, err)
			require.NotEmpty(t, pipelines)

			err = runner.RunPipeline(t.Context(), pipelines[0], runner.PipelineOptions{
				Job:          "default",
				JSON:         true,
				AllPipelines: pipelines,
			})
			require.Error(t, err, "expected pipeline to fail for %s", tt.fixture)
			tt.checkErr(t, err)
		})
	}
}

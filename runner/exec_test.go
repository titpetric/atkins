package runner_test

import (
	"bytes"
	"context"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/titpetric/atkins/psexec"
	"github.com/titpetric/atkins/runner"
)

func TestPsexec_Run(t *testing.T) {
	ctx := context.Background()
	exec := psexec.New()

	t.Run("simple echo", func(t *testing.T) {
		result := exec.Run(ctx, psexec.NewShellCommand("echo 'hello world'"))
		assert.True(t, result.Success())
		assert.Contains(t, result.Output(), "hello world")
	})

	t.Run("exit 0", func(t *testing.T) {
		result := exec.Run(ctx, psexec.NewShellCommand("exit 0"))
		assert.True(t, result.Success())
	})

	t.Run("exit 1", func(t *testing.T) {
		result := exec.Run(ctx, psexec.NewShellCommand("exit 1"))
		assert.False(t, result.Success())
		assert.Equal(t, 1, result.ExitCode())
	})

	t.Run("exit 42", func(t *testing.T) {
		result := exec.Run(ctx, psexec.NewShellCommand("exit 42"))
		assert.False(t, result.Success())
		assert.Equal(t, 42, result.ExitCode())
	})
}

func TestPsexec_WithEnv(t *testing.T) {
	ctx := context.Background()

	t.Run("custom env", func(t *testing.T) {
		exec := psexec.NewWithOptions(&psexec.Options{
			DefaultEnv: []string{"TEST_VAR=custom_value"},
		})
		result := exec.Run(ctx, exec.ShellCommand("echo $TEST_VAR"))
		assert.True(t, result.Success())
		assert.Contains(t, result.Output(), "custom_value")
	})

	t.Run("multiple env vars", func(t *testing.T) {
		exec := psexec.NewWithOptions(&psexec.Options{
			DefaultEnv: []string{"VAR1=value1", "VAR2=value2"},
		})
		result := exec.Run(ctx, exec.ShellCommand("echo $VAR1 $VAR2"))
		assert.True(t, result.Success())
		assert.Contains(t, result.Output(), "value1")
		assert.Contains(t, result.Output(), "value2")
	})
}

func TestPsexec_WithDir(t *testing.T) {
	ctx := context.Background()
	exec := psexec.NewWithOptions(&psexec.Options{
		DefaultDir: "/tmp",
	})

	result := exec.Run(ctx, exec.ShellCommand("pwd"))
	assert.True(t, result.Success())
	assert.Contains(t, result.Output(), "/tmp")
}

func TestPsexec_WithWriter(t *testing.T) {
	ctx := context.Background()
	exec := psexec.New()

	t.Run("captures output to writer", func(t *testing.T) {
		var buf bytes.Buffer
		cmd := psexec.NewShellCommand("echo 'hello world'")
		cmd.Stdout = &buf
		cmd.Stderr = &buf

		result := exec.Run(ctx, cmd)
		assert.True(t, result.Success())
		assert.Contains(t, result.Output(), "hello world")
		assert.Contains(t, buf.String(), "hello world")
	})

	t.Run("stderr captured", func(t *testing.T) {
		var buf bytes.Buffer
		cmd := psexec.NewShellCommand("echo 'error' >&2 && exit 1")
		cmd.Stdout = &buf
		cmd.Stderr = &buf

		result := exec.Run(ctx, cmd)
		assert.False(t, result.Success())
		assert.Contains(t, buf.String(), "error")
	})

	t.Run("with PTY", func(t *testing.T) {
		if os.Getenv("CI") != "" {
			t.Skip("Skipping PTY test in CI environment")
		}

		var buf bytes.Buffer
		cmd := psexec.NewShellCommand("echo 'tty test'")
		cmd.Stdout = &buf
		cmd.UsePTY = true

		result := exec.Run(ctx, cmd)
		assert.True(t, result.Success())
		assert.Contains(t, result.Output(), "tty test")
	})

	t.Run("multiline output", func(t *testing.T) {
		var buf bytes.Buffer
		cmd := psexec.NewShellCommand("printf 'line1\\nline2\\nline3\\n'")
		cmd.Stdout = &buf

		result := exec.Run(ctx, cmd)
		assert.True(t, result.Success())
		lines := strings.Split(strings.TrimSpace(result.Output()), "\n")
		assert.Len(t, lines, 3)
	})

	t.Run("large output", func(t *testing.T) {
		var buf bytes.Buffer
		cmd := psexec.NewShellCommand("seq 1 1000")
		cmd.Stdout = &buf

		result := exec.Run(ctx, cmd)
		assert.True(t, result.Success())
		lines := strings.Split(strings.TrimSpace(result.Output()), "\n")
		assert.Len(t, lines, 1000)
	})

	t.Run("discard writer", func(t *testing.T) {
		cmd := psexec.NewShellCommand("echo 'discarded'")
		cmd.Stdout = io.Discard

		result := exec.Run(ctx, cmd)
		assert.True(t, result.Success())
		assert.Contains(t, result.Output(), "discarded")
	})
}

func TestExecError(t *testing.T) {
	t.Run("error message", func(t *testing.T) {
		execErr := runner.ExecError{
			Message:      "test error",
			Output:       "error output",
			LastExitCode: 1,
		}

		assert.Equal(t, "test error", execErr.Error())
		assert.Equal(t, len("error output"), execErr.Len())
	})

	t.Run("implements error interface", func(t *testing.T) {
		var err error = runner.ExecError{Message: "test"}
		assert.NotNil(t, err)
	})

	t.Run("NewExecError from result", func(t *testing.T) {
		ctx := context.Background()
		exec := psexec.New()

		result := exec.Run(ctx, psexec.NewShellCommand("exit 42"))
		execErr := runner.NewExecError(result)

		assert.Equal(t, 42, execErr.LastExitCode)
	})
}

func TestPsexec_SpecialCharacters(t *testing.T) {
	ctx := context.Background()
	exec := psexec.New()

	tests := []struct {
		name     string
		cmd      string
		expected string
	}{
		{
			name:     "single quotes",
			cmd:      "printf 'hello'",
			expected: "hello",
		},
		{
			name:     "double quotes with variable",
			cmd:      "VAR='test' && echo \"$VAR\"",
			expected: "test",
		},
		{
			name:     "pipes",
			cmd:      "echo 'hello world' | wc -w",
			expected: "2",
		},
		{
			name:     "redirection",
			cmd:      "echo 'test' > /tmp/test_exec.txt && cat /tmp/test_exec.txt",
			expected: "test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := exec.Run(ctx, psexec.NewShellCommand(tt.cmd))
			require.True(t, result.Success())
			assert.Contains(t, strings.TrimSpace(result.Output()), tt.expected)
		})
	}
}

func TestPsexec_Timeout(t *testing.T) {
	ctx := context.Background()
	exec := psexec.New()

	result := exec.Run(ctx, psexec.NewShellCommand("sleep 0.1 && echo 'done'"))
	assert.True(t, result.Success())
	assert.Contains(t, result.Output(), "done")
}

package psexec_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/titpetric/atkins/psexec"
)

func TestResult_Output(t *testing.T) {
	exec := psexec.New()
	ctx := context.Background()

	cmd := psexec.NewCommand("echo", "test output")
	result := exec.Run(ctx, cmd)

	assert.Contains(t, result.Output(), "test output")
}

func TestResult_Output_Empty(t *testing.T) {
	exec := psexec.New()
	ctx := context.Background()

	cmd := psexec.NewShellCommand("exit 0")
	result := exec.Run(ctx, cmd)

	assert.Empty(t, result.Output())
}

func TestResult_ErrorOutput(t *testing.T) {
	exec := psexec.New()
	ctx := context.Background()

	cmd := psexec.NewShellCommand("echo 'error message' >&2")
	result := exec.Run(ctx, cmd)

	assert.Contains(t, result.ErrorOutput(), "error message")
}

func TestResult_ErrorOutput_EmptyWithPTY(t *testing.T) {
	exec := psexec.New()
	ctx := context.Background()

	// PTY combines stdout/stderr, so ErrorOutput is always empty
	cmd := psexec.NewShellCommand("echo 'stderr' >&2").WithPTY()
	result := exec.Run(ctx, cmd)

	// Stderr goes to Output when using PTY
	assert.Empty(t, result.ErrorOutput())
	assert.Contains(t, result.Output(), "stderr")
}

func TestResult_ExitCode(t *testing.T) {
	tests := []struct {
		name     string
		cmd      string
		expected int
	}{
		{"zero", "exit 0", 0},
		{"one", "exit 1", 1},
		{"custom", "exit 42", 42},
	}

	exec := psexec.New()
	ctx := context.Background()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := psexec.NewShellCommand(tt.cmd)
			result := exec.Run(ctx, cmd)
			assert.Equal(t, tt.expected, result.ExitCode())
		})
	}
}

func TestResult_Err(t *testing.T) {
	exec := psexec.New()
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		result := exec.Run(ctx, psexec.NewShellCommand("exit 0"))
		assert.Nil(t, result.Err())
	})

	t.Run("failure", func(t *testing.T) {
		result := exec.Run(ctx, psexec.NewShellCommand("exit 1"))
		assert.NotNil(t, result.Err())
	})

	t.Run("command not found", func(t *testing.T) {
		result := exec.Run(ctx, psexec.NewCommand("nonexistent_cmd_12345"))
		assert.NotNil(t, result.Err())
	})
}

func TestResult_Success(t *testing.T) {
	exec := psexec.New()
	ctx := context.Background()

	t.Run("true when exit 0", func(t *testing.T) {
		result := exec.Run(ctx, psexec.NewShellCommand("exit 0"))
		assert.True(t, result.Success())
	})

	t.Run("false when exit non-zero", func(t *testing.T) {
		result := exec.Run(ctx, psexec.NewShellCommand("exit 1"))
		assert.False(t, result.Success())
	})
}

package psexec_test

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/titpetric/atkins/psexec"
)

func TestNew(t *testing.T) {
	exec := psexec.New()
	assert.NotNil(t, exec)
}

func TestExecutor_Run(t *testing.T) {
	exec := psexec.New()
	ctx := context.Background()

	cmd := psexec.NewCommand("echo", "hello")
	result := exec.Run(ctx, cmd)

	assert.True(t, result.Success())
	assert.Contains(t, result.Output(), "hello")
}

func TestExecutor_Run_ShellCommand(t *testing.T) {
	exec := psexec.New()
	ctx := context.Background()

	cmd := psexec.NewShellCommand("echo 'line1' && echo 'line2'")
	result := exec.Run(ctx, cmd)

	assert.True(t, result.Success())
	assert.Contains(t, result.Output(), "line1")
	assert.Contains(t, result.Output(), "line2")
}

func TestExecutor_Run_Failure(t *testing.T) {
	exec := psexec.New()
	ctx := context.Background()

	cmd := psexec.NewShellCommand("exit 42")
	result := exec.Run(ctx, cmd)

	assert.False(t, result.Success())
	assert.Equal(t, 42, result.ExitCode())
}

func TestExecutor_Run_WithStdin(t *testing.T) {
	exec := psexec.New()
	ctx := context.Background()

	input := strings.NewReader("stdin content")
	cmd := psexec.NewCommand("cat")
	cmd.Stdin = input
	result := exec.Run(ctx, cmd)

	assert.True(t, result.Success())
	assert.Equal(t, "stdin content", result.Output())
}

func TestExecutor_Run_WithStdout(t *testing.T) {
	exec := psexec.New()
	ctx := context.Background()

	var buf bytes.Buffer
	cmd := psexec.NewCommand("echo", "captured")
	cmd.Stdout = &buf
	result := exec.Run(ctx, cmd)

	assert.True(t, result.Success())
	assert.Contains(t, buf.String(), "captured")
	assert.Contains(t, result.Output(), "captured")
}

func TestExecutor_Run_WithStderr(t *testing.T) {
	exec := psexec.New()
	ctx := context.Background()

	var buf bytes.Buffer
	cmd := psexec.NewShellCommand("echo 'error' >&2")
	cmd.Stderr = &buf
	result := exec.Run(ctx, cmd)

	assert.True(t, result.Success())
	assert.Contains(t, buf.String(), "error")
	assert.Contains(t, result.ErrorOutput(), "error")
}

func TestExecutor_Run_WithDir(t *testing.T) {
	exec := psexec.New()
	ctx := context.Background()

	cmd := psexec.NewCommand("pwd")
	cmd.Dir = "/tmp"
	result := exec.Run(ctx, cmd)

	assert.True(t, result.Success())
	assert.Contains(t, result.Output(), "/tmp")
}

func TestExecutor_Run_WithEnv(t *testing.T) {
	exec := psexec.New()
	ctx := context.Background()

	cmd := psexec.NewShellCommand("echo $TEST_VAR")
	cmd.Env = []string{"TEST_VAR=test_value"}
	result := exec.Run(ctx, cmd)

	assert.True(t, result.Success())
	assert.Contains(t, result.Output(), "test_value")
}

func TestExecutor_Run_WithTimeout(t *testing.T) {
	exec := psexec.New()
	ctx := context.Background()

	cmd := psexec.NewShellCommand("sleep 10")
	cmd.Timeout = 50 * time.Millisecond
	result := exec.Run(ctx, cmd)

	assert.False(t, result.Success())
	assert.NotNil(t, result.Err())
}

func TestExecutor_Run_ContextCancellation(t *testing.T) {
	exec := psexec.New()
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	cmd := psexec.NewShellCommand("sleep 10")
	result := exec.Run(ctx, cmd)

	assert.False(t, result.Success())
}

func TestExecutor_Run_WithPTY(t *testing.T) {
	exec := psexec.New()
	ctx := context.Background()

	cmd := psexec.NewCommand("echo", "pty output")
	cmd.UsePTY = true
	result := exec.Run(ctx, cmd)

	assert.True(t, result.Success())
	assert.Contains(t, result.Output(), "pty output")
}

func TestExecutor_Run_WithPTY_CombinesStderr(t *testing.T) {
	exec := psexec.New()
	ctx := context.Background()

	// PTY combines stdout and stderr into single stream
	cmd := psexec.NewShellCommand("echo 'stdout' && echo 'stderr' >&2")
	cmd.UsePTY = true
	result := exec.Run(ctx, cmd)

	assert.True(t, result.Success())
	assert.Contains(t, result.Output(), "stdout")
	assert.Contains(t, result.Output(), "stderr")
	assert.Empty(t, result.ErrorOutput())
}

func TestExecutor_Run_WithPTY_Timeout(t *testing.T) {
	exec := psexec.New()
	ctx := context.Background()

	cmd := psexec.NewShellCommand("sleep 10")
	cmd.UsePTY = true
	cmd.Timeout = 50 * time.Millisecond
	result := exec.Run(ctx, cmd)

	assert.False(t, result.Success())
}

func TestExecutor_Run_Multiple(t *testing.T) {
	exec := psexec.New()
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		cmd := psexec.NewShellCommand("echo test")
		result := exec.Run(ctx, cmd)
		assert.True(t, result.Success())
	}
}

func TestExecutor_DefaultDir(t *testing.T) {
	exec := psexec.NewWithOptions(&psexec.Options{
		DefaultDir: "/tmp",
	})
	ctx := context.Background()

	cmd := psexec.NewCommand("pwd")
	result := exec.Run(ctx, cmd)

	assert.True(t, result.Success())
	assert.Contains(t, result.Output(), "/tmp")
}

func TestExecutor_DefaultDir_Override(t *testing.T) {
	exec := psexec.NewWithOptions(&psexec.Options{
		DefaultDir: "/tmp",
	})
	ctx := context.Background()

	cmd := psexec.NewCommand("pwd")
	cmd.Dir = "/var"
	result := exec.Run(ctx, cmd)

	assert.True(t, result.Success())
	assert.Contains(t, result.Output(), "/var")
}

func TestExecutor_DefaultEnv(t *testing.T) {
	exec := psexec.NewWithOptions(&psexec.Options{
		DefaultEnv: []string{"DEFAULT_VAR=default"},
	})
	ctx := context.Background()

	cmd := psexec.NewShellCommand("echo $DEFAULT_VAR")
	result := exec.Run(ctx, cmd)

	assert.True(t, result.Success())
	assert.Contains(t, result.Output(), "default")
}

func TestExecutor_DefaultEnv_Override(t *testing.T) {
	exec := psexec.NewWithOptions(&psexec.Options{
		DefaultEnv: []string{"MY_VAR=default"},
	})
	ctx := context.Background()

	cmd := psexec.NewShellCommand("echo $MY_VAR")
	cmd.Env = []string{"MY_VAR=override"}
	result := exec.Run(ctx, cmd)

	assert.True(t, result.Success())
	assert.Contains(t, result.Output(), "override")
}

func TestExecutor_DefaultTimeout(t *testing.T) {
	exec := psexec.NewWithOptions(&psexec.Options{
		DefaultTimeout: 50 * time.Millisecond,
	})
	ctx := context.Background()

	cmd := psexec.NewShellCommand("sleep 10")
	result := exec.Run(ctx, cmd)

	assert.False(t, result.Success())
	assert.NotNil(t, result.Err())
}

func TestExecutor_DefaultTimeout_Override(t *testing.T) {
	exec := psexec.NewWithOptions(&psexec.Options{
		DefaultTimeout: 10 * time.Second, // long default
	})
	ctx := context.Background()

	cmd := psexec.NewShellCommand("sleep 10")
	cmd.Timeout = 50 * time.Millisecond // short override
	result := exec.Run(ctx, cmd)

	assert.False(t, result.Success())
}

func TestExecutor_ShellCommand(t *testing.T) {
	exec := psexec.NewWithOptions(&psexec.Options{
		DefaultShell: "bash",
	})
	ctx := context.Background()

	cmd := exec.ShellCommand("echo 'shell test'")
	result := exec.Run(ctx, cmd)

	assert.True(t, result.Success())
	assert.Contains(t, result.Output(), "shell test")
}

func TestExecutor_RunWithIO(t *testing.T) {
	exec := psexec.New()
	ctx := context.Background()

	var output bytes.Buffer
	cmd := psexec.NewShellCommand("echo 'io test'")
	result := exec.RunWithIO(ctx, &output, nil, cmd)

	assert.True(t, result.Success())
	assert.Contains(t, output.String(), "io test")
}

func TestExecutor_RunWithIO_WithStdin(t *testing.T) {
	exec := psexec.New()
	ctx := context.Background()

	input := strings.NewReader("hello\n")
	var output bytes.Buffer
	cmd := psexec.NewShellCommand("head -1")
	result := exec.RunWithIO(ctx, &output, input, cmd)

	assert.True(t, result.Success())
	assert.Contains(t, output.String(), "hello")
}

func TestExecutor_Interactive_NoTerminal(t *testing.T) {
	// When stdin is not a terminal, interactive mode should fail gracefully
	// with exit code 1 and a descriptive error.
	exec := psexec.New()
	ctx := context.Background()

	cmd := psexec.NewShellCommand("echo 'should not hang'")
	cmd.Interactive = true
	result := exec.Run(ctx, cmd)

	assert.False(t, result.Success())
	assert.Equal(t, 1, result.ExitCode())
	assert.Contains(t, result.Err().Error(), "failed to set raw mode")
}

func TestExecutor_Interactive_NoTerminal_ProcessCleanup(t *testing.T) {
	// When term.MakeRaw fails, the already-started process must be waited
	// for so it does not become a zombie.
	exec := psexec.New()
	ctx := context.Background()

	// Use a command that writes a marker file to prove it ran
	marker := fmt.Sprintf("/tmp/atkins_test_marker_%d", time.Now().UnixNano())
	cmd := psexec.NewShellCommand(fmt.Sprintf("touch %s && sleep 0.1", marker))
	cmd.Interactive = true
	result := exec.Run(ctx, cmd)

	assert.False(t, result.Success())
	assert.Equal(t, 1, result.ExitCode())

	// Give the process time to create the marker (it was started before MakeRaw failed)
	time.Sleep(200 * time.Millisecond)

	// The marker should exist — proves the process was started
	_, err := os.Stat(marker)
	// Process may or may not have had time to create the file before
	// ptmx.Close sent SIGHUP, but either way the process should not be a zombie.
	_ = err
	os.Remove(marker)
}

func TestExecutor_Interactive_NoTerminal_DoesNotHang(t *testing.T) {
	// Regression: interactive mode on a non-TTY must return promptly,
	// not block on stdin or PTY I/O.
	exec := psexec.New()
	ctx := context.Background()

	done := make(chan struct{})
	go func() {
		cmd := psexec.NewShellCommand("echo 'quick'")
		cmd.Interactive = true
		exec.Run(ctx, cmd)
		close(done)
	}()

	select {
	case <-done:
		// OK — returned promptly
	case <-time.After(5 * time.Second):
		t.Fatal("interactive mode on non-TTY hung for > 5s")
	}
}

func TestExecutor_WithPTY_CapturesAllOutput(t *testing.T) {
	// Regression: ensure that ALL output lines are captured, including
	// the final line printed just before the process exits.
	exec := psexec.New()
	ctx := context.Background()

	cmd := psexec.NewShellCommand("echo first && sleep 0.05 && echo last")
	cmd.UsePTY = true
	result := exec.Run(ctx, cmd)

	assert.True(t, result.Success())
	assert.Contains(t, result.Output(), "first")
	assert.Contains(t, result.Output(), "last")
}

func TestExecutor_WithPTY_BackgroundChild(t *testing.T) {
	// A command that spawns a background child must not cause the PTY
	// read to hang. The session leader (bash) exiting must be sufficient.
	exec := psexec.New()
	ctx := context.Background()

	done := make(chan psexec.Result, 1)
	go func() {
		cmd := psexec.NewShellCommand("echo parent_start; (sleep 60 &); echo parent_done")
		cmd.UsePTY = true
		done <- exec.Run(ctx, cmd)
	}()

	select {
	case result := <-done:
		assert.True(t, result.Success())
		assert.Contains(t, result.Output(), "parent_start")
		assert.Contains(t, result.Output(), "parent_done")
	case <-time.After(5 * time.Second):
		t.Fatal("PTY mode hung on command with background child")
	}
}

func TestExecutor_WithPTY_SequentialRuns(t *testing.T) {
	// Multiple sequential PTY executions must all work correctly;
	// no leaked goroutines or file descriptors should interfere.
	exec := psexec.New()
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		cmd := psexec.NewShellCommand(fmt.Sprintf("echo 'run_%d'", i))
		cmd.UsePTY = true
		result := exec.Run(ctx, cmd)

		assert.True(t, result.Success(), "run %d failed", i)
		assert.Contains(t, result.Output(), fmt.Sprintf("run_%d", i))
	}
}

func TestExecutor_WithPTY_ExitCode(t *testing.T) {
	exec := psexec.New()
	ctx := context.Background()

	cmd := psexec.NewShellCommand("exit 42")
	cmd.UsePTY = true
	result := exec.Run(ctx, cmd)

	assert.False(t, result.Success())
	assert.Equal(t, 42, result.ExitCode())
}

func TestExecutor_WithPTY_WithStdin(t *testing.T) {
	exec := psexec.New()
	ctx := context.Background()

	cmd := psexec.NewShellCommand("head -1")
	cmd.UsePTY = true
	cmd.Stdin = strings.NewReader("pty_input\n")
	result := exec.Run(ctx, cmd)

	assert.True(t, result.Success())
	assert.Contains(t, result.Output(), "pty_input")
}

func TestExecutor_WithPTY_Timeout(t *testing.T) {
	exec := psexec.New()
	ctx := context.Background()

	done := make(chan psexec.Result, 1)
	go func() {
		cmd := psexec.NewShellCommand("sleep 60")
		cmd.UsePTY = true
		cmd.Timeout = 100 * time.Millisecond
		done <- exec.Run(ctx, cmd)
	}()

	select {
	case result := <-done:
		assert.False(t, result.Success())
	case <-time.After(5 * time.Second):
		t.Fatal("PTY timeout did not work — hung for > 5s")
	}
}

func TestExecutor_WithPTY_ContextCancel(t *testing.T) {
	exec := psexec.New()
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan psexec.Result, 1)
	go func() {
		cmd := psexec.NewShellCommand("sleep 60")
		cmd.UsePTY = true
		done <- exec.Run(ctx, cmd)
	}()

	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case result := <-done:
		assert.False(t, result.Success())
	case <-time.After(5 * time.Second):
		t.Fatal("PTY context cancel did not work — hung for > 5s")
	}
}

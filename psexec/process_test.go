package psexec_test

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/titpetric/atkins/psexec"
)

func TestExecutor_Start(t *testing.T) {
	exec := psexec.New()
	ctx := context.Background()

	cmd := psexec.NewCommand("echo", "started")
	proc, err := exec.Start(ctx, cmd)
	require.NoError(t, err)
	require.NotNil(t, proc)
	defer proc.Close()

	result := proc.Wait()
	assert.True(t, result.Success())
}

func TestProcess_PTY(t *testing.T) {
	exec := psexec.New()
	ctx := context.Background()

	cmd := psexec.NewShellCommand("echo test")
	proc, err := exec.Start(ctx, cmd)
	require.NoError(t, err)
	defer proc.Close()

	pty := proc.PTY()
	assert.NotNil(t, pty)

	proc.Wait()
}

func TestProcess_Read(t *testing.T) {
	exec := psexec.New()
	ctx := context.Background()

	cmd := psexec.NewCommand("echo", "read test")
	proc, err := exec.Start(ctx, cmd)
	require.NoError(t, err)
	defer proc.Close()

	buf := make([]byte, 1024)
	n, _ := proc.Read(buf)

	assert.Greater(t, n, 0)
	assert.Contains(t, string(buf[:n]), "read test")

	proc.Wait()
}

func TestProcess_Write(t *testing.T) {
	exec := psexec.New()
	ctx := context.Background()

	cmd := psexec.NewCommand("cat")
	proc, err := exec.Start(ctx, cmd)
	require.NoError(t, err)
	defer proc.Close()

	n, err := proc.Write([]byte("write test\n"))
	assert.NoError(t, err)
	assert.Greater(t, n, 0)

	buf := make([]byte, 1024)
	n, _ = proc.Read(buf)
	assert.Contains(t, string(buf[:n]), "write test")

	proc.Close()
	proc.Wait()
}

func TestProcess_Close(t *testing.T) {
	exec := psexec.New()
	ctx := context.Background()

	cmd := psexec.NewShellCommand("sleep 10")
	proc, err := exec.Start(ctx, cmd)
	require.NoError(t, err)

	err = proc.Close()
	assert.NoError(t, err)

	// Double close should be safe
	err = proc.Close()
	assert.NoError(t, err)

	proc.Wait()
}

func TestProcess_Wait(t *testing.T) {
	exec := psexec.New()
	ctx := context.Background()

	cmd := psexec.NewShellCommand("echo done")
	proc, err := exec.Start(ctx, cmd)
	require.NoError(t, err)
	defer proc.Close()

	result := proc.Wait()
	assert.True(t, result.Success())
}

func TestProcess_Wait_Failure(t *testing.T) {
	exec := psexec.New()
	ctx := context.Background()

	cmd := psexec.NewShellCommand("exit 5")
	proc, err := exec.Start(ctx, cmd)
	require.NoError(t, err)
	defer proc.Close()

	result := proc.Wait()
	assert.False(t, result.Success())
	assert.Equal(t, 5, result.ExitCode())
}

func TestProcess_Done(t *testing.T) {
	exec := psexec.New()
	ctx := context.Background()

	cmd := psexec.NewShellCommand("echo done")
	proc, err := exec.Start(ctx, cmd)
	require.NoError(t, err)
	defer proc.Close()

	done := proc.Done()
	assert.NotNil(t, done)

	select {
	case <-done:
		// Process completed
	case <-time.After(2 * time.Second):
		t.Fatal("process did not complete")
	}
}

func TestProcess_Resize(t *testing.T) {
	exec := psexec.New()
	ctx := context.Background()

	cmd := psexec.NewShellCommand("sleep 0.1")
	proc, err := exec.Start(ctx, cmd)
	require.NoError(t, err)
	defer proc.Close()

	err = proc.Resize(40, 120)
	assert.NoError(t, err)

	proc.Wait()
}

func TestProcess_Signal(t *testing.T) {
	exec := psexec.New()
	ctx := context.Background()

	cmd := psexec.NewShellCommand("sleep 10")
	proc, err := exec.Start(ctx, cmd)
	require.NoError(t, err)
	defer proc.Close()

	err = proc.Signal(syscall.SIGTERM)
	assert.NoError(t, err)

	result := proc.Wait()
	assert.False(t, result.Success())
}

func TestProcess_Signal_NotStarted(t *testing.T) {
	exec := psexec.New()
	ctx := context.Background()

	cmd := psexec.NewCommand("nonexistent_12345")
	proc, err := exec.Start(ctx, cmd)

	if err == nil {
		defer proc.Close()
		// If somehow started, signal should work
		_ = proc.Signal(os.Kill)
		proc.Wait()
	}
}

func TestProcess_PID(t *testing.T) {
	exec := psexec.New()
	ctx := context.Background()

	cmd := psexec.NewShellCommand("sleep 0.1")
	proc, err := exec.Start(ctx, cmd)
	require.NoError(t, err)
	defer proc.Close()

	pid := proc.PID()
	assert.Greater(t, pid, 0)

	proc.Wait()
}

func TestProcess_Pipe(t *testing.T) {
	exec := psexec.New()
	ctx := context.Background()

	cmd := psexec.NewShellCommand("echo 'piped'")
	proc, err := exec.Start(ctx, cmd)
	require.NoError(t, err)
	defer proc.Close()

	var output bytes.Buffer
	err = proc.Pipe(&output, nil)

	assert.NoError(t, err)
	assert.Contains(t, output.String(), "piped")
}

func TestProcess_Pipe_WithStdin(t *testing.T) {
	exec := psexec.New()
	ctx := context.Background()

	cmd := psexec.NewShellCommand("head -1")
	proc, err := exec.Start(ctx, cmd)
	require.NoError(t, err)
	defer proc.Close()

	input := bytes.NewReader([]byte("input line\n"))
	var output bytes.Buffer

	err = proc.Pipe(&output, input)
	assert.NoError(t, err)
	assert.Contains(t, output.String(), "input line")
}

func TestProcess_Pipe_Failure(t *testing.T) {
	executor := psexec.New()
	ctx := context.Background()

	cmd := psexec.NewShellCommand("exit 3")
	proc, err := executor.Start(ctx, cmd)
	require.NoError(t, err)
	defer proc.Close()

	err = proc.Pipe(nil, nil)
	assert.Error(t, err)
}

func TestProcess_ContextCancellation_KillsProcess(t *testing.T) {
	executor := psexec.New()
	ctx, cancel := context.WithCancel(context.Background())

	cmd := psexec.NewShellCommand("sleep 60")
	proc, err := executor.Start(ctx, cmd)
	require.NoError(t, err)
	defer proc.Close()

	pid := proc.PID()
	require.Greater(t, pid, 0)

	// Cancel context
	cancel()

	// Wait for process to finish
	result := proc.Wait()
	assert.False(t, result.Success())

	// Give OS time to clean up
	time.Sleep(50 * time.Millisecond)

	// Verify process is dead using kill -0
	checkCmd := exec.Command("kill", "-0", fmt.Sprintf("%d", pid))
	err = checkCmd.Run()
	assert.Error(t, err, "process should be killed after context cancel")
}

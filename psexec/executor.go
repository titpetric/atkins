package psexec

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/creack/pty"
	"golang.org/x/term"
)

// Executor manages process execution.
type Executor struct {
	// DefaultEnv is the default environment for all commands.
	DefaultEnv []string
	// DefaultDir is the default working directory for all commands.
	DefaultDir string
	// DefaultTimeout is the default timeout for commands when not specified.
	// Zero means no timeout.
	DefaultTimeout time.Duration
	// DefaultShell is the shell used for shell commands.
	// Defaults to "bash" if empty.
	DefaultShell string
}

// New creates a new Executor with default settings.
func New() *Executor {
	return &Executor{
		DefaultShell: "bash",
	}
}

// ShellCommand creates a new Command that runs via the executor's configured shell.
func (e *Executor) ShellCommand(script string) *Command {
	shell := e.DefaultShell
	if shell == "" {
		shell = "bash"
	}
	return &Command{
		Name: shell,
		Args: []string{"-c", script},
	}
}

// Run executes a command and returns the result.
func (e *Executor) Run(ctx context.Context, cmd *Command) Result {
	if cmd.Interactive {
		return e.runInteractive(ctx, cmd)
	}
	if cmd.UsePTY {
		return e.runWithPTY(ctx, cmd)
	}
	return e.runStandard(ctx, cmd)
}

// prepareCmd creates and configures an exec.Cmd from a Command.
func (e *Executor) prepareCmd(ctx context.Context, cmd *Command) *exec.Cmd {
	execCmd := exec.CommandContext(ctx, cmd.Name, cmd.Args...)

	if cmd.Dir != "" {
		execCmd.Dir = cmd.Dir
	} else if e.DefaultDir != "" {
		execCmd.Dir = e.DefaultDir
	}

	execCmd.Env = e.buildEnv(cmd.Env)
	return execCmd
}

// applyTimeout applies timeout to context if configured.
func (e *Executor) applyTimeout(ctx context.Context, cmd *Command) (context.Context, context.CancelFunc) {
	timeout := cmd.Timeout
	if timeout == 0 {
		timeout = e.DefaultTimeout
	}
	if timeout > 0 {
		return context.WithTimeout(ctx, timeout)
	}
	return ctx, func() {}
}

// startPTY starts a command with PTY and sets terminal size.
func (e *Executor) startPTY(execCmd *exec.Cmd) (*os.File, error) {
	ptmx, err := pty.Start(execCmd)
	if err != nil {
		return nil, fmt.Errorf("failed to start PTY: %w", err)
	}
	if size := e.terminalSize(); size != nil {
		_ = pty.Setsize(ptmx, size)
	}
	return ptmx, nil
}

// runStandard executes a command without PTY allocation.
func (e *Executor) runStandard(ctx context.Context, cmd *Command) Result {
	result := &processResult{stdout: new(bytes.Buffer), stderr: new(bytes.Buffer)}
	startTime := time.Now()
	defer func() { result.duration = time.Since(startTime) }()

	ctx, cancel := e.applyTimeout(ctx, cmd)
	defer cancel()

	execCmd := e.prepareCmd(ctx, cmd)

	if cmd.Stdin != nil {
		execCmd.Stdin = cmd.Stdin
	}
	if cmd.Stdout != nil {
		execCmd.Stdout = io.MultiWriter(cmd.Stdout, result.stdout)
	} else {
		execCmd.Stdout = result.stdout
	}
	if cmd.Stderr != nil {
		execCmd.Stderr = io.MultiWriter(cmd.Stderr, result.stderr)
	} else {
		execCmd.Stderr = result.stderr
	}

	if err := execCmd.Run(); err != nil {
		result.err = err
		result.exitCode = e.extractExitCode(execCmd, err)
	}

	return result
}

// runWithPTY executes a command with PTY allocation.
func (e *Executor) runWithPTY(ctx context.Context, cmd *Command) Result {
	result := &processResult{stdout: new(bytes.Buffer), stderr: new(bytes.Buffer)}
	startTime := time.Now()
	defer func() { result.duration = time.Since(startTime) }()

	ctx, cancel := e.applyTimeout(ctx, cmd)
	defer cancel()

	execCmd := e.prepareCmd(ctx, cmd)

	ptmx, err := e.startPTY(execCmd)
	if err != nil {
		result.err = err
		result.exitCode = 1
		return result
	}
	defer func() { _ = ptmx.Close() }()

	var wg sync.WaitGroup
	if cmd.Stdin != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := io.Copy(ptmx, cmd.Stdin); err != nil && !errors.Is(err, io.EOF) {
				log.Printf("psexec: stdin copy error: %v", err)
			}
		}()
	}

	var outputBuf bytes.Buffer
	writers := []io.Writer{&outputBuf}
	if cmd.Stdout != nil {
		writers = append(writers, cmd.Stdout)
	}

	if _, err := io.Copy(io.MultiWriter(writers...), ptmx); err != nil && !errors.Is(err, io.EOF) {
		log.Printf("psexec: stdout copy error: %v", err)
	}

	wg.Wait()

	if err := execCmd.Wait(); err != nil {
		result.err = err
		result.exitCode = e.extractExitCode(execCmd, err)
	}

	result.stdout = &outputBuf
	return result
}

// runInteractive executes a command in full interactive mode.
func (e *Executor) runInteractive(ctx context.Context, cmd *Command) Result {
	result := &processResult{stdout: new(bytes.Buffer), stderr: new(bytes.Buffer)}
	startTime := time.Now()
	defer func() { result.duration = time.Since(startTime) }()

	execCmd := e.prepareCmd(ctx, cmd)

	ptmx, err := e.startPTY(execCmd)
	if err != nil {
		result.err = err
		result.exitCode = 1
		return result
	}
	defer func() { _ = ptmx.Close() }()

	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		result.err = fmt.Errorf("failed to set raw mode: %w", err)
		result.exitCode = 1
		return result
	}
	defer func() { _ = term.Restore(int(os.Stdin.Fd()), oldState) }()

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		if _, err := io.Copy(ptmx, os.Stdin); err != nil && !errors.Is(err, io.EOF) {
			log.Printf("psexec: stdin copy error: %v", err)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		if _, err := io.Copy(os.Stdout, ptmx); err != nil && !errors.Is(err, io.EOF) {
			log.Printf("psexec: stdout copy error: %v", err)
		}
	}()

	if err := execCmd.Wait(); err != nil {
		result.err = err
		result.exitCode = e.extractExitCode(execCmd, err)
	}

	return result
}

// RunWithIO executes a command with custom I/O streams, suitable for websocket transport.
func (e *Executor) RunWithIO(ctx context.Context, stdout io.Writer, stdin io.Reader, cmd *Command) Result {
	result := &processResult{stdout: new(bytes.Buffer), stderr: new(bytes.Buffer)}
	startTime := time.Now()
	defer func() { result.duration = time.Since(startTime) }()

	execCmd := e.prepareCmd(ctx, cmd)

	ptmx, err := e.startPTY(execCmd)
	if err != nil {
		result.err = err
		result.exitCode = 1
		return result
	}

	var wg sync.WaitGroup

	if stdin != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := io.Copy(ptmx, stdin); err != nil && !errors.Is(err, io.EOF) {
				log.Printf("psexec: stdin copy error: %v", err)
			}
		}()
	}

	if stdout != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := io.Copy(stdout, ptmx); err != nil && !errors.Is(err, io.EOF) {
				log.Printf("psexec: stdout copy error: %v", err)
			}
		}()
	}

	err = execCmd.Wait()
	wg.Wait()
	_ = ptmx.Close()

	if err != nil {
		result.err = err
		result.exitCode = e.extractExitCode(execCmd, err)
	}

	return result
}

// buildEnv constructs the environment for a command.
func (e *Executor) buildEnv(cmdEnv []string) []string {
	env := os.Environ()

	// Helper to set/replace env var
	set := func(kv string) {
		idx := strings.Index(kv, "=")
		if idx == -1 {
			return
		}
		key := kv[:idx]
		prefix := key + "="

		// Remove existing
		for i := 0; i < len(env); i++ {
			if strings.HasPrefix(env[i], prefix) {
				env = append(env[:i], env[i+1:]...)
				i--
			}
		}
		env = append(env, kv)
	}

	for _, kv := range e.DefaultEnv {
		set(kv)
	}
	for _, kv := range cmdEnv {
		set(kv)
	}

	return env
}

// extractExitCode extracts the exit code from a completed command.
func (e *Executor) extractExitCode(cmd *exec.Cmd, err error) int {
	if cmd.ProcessState != nil {
		if status, ok := cmd.ProcessState.Sys().(syscall.WaitStatus); ok {
			return status.ExitStatus()
		}
	}
	if err != nil {
		return 1
	}
	return 0
}

// terminalSize returns the current terminal size.
func (e *Executor) terminalSize() *pty.Winsize {
	if width, height, err := term.GetSize(int(os.Stdout.Fd())); err == nil {
		return &pty.Winsize{
			Rows: uint16(height),
			Cols: uint16(width),
		}
	}

	cols := os.Getenv("COLUMNS")
	lines := os.Getenv("LINES")

	width := 80
	height := 24

	if cols != "" {
		_, _ = fmt.Sscanf(cols, "%d", &width)
	}
	if lines != "" {
		_, _ = fmt.Sscanf(lines, "%d", &height)
	}

	return &pty.Winsize{
		Rows: uint16(height),
		Cols: uint16(width),
	}
}

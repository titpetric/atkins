package psexec

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
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
}

// New creates a new Executor with default settings.
func New() *Executor {
	return &Executor{}
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

// runStandard executes a command without PTY allocation.
func (e *Executor) runStandard(ctx context.Context, cmd *Command) Result {
	result := newResult()
	startTime := time.Now()
	defer func() { result.duration = time.Since(startTime) }()

	// Apply timeout if specified
	if cmd.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, cmd.Timeout)
		defer cancel()
	}

	// Build the exec.Cmd
	execCmd := exec.CommandContext(ctx, cmd.Name, cmd.Args...)

	// Set working directory
	if cmd.Dir != "" {
		execCmd.Dir = cmd.Dir
	} else if e.DefaultDir != "" {
		execCmd.Dir = e.DefaultDir
	}

	// Set environment
	execCmd.Env = e.buildEnv(cmd.Env)

	// Set stdin if provided
	if cmd.Stdin != nil {
		execCmd.Stdin = cmd.Stdin
	}

	// Set stdout - use provided writer or capture
	if cmd.Stdout != nil {
		execCmd.Stdout = io.MultiWriter(cmd.Stdout, result.stdout)
	} else {
		execCmd.Stdout = result.stdout
	}

	// Set stderr - use provided writer or capture
	if cmd.Stderr != nil {
		execCmd.Stderr = io.MultiWriter(cmd.Stderr, result.stderr)
	} else {
		execCmd.Stderr = result.stderr
	}

	// Run the command
	err := execCmd.Run()
	if err != nil {
		result.err = err
		result.exitCode = e.getExitCode(execCmd, err)
	}

	return result
}

// runWithPTY executes a command with PTY allocation.
func (e *Executor) runWithPTY(ctx context.Context, cmd *Command) Result {
	result := newResult()
	startTime := time.Now()
	defer func() { result.duration = time.Since(startTime) }()

	// Apply timeout if specified
	if cmd.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, cmd.Timeout)
		defer cancel()
	}

	// Build the exec.Cmd
	execCmd := exec.CommandContext(ctx, cmd.Name, cmd.Args...)

	// Set working directory
	if cmd.Dir != "" {
		execCmd.Dir = cmd.Dir
	} else if e.DefaultDir != "" {
		execCmd.Dir = e.DefaultDir
	}

	// Set environment
	execCmd.Env = e.buildEnv(cmd.Env)

	// Start with PTY
	ptmx, err := pty.Start(execCmd)
	if err != nil {
		result.err = fmt.Errorf("failed to start PTY: %w", err)
		result.exitCode = 1
		return result
	}
	defer ptmx.Close()

	// Set terminal size
	if size := getTerminalSize(); size != nil {
		pty.Setsize(ptmx, size)
	}

	// Handle stdin if provided
	var wg sync.WaitGroup
	if cmd.Stdin != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			io.Copy(ptmx, cmd.Stdin)
		}()
	}

	// Read output
	var outputBuf bytes.Buffer
	var writers []io.Writer
	writers = append(writers, &outputBuf)
	if cmd.Stdout != nil {
		writers = append(writers, cmd.Stdout)
	}
	multiWriter := io.MultiWriter(writers...)

	// Copy PTY output
	io.Copy(multiWriter, ptmx)

	// Wait for stdin copy if running
	wg.Wait()

	// Wait for command to complete
	err = execCmd.Wait()
	if err != nil {
		result.err = err
		result.exitCode = e.getExitCode(execCmd, err)
	}

	// Store output (PTY combines stdout/stderr)
	result.stdout = &outputBuf

	return result
}

// runInteractive executes a command in full interactive mode.
func (e *Executor) runInteractive(ctx context.Context, cmd *Command) Result {
	result := newResult()
	startTime := time.Now()
	defer func() { result.duration = time.Since(startTime) }()

	// Build the exec.Cmd
	execCmd := exec.CommandContext(ctx, cmd.Name, cmd.Args...)

	// Set working directory
	if cmd.Dir != "" {
		execCmd.Dir = cmd.Dir
	} else if e.DefaultDir != "" {
		execCmd.Dir = e.DefaultDir
	}

	// Set environment
	execCmd.Env = e.buildEnv(cmd.Env)

	// Start with PTY
	ptmx, err := pty.Start(execCmd)
	if err != nil {
		result.err = fmt.Errorf("failed to start PTY: %w", err)
		result.exitCode = 1
		return result
	}
	defer ptmx.Close()

	// Set terminal size
	if size := getTerminalSize(); size != nil {
		pty.Setsize(ptmx, size)
	}

	// Put terminal in raw mode
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		result.err = fmt.Errorf("failed to set raw mode: %w", err)
		result.exitCode = 1
		return result
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)

	// Bidirectional copy
	var wg sync.WaitGroup

	// stdin -> PTY
	wg.Add(1)
	go func() {
		defer wg.Done()
		io.Copy(ptmx, os.Stdin)
	}()

	// PTY -> stdout
	wg.Add(1)
	go func() {
		defer wg.Done()
		io.Copy(os.Stdout, ptmx)
	}()

	// Wait for command to complete
	err = execCmd.Wait()
	if err != nil {
		result.err = err
		result.exitCode = e.getExitCode(execCmd, err)
	}

	return result
}

// RunWithIO executes a command with custom I/O streams, suitable for websocket transport.
func (e *Executor) RunWithIO(ctx context.Context, stdout io.Writer, stdin io.Reader, cmd *Command) Result {
	result := newResult()
	startTime := time.Now()
	defer func() { result.duration = time.Since(startTime) }()

	// Build the exec.Cmd
	execCmd := exec.CommandContext(ctx, cmd.Name, cmd.Args...)

	// Set working directory
	if cmd.Dir != "" {
		execCmd.Dir = cmd.Dir
	} else if e.DefaultDir != "" {
		execCmd.Dir = e.DefaultDir
	}

	// Set environment
	execCmd.Env = e.buildEnv(cmd.Env)

	// Start with PTY for interactive-like behavior
	ptmx, err := pty.Start(execCmd)
	if err != nil {
		result.err = fmt.Errorf("failed to start PTY: %w", err)
		result.exitCode = 1
		return result
	}

	// Set terminal size
	if size := getTerminalSize(); size != nil {
		pty.Setsize(ptmx, size)
	}

	// Bidirectional copy
	var wg sync.WaitGroup

	// stdin -> PTY
	if stdin != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			io.Copy(ptmx, stdin)
		}()
	}

	// PTY -> stdout
	if stdout != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			io.Copy(stdout, ptmx)
		}()
	}

	// Wait for command to complete
	err = execCmd.Wait()

	// Wait for I/O goroutines to finish
	wg.Wait()

	// Close PTY to signal EOF to readers
	ptmx.Close()

	if err != nil {
		result.err = err
		result.exitCode = e.getExitCode(execCmd, err)
	}

	return result
}

// buildEnv constructs the environment for a command.
func (e *Executor) buildEnv(cmdEnv []string) []string {
	// Start with current process environment
	env := os.Environ()

	// Add default environment
	for _, kv := range e.DefaultEnv {
		env = setEnv(env, kv)
	}

	// Add command-specific environment
	for _, kv := range cmdEnv {
		env = setEnv(env, kv)
	}

	return env
}

// setEnv sets an environment variable, replacing any existing value.
func setEnv(env []string, kv string) []string {
	key := ""
	for i, c := range kv {
		if c == '=' {
			key = kv[:i]
			break
		}
	}
	if key == "" {
		return env
	}

	// Remove existing key
	for i := 0; i < len(env); i++ {
		if len(env[i]) > len(key) && env[i][:len(key)+1] == key+"=" {
			env = append(env[:i], env[i+1:]...)
			i--
		}
	}

	return append(env, kv)
}

// getExitCode extracts the exit code from a completed command.
func (e *Executor) getExitCode(cmd *exec.Cmd, err error) int {
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

// getTerminalSize returns the current terminal size.
func getTerminalSize() *pty.Winsize {
	// Try to get actual terminal size
	if width, height, err := term.GetSize(int(os.Stdout.Fd())); err == nil {
		return &pty.Winsize{
			Rows: uint16(height),
			Cols: uint16(width),
		}
	}

	// Fall back to environment variables
	cols := os.Getenv("COLUMNS")
	lines := os.Getenv("LINES")

	width := 80
	height := 24

	if cols != "" {
		fmt.Sscanf(cols, "%d", &width)
	}
	if lines != "" {
		fmt.Sscanf(lines, "%d", &height)
	}

	return &pty.Winsize{
		Rows: uint16(height),
		Cols: uint16(width),
	}
}

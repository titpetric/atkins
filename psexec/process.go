package psexec

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"syscall"

	"github.com/creack/pty"
)

// Process represents a running process with PTY support.
type Process struct {
	cmd    *exec.Cmd
	ptmx   *os.File
	result *processResult

	mu     sync.Mutex
	done   chan struct{}
	closed bool
}

// Start begins execution of a command and returns a Process handle.
// The process can be used for bidirectional I/O, particularly useful
// for websocket transport.
func (e *Executor) Start(ctx context.Context, cmd *Command) (*Process, error) {
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
		return nil, fmt.Errorf("failed to start PTY: %w", err)
	}

	// Set terminal size
	if size := getTerminalSize(); size != nil {
		pty.Setsize(ptmx, size)
	}

	proc := &Process{
		cmd:    execCmd,
		ptmx:   ptmx,
		result: newResult(),
		done:   make(chan struct{}),
	}

	// Start a goroutine to wait for completion
	go proc.wait()

	return proc, nil
}

// wait waits for the process to complete and captures the result.
func (p *Process) wait() {
	err := p.cmd.Wait()

	p.mu.Lock()
	defer p.mu.Unlock()

	if err != nil {
		p.result.err = err
		if p.cmd.ProcessState != nil {
			if status, ok := p.cmd.ProcessState.Sys().(syscall.WaitStatus); ok {
				p.result.exitCode = status.ExitStatus()
			} else {
				p.result.exitCode = 1
			}
		} else {
			p.result.exitCode = 1
		}
	}

	close(p.done)
}

// PTY returns the PTY file handle for direct I/O.
// This is useful for websocket transport where you want to
// directly copy between the websocket and the PTY.
func (p *Process) PTY() *os.File {
	return p.ptmx
}

// Read reads from the process output (PTY).
func (p *Process) Read(b []byte) (int, error) {
	return p.ptmx.Read(b)
}

// Write writes to the process input (PTY).
func (p *Process) Write(b []byte) (int, error) {
	return p.ptmx.Write(b)
}

// Close closes the PTY and terminates the process if still running.
func (p *Process) Close() error {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return nil
	}
	p.closed = true
	p.mu.Unlock()

	// Close the PTY
	if p.ptmx != nil {
		p.ptmx.Close()
	}

	// Kill the process if still running
	if p.cmd.Process != nil {
		p.cmd.Process.Kill()
	}

	return nil
}

// Wait waits for the process to complete and returns the result.
func (p *Process) Wait() Result {
	<-p.done
	return p.result
}

// Done returns a channel that is closed when the process completes.
func (p *Process) Done() <-chan struct{} {
	return p.done
}

// Resize resizes the PTY window.
func (p *Process) Resize(rows, cols uint16) error {
	return pty.Setsize(p.ptmx, &pty.Winsize{
		Rows: rows,
		Cols: cols,
	})
}

// Signal sends a signal to the process.
func (p *Process) Signal(sig os.Signal) error {
	if p.cmd.Process == nil {
		return fmt.Errorf("process not started")
	}
	return p.cmd.Process.Signal(sig)
}

// PID returns the process ID.
func (p *Process) PID() int {
	if p.cmd.Process == nil {
		return 0
	}
	return p.cmd.Process.Pid
}

// Pipe sets up bidirectional I/O between the process and the provided
// reader/writer. This is the primary method for websocket integration.
func (p *Process) Pipe(stdout io.Writer, stdin io.Reader) error {
	var wg sync.WaitGroup

	// stdin -> PTY
	if stdin != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			io.Copy(p.ptmx, stdin)
		}()
	}

	// PTY -> stdout
	if stdout != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			io.Copy(stdout, p.ptmx)
		}()
	}

	// Wait for process to complete
	<-p.done

	// Wait for I/O to complete
	wg.Wait()

	return p.result.Err()
}

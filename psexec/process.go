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
	"sync"
	"syscall"
	"time"

	"github.com/creack/pty"
)

// Process represents a running process with PTY support.
type Process struct {
	cmd       *exec.Cmd
	ptmx      *os.File
	result    *processResult
	startTime time.Time

	mu     sync.Mutex
	done   chan struct{}
	closed bool
}

// Start begins execution of a command and returns a Process handle.
// The process can be used for bidirectional I/O, particularly useful
// for websocket transport.
func (e *Executor) Start(ctx context.Context, cmd *Command) (*Process, error) {
	execCmd := e.prepareCmd(ctx, cmd)

	ptmx, err := e.startPTY(execCmd)
	if err != nil {
		return nil, err
	}

	proc := &Process{
		cmd:       execCmd,
		ptmx:      ptmx,
		result:    &processResult{stdout: new(bytes.Buffer), stderr: new(bytes.Buffer)},
		startTime: time.Now(),
		done:      make(chan struct{}),
	}

	go proc.wait()

	return proc, nil
}

// wait waits for the process to complete and captures the result.
func (p *Process) wait() {
	err := p.cmd.Wait()

	p.mu.Lock()
	defer p.mu.Unlock()

	p.result.duration = time.Since(p.startTime)

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

	if p.ptmx != nil {
		_ = p.ptmx.Close()
	}

	if p.cmd.Process != nil {
		_ = p.cmd.Process.Kill()
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
	return pty.Setsize(p.ptmx, &pty.Winsize{Rows: rows, Cols: cols})
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

	if stdin != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := io.Copy(p.ptmx, stdin); err != nil && !errors.Is(err, io.EOF) {
				log.Printf("psexec: stdin copy error: %v", err)
			}
		}()
	}

	if stdout != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := io.Copy(stdout, p.ptmx); err != nil && !errors.Is(err, io.EOF) {
				log.Printf("psexec: stdout copy error: %v", err)
			}
		}()
	}

	<-p.done
	wg.Wait()

	return p.result.Err()
}

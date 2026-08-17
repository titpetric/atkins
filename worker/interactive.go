package worker

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os/exec"
	"time"

	"github.com/creack/pty"

	"github.com/titpetric/atkins/client"
)

// An ordinary job has no stdin. That is the right default for a build —
// a command that stops to ask a question on a machine nobody is watching
// is a command that holds the agent until its lease runs out — and it is
// the wrong one for a job that declared `interactive: true`, which said
// in its own pipeline that it reads a terminal.
//
// For those the agent allocates a pty and pumps the server's input queue
// into it. What is on the other end of that queue is a terminal in a
// browser, so the whole path from a keypress to the process is: post,
// queue, the agent's next collection, the pty.
//
// The pty is also why the output looks right. A command with a pipe for
// output knows it is not talking to a terminal and says so — no colour,
// no cursor movement, no progress that overwrites itself — and a
// terminal page rendering that is a terminal page showing a transcript.

// terminalSize is what the agent tells the command its terminal is.
//
// It is fixed rather than negotiated. The browser knows its own size and
// could send it, and then the agent would have to carry a resize channel
// alongside the input one for a value that only matters to programs that
// draw boxes. Eighty by twenty-four is what every terminal has defaulted
// to for forty years, and atkins' own tree is built to fit it.
var terminalSize = &pty.Winsize{Rows: 24, Cols: 100}

// interactiveFlush is how often output is shipped for a job somebody is
// typing at.
//
// The batching interval the rest of a build uses is two seconds, which
// is right for a page somebody reads and unusable for a terminal
// somebody types at: it would be two seconds before a keystroke echoed.
// This is the other end of that trade — more requests, for a terminal
// that behaves like one.
const interactiveFlush = 100 * time.Millisecond

// runInteractive runs a job's command on a pty, with the server's input
// queue wired to its stdin.
//
// It returns the same Result the piped path does, so everything around
// it — the timeout, the lease, the artefacts, the status report — is
// unchanged. What differs is entirely inside the command's own view of
// the world: it has a terminal.
func (w *Worker) runInteractive(ctx context.Context, job *jobContext, workspace *Workspace) Result {
	cmd := exec.CommandContext(ctx, w.opts.Shell, "-c", job.Job.Command)
	cmd.Dir = workspace.Dir
	cmd.Env = w.environment(job, workspace)

	// pty.Start puts the command in its own session, which is the
	// process group the cancel below signals. setProcessGroup is not
	// used here because it would fight with that.
	cmd.Cancel = func() error { return killGroup(cmd) }
	cmd.WaitDelay = killGrace

	terminal, err := pty.Start(cmd)
	if err != nil {
		return Result{ExitCode: 1, Error: "could not allocate a terminal for this job: " + err.Error()}
	}
	_ = pty.Setsize(terminal, terminalSize)

	output := newLogWriter(ctx, w, job.Job.ID)
	output.interval(interactiveFlush)

	// Reading the pty is what detects the command exiting: the last
	// close of the child side ends the read. Wait is called after, so
	// the exit status is not collected before the output is drained.
	drained := make(chan struct{})
	go func() {
		defer close(drained)
		_, _ = io.Copy(output, terminal)
	}()

	// The keystroke pump stops when the command does. It is not waited
	// for: a collection in flight is a request the server will answer in
	// its own time, and nothing downstream depends on it having
	// returned.
	typing, stopTyping := context.WithCancel(ctx)
	go w.pumpInput(typing, job, terminal)

	err = cmd.Wait()
	stopTyping()

	// Closing unblocks the copy above if the command left a child
	// holding the terminal open.
	_ = terminal.Close()
	<-drained
	output.Close()

	return w.result(ctx, job, err)
}

// pumpInput writes what the browser types into the command's terminal.
//
// A collection that fails is logged and retried after a pause rather
// than ending the pump: the job is running either way, and a keystroke
// lost to a server restart is a keystroke somebody types again. A 409
// means the server no longer leases this job to us, which the heartbeat
// is already acting on, so the pump just stops.
func (w *Worker) pumpInput(ctx context.Context, job *jobContext, terminal io.Writer) {
	for {
		if ctx.Err() != nil {
			return
		}

		typed, err := w.client.CollectInput(ctx, job.Job.ID, w.opts.AgentID)
		if err != nil {
			if ctx.Err() != nil || leaseLost(err) {
				return
			}

			log.Printf("[agent] collecting input for job %s failed: %v", job.Job.ID, err)
			if !sleep(ctx, w.opts.PollInterval) {
				return
			}
			continue
		}

		if len(typed) == 0 {
			continue
		}

		if _, err := terminal.Write(typed); err != nil {
			// The command has gone. Whoever is typing finds out from the
			// job settling.
			return
		}
	}
}

// result turns the outcome of a pty command into a Result.
//
// It is the same reading the piped path does, factored out so the two
// cannot disagree about what a timeout or a signal means.
func (w *Worker) result(ctx context.Context, job *jobContext, err error) Result {
	result := Result{}

	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		result.TimedOut = true
		result.ExitCode = 1
		result.Error = fmt.Sprintf("job exceeded the %s agent timeout", w.opts.JobTimeout)
		w.appendLog(ctx, job.Job.ID, client.StreamError, result.Error+"\n")
		return result
	}

	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			result.ExitCode = exitErr.ExitCode()
		} else {
			result.ExitCode = 1
			result.Error = err.Error()
			w.appendLog(ctx, job.Job.ID, client.StreamError, err.Error()+"\n")
		}
	}

	return result
}

// killGroup stops a command and everything it started.
//
// pty.Start puts the command in a session of its own, so the group to
// signal is the command's pid. It is the same reasoning as
// setProcessGroup: a job's real work is the shell's children, and
// signalling the shell alone leaves them holding the terminal.
func killGroup(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return nil
	}
	return killProcessGroup(cmd.Process.Pid)
}

// interval retunes how often the writer ships what it has collected.
//
// A terminal wants its output as it happens; a build wants it batched.
// The writer is the same either way, and this is the one knob between
// the two.
func (w *logWriter) interval(every time.Duration) {
	w.ticker.Reset(every)
}

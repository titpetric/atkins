package worker

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/titpetric/atkins/client"
)

// Result is the outcome of running a job command.
type Result struct {
	ExitCode int
	TimedOut bool
	Error    string
}

// killGrace is how long a killed command has to release the agent's
// pipes before they are closed from under it.
const killGrace = 5 * time.Second

// execute runs the job command in the prepared workspace.
func (w *Worker) execute(ctx context.Context, job *jobContext, workspace *Workspace) Result {
	cmd := exec.CommandContext(ctx, w.opts.Shell, "-c", job.Job.Command)
	cmd.Dir = workspace.Dir
	cmd.Env = w.environment(job, workspace)
	setProcessGroup(cmd)

	// A child that escapes the group, or one wedged in uninterruptible
	// I/O, must not hold the agent open indefinitely. After the kill,
	// give the pipes a moment and then move on.
	cmd.WaitDelay = killGrace

	output := newLogWriter(ctx, w, job.Job.ID)
	cmd.Stdout = output
	cmd.Stderr = output

	err := cmd.Run()
	output.Close()

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

// environment builds the process environment for a job command.
//
// ATKINS_NO_DISPATCH is the important one: without it the atkins the
// agent runs would see the agent's own credentials, hand the work
// straight back to the server, and nothing would ever execute. A
// pipeline that genuinely wants to queue child jobs clears it.
func (w *Worker) environment(job *jobContext, workspace *Workspace) []string {
	env := append(os.Environ(),
		client.EnvCI+"=true",
		client.EnvNoDispatch+"=1",
		client.EnvJobID+"="+job.Job.ID,
		client.EnvRootJobID+"="+job.Job.RootID,
		"ATKINS_AGENT_ID="+w.opts.AgentID,
		"ATKINS_WORKSPACE="+workspace.Root,
	)

	if job.Job.ParentID != "" {
		env = append(env, client.EnvParentJobID+"="+job.Job.ParentID)
	}
	if params := strings.TrimSpace(job.Job.Params); params != "" && params != "{}" {
		env = append(env, client.EnvJobParams+"="+params)
	}
	if job.Repository != nil {
		env = append(env,
			"ATKINS_REPOSITORY="+job.Repository.Slug,
			"ATKINS_REMOTE_URL="+job.Repository.RemoteURL,
		)
	}
	if job.Job.Revision != "" {
		env = append(env, "ATKINS_REVISION="+job.Job.Revision)
	}
	if job.Job.Branch != "" {
		env = append(env, "ATKINS_BRANCH="+job.Job.Branch)
	}

	return env
}

// logWriter batches command output and ships it to the server.
//
// Output is flushed on size or on a timer rather than per write, so a
// chatty build doesn't turn into one HTTP request per line, while a
// slow one still shows progress on the job page before it finishes.
type logWriter struct {
	ctx    context.Context
	worker *Worker
	jobID  string

	mu     sync.Mutex
	buffer strings.Builder

	ticker *time.Ticker
	done   chan struct{}
	once   sync.Once
}

// flushSize and flushInterval bound how far behind the job page runs.
const (
	flushSize     = 8 * 1024
	flushInterval = 2 * time.Second
)

func newLogWriter(ctx context.Context, worker *Worker, jobID string) *logWriter {
	w := &logWriter{
		ctx:    ctx,
		worker: worker,
		jobID:  jobID,
		ticker: time.NewTicker(flushInterval),
		done:   make(chan struct{}),
	}

	go func() {
		for {
			select {
			case <-w.done:
				return
			case <-w.ticker.C:
				w.flush()
			}
		}
	}()

	return w
}

// Write implements io.Writer.
func (w *logWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	w.buffer.Write(p)
	size := w.buffer.Len()
	w.mu.Unlock()

	if size >= flushSize {
		w.flush()
	}

	return len(p), nil
}

// flush ships whatever has accumulated.
func (w *logWriter) flush() {
	w.mu.Lock()
	content := w.buffer.String()
	w.buffer.Reset()
	w.mu.Unlock()

	if content == "" {
		return
	}

	w.worker.appendLog(w.ctx, w.jobID, client.StreamOutput, content)
}

// Close stops the timer and ships the remainder.
func (w *logWriter) Close() {
	w.once.Do(func() {
		w.ticker.Stop()
		close(w.done)
		w.flush()
	})
}

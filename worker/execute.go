package worker

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/titpetric/atkins/client"
	"github.com/titpetric/atkins/runner"
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
// The job's own variables are named here, the workspace contributes
// what it laid out through Workspace.Env, and the pair is applied on
// top of the filtered environment inherited returns. Workspace.Env is
// applied last, so a caller that puts a value there before the command
// starts has it reach the job.
//
// ATKINS_NO_DISPATCH is the important one: without it the atkins the
// agent runs would see the agent's own credentials, hand the work
// straight back to the server, and nothing would ever execute. A
// pipeline that genuinely wants to queue child jobs clears it.
func (w *Worker) environment(job *jobContext, workspace *Workspace) []string {
	env := runner.Env{
		client.EnvCI:         "true",
		client.EnvNoDispatch: "1",
		client.EnvJobID:      job.Job.ID,
		client.EnvRootJobID:  job.Job.RootID,
		"ATKINS_AGENT_ID":    w.opts.AgentID,
	}

	// The server a child job is dispatched to. A pipeline that clears
	// ATKINS_NO_DISPATCH still has to log in to reach the queue.
	if server := w.client.Server(); server != "" {
		env[client.EnvServer] = server
	}

	if job.Job.ParentID != "" {
		env[client.EnvParentJobID] = job.Job.ParentID
	}
	if params := strings.TrimSpace(job.Job.Params); params != "" && params != "{}" {
		env[client.EnvJobParams] = params
	}
	if job.Repository != nil {
		env["ATKINS_REPOSITORY"] = job.Repository.Slug
		env["ATKINS_REMOTE_URL"] = job.Repository.RemoteURL
	}

	maps.Copy(env, workspace.Env)

	// A name set twice takes its last value, so these override anything
	// of the same name that survived the filter.
	return append(inherited(), env.Environ()...)
}

// inherited returns the agent's process environment with the agent's
// own settings filtered out.
//
// ATKINS_* is the agent's namespace as much as the job's, and it holds
// the agent's credentials: ATKINS_AGENT_TOKEN admits its holder to the
// queue, to jobs dispatched for other repositories and to the deploy
// keys with their private halves, and an installation running the
// server beside the agent has ATKINS_SIGNING_KEY there too, which
// signs every token the server issues. PLATFORM_DB_* is filtered for
// the same reason, as a database URL carries its own password.
//
// The whole namespace is filtered and environment sets the job's
// variables by name, so a setting the agent gains later stays with the
// agent until someone exports it deliberately.
func inherited() []string {
	environ := os.Environ()
	env := make([]string, 0, len(environ))

	for _, entry := range environ {
		name, _, _ := strings.Cut(entry, "=")
		if strings.HasPrefix(name, "ATKINS_") || strings.HasPrefix(name, "PLATFORM_DB_") {
			continue
		}
		env = append(env, entry)
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

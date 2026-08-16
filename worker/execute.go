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
// The job's ATKINS_* variables are set here rather than inherited; see
// inherited for why the agent's own are dropped first.
//
// ATKINS_NO_DISPATCH is the important one: without it the atkins the
// agent runs would see the agent's own credentials, hand the work
// straight back to the server, and nothing would ever execute. A
// pipeline that genuinely wants to queue child jobs clears it.
func (w *Worker) environment(job *jobContext, workspace *Workspace) []string {
	env := append(inherited(),
		client.EnvCI+"=true",
		client.EnvNoDispatch+"=1",
		client.EnvJobID+"="+job.Job.ID,
		client.EnvRootJobID+"="+job.Job.RootID,
		"ATKINS_AGENT_ID="+w.opts.AgentID,
		"ATKINS_WORKSPACE="+workspace.Root,
		client.EnvArtefacts+"="+workspace.Artefacts,
	)

	// The server a child job would be dispatched to. The URL is not a
	// credential; reaching the queue still needs a login.
	if server := w.client.Server(); server != "" {
		env = append(env, client.EnvServer+"="+server)
	}

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
	// The checkout the agent produced, not the one the job asked for.
	// ATKINS_REVISION stays the name it has always had and now always
	// holds a resolved sha, so a pipeline reading it can pin an artefact
	// to a commit even when the job named a branch or a moving tag.
	if workspace.Checkout.Ref != "" {
		env = append(env, "ATKINS_REF="+workspace.Checkout.Ref)
	}
	if sha := workspace.Checkout.CommitSHA; sha != "" {
		env = append(env,
			"ATKINS_COMMIT_SHA="+sha,
			"ATKINS_REVISION="+sha,
		)
	}
	if workspace.Checkout.Branch != "" {
		env = append(env, "ATKINS_BRANCH="+workspace.Checkout.Branch)
	}

	return env
}

// inherited is the agent's process environment with the agent's own
// configuration removed.
//
// ATKINS_* is the agent's namespace before it is the job's: it carries
// ATKINS_AGENT_TOKEN, the fleet-wide enrolment secret, and on a
// single-host install the server's ATKINS_SIGNING_KEY as well. A job
// that can read the first can enrol from anywhere, claim work from the
// whole queue and fetch the deploy keys with their private halves —
// the escalation the repository allowlist exists to prevent; one that
// can read the second mints any token, admin included.
//
// So the whole namespace goes, and environment puts back the variables
// a job is documented to receive. That way a secret the agent gains
// later is private without anyone remembering to add it here.
// PLATFORM_DB_* goes with it: a database URL carries its password.
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

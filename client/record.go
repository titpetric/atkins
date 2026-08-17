package client

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
)

// RecordOptions describes the local run being logged.
type RecordOptions struct {
	// Directory is where atkins was invoked. Empty means the process
	// working directory.
	Directory string

	// Command is the atkins invocation. Empty means os.Args.
	Command string

	// Server overrides which logged-in server to record on.
	Server string
}

// Recorder is a run happening here that is logged on a CI/CD server.
//
// It is the other half of dispatch: the pipeline executes on this
// machine, as it does on a machine that never logged in, and the server
// ends up with the job an agent would have produced — the same command,
// checkout, output, exit code and duration. A history of what a team
// runs should not depend on where the tooling happens to be installed.
//
// A nil *Recorder is the unrecorded run, and every method tolerates it,
// so the caller has no branch to write.
type Recorder struct {
	client  *Client
	jobID   string
	url     string
	agentID string

	mu     sync.Mutex
	buffer strings.Builder

	// stop ends the lease renewals.
	stop func()
}

// recordHeartbeat is how often the lease is renewed. It is well inside
// the server's default lease of 15 minutes, and a local run that dies
// without reporting — a closed laptop, a killed terminal — is reclaimed
// as a timeout the same way an agent that disappears is.
const recordHeartbeat = 30 * time.Second

// logChunk bounds one append, so a long transcript arrives in pieces
// rather than as one request the server may refuse.
const logChunk = 32 * 1024

// Record opens a job for a run about to happen here.
//
// It returns nil when there is nothing to record against: no
// credential, no git repository, recording turned off, or a server that
// can't be reached. Recording is bookkeeping, and losing it must not
// cost the run — unlike dispatch, which is the run.
func Record(ctx context.Context, opts RecordOptions) *Recorder {
	if !Settings().Record || DispatchDisabled() {
		return nil
	}

	// A run an agent started is already a job, streamed by the agent
	// that owns it. Recording it again would file the same work twice.
	if os.Getenv(EnvJobID) != "" {
		return nil
	}

	c, err := Open(configuredServer(opts.Server))
	if err != nil {
		return nil
	}

	directory := opts.Directory
	if directory == "" {
		directory, _ = os.Getwd()
	}

	checkout, err := DetectCheckout(directory)
	if err != nil {
		return nil
	}

	command := opts.Command
	if command == "" {
		command = Command(os.Args)
	}

	agentID, _ := os.Hostname()
	if agentID == "" {
		agentID = "local"
	}

	callCtx, cancel := context.WithTimeout(ctx, configuredTimeout())
	defer cancel()

	response, err := c.Dispatch(callCtx, DispatchRequest{
		Repository:       checkout.Payload(),
		WorkingDirectory: checkout.WorkingDirectory,
		Command:          command,
		Labels:           Labels(),
		Agent:            agentID,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "atkins: recording this run on %s failed: %v\n", c.Server(), err)
		return nil
	}

	recorder := &Recorder{
		client:  c,
		jobID:   response.JobID,
		url:     c.JobURL(response.JobID, response.ViewToken),
		agentID: agentID,
	}

	// The commit is recorded the way an agent records what it checked
	// out, so a job page says which code the log came from.
	recorder.reportCheckout(ctx, checkout)
	recorder.Log(ctx, header(checkout, command, agentID))
	recorder.heartbeat(ctx)

	return recorder
}

// JobID is the server's ID for this run, empty when unrecorded.
func (r *Recorder) JobID() string {
	if r == nil {
		return ""
	}
	return r.jobID
}

// URL is where the run is watched in a browser, empty when unrecorded.
func (r *Recorder) URL() string {
	if r == nil {
		return ""
	}
	return r.url
}

// Write buffers output for the job log. It makes a Recorder an
// io.Writer, which is what lets the runner hand it a transcript without
// knowing what a job is.
func (r *Recorder) Write(p []byte) (int, error) {
	if r == nil {
		return len(p), nil
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.buffer.Write(p)

	return len(p), nil
}

// Log appends content to the job log immediately.
func (r *Recorder) Log(ctx context.Context, content string) {
	if r == nil || content == "" {
		return
	}

	for _, chunk := range chunks(content, logChunk) {
		callCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), configuredTimeout())
		err := r.client.AppendLog(callCtx, r.jobID, StreamOutput, chunk)
		cancel()

		if err != nil {
			fmt.Fprintf(os.Stderr, "atkins: appending to job %s failed: %v\n", r.jobID, err)
			return
		}
	}
}

// Finish flushes what was written and settles the job.
//
// The status is derived from the same exit code the shell sees, so a
// recorded run and the terminal it ran in never disagree.
func (r *Recorder) Finish(ctx context.Context, exitCode int, runErr error) {
	if r == nil {
		return
	}

	if r.stop != nil {
		r.stop()
	}

	r.mu.Lock()
	transcript := r.buffer.String()
	r.buffer.Reset()
	r.mu.Unlock()

	r.Log(ctx, transcript)

	// A pipeline that reported a failure without an exit code — the
	// shell sees zero, the run did not work — is recorded as failed.
	// The job page saying passed is the one reading nobody can correct.
	status := StatusPassed
	if exitCode != 0 || runErr != nil {
		status = StatusFailed
	}

	message := ""
	if runErr != nil {
		message = runErr.Error()
	}

	callCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), configuredTimeout())
	defer cancel()

	if err := r.client.ReportStatus(callCtx, r.jobID, JobStatusRequest{
		Status:   status,
		ExitCode: int64(exitCode),
		Error:    message,
	}); err != nil {
		fmt.Fprintf(os.Stderr, "atkins: settling job %s failed: %v\n", r.jobID, err)
	}
}

// Cancelled settles the job as cancelled, for a run the user stopped.
func (r *Recorder) Cancelled(ctx context.Context) {
	if r == nil {
		return
	}

	if r.stop != nil {
		r.stop()
	}

	r.mu.Lock()
	transcript := r.buffer.String()
	r.buffer.Reset()
	r.mu.Unlock()

	r.Log(ctx, transcript)

	callCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), configuredTimeout())
	defer cancel()

	if err := r.client.ReportStatus(callCtx, r.jobID, JobStatusRequest{
		Status:   StatusCancelled,
		ExitCode: 1,
		Error:    "cancelled",
	}); err != nil {
		fmt.Fprintf(os.Stderr, "atkins: settling job %s failed: %v\n", r.jobID, err)
	}
}

// reportCheckout records the commit the run is against.
func (r *Recorder) reportCheckout(ctx context.Context, checkout *Checkout) {
	if checkout.Revision == "" {
		return
	}

	ref := checkout.Branch
	if ref == "" {
		ref = checkout.Revision
	}

	callCtx, cancel := context.WithTimeout(ctx, configuredTimeout())
	defer cancel()

	// A checkout nobody could record is not worth a word: the log that
	// follows is the record, and it is already on its way.
	_ = r.client.ReportCheckout(callCtx, r.jobID, JobCheckoutRequest{
		Ref:       ref,
		CommitSHA: checkout.Revision,
	})
}

// heartbeat renews the lease until Finish stops it.
func (r *Recorder) heartbeat(ctx context.Context) {
	ctx, cancel := context.WithCancel(context.WithoutCancel(ctx))

	done := make(chan struct{})
	go func() {
		defer close(done)

		ticker := time.NewTicker(recordHeartbeat)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				callCtx, callCancel := context.WithTimeout(ctx, configuredTimeout())
				// A missed renewal is retried on the next tick. The run
				// is here and continues either way; what it risks is the
				// server reclaiming the record, which is what a run that
				// really has gone away looks like too.
				_ = r.client.Heartbeat(callCtx, r.jobID, r.agentID)
				callCancel()
			}
		}
	}()

	r.stop = func() {
		cancel()
		<-done
	}
}

// header is the first thing on a recorded job's page: where the run is
// happening, and against what.
//
// It matters because the log arrives at the end. Until then the page
// says a machine is running a command, which is the same thing a person
// watching an agent sees while it clones.
func header(checkout *Checkout, command, agentID string) string {
	var b strings.Builder

	fmt.Fprintf(&b, "running locally on %s\n", agentID)
	fmt.Fprintf(&b, "$ %s\n", command)

	if checkout.WorkingDirectory != "" {
		fmt.Fprintf(&b, "directory: %s\n", checkout.WorkingDirectory)
	}
	if checkout.Revision != "" {
		fmt.Fprintf(&b, "commit: %s", shortRevision(checkout.Revision))
		if checkout.Branch != "" && checkout.Branch != "HEAD" {
			fmt.Fprintf(&b, " (%s)", checkout.Branch)
		}
		// Said out loud, because the log below did not come from the
		// commit the job names.
		if checkout.Dirty {
			b.WriteString(" with uncommitted changes")
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	return b.String()
}

// chunks splits s into pieces of at most size bytes, on a line boundary
// where one is near enough that a log entry isn't cut mid-line.
func chunks(s string, size int) []string {
	var parts []string

	for len(s) > size {
		cut := size
		if index := strings.LastIndexByte(s[:size], '\n'); index > size/2 {
			cut = index + 1
		}

		parts = append(parts, s[:cut])
		s = s[cut:]
	}

	if s != "" {
		parts = append(parts, s)
	}

	return parts
}

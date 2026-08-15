// Package worker is the atkins CI/CD agent.
//
// An agent claims jobs from a server, reproduces the checkout they were
// dispatched from, runs the command, streams the output back and
// reports the outcome. It is the half that makes `atkins` on a laptop
// hand its work to a machine that has the tooling.
//
// The agent keeps a repository cache under <DataDir>/repos so repeated
// jobs for one repository fetch rather than clone, and gives every job
// its own work tree under <DataDir>/work.
package worker

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/titpetric/atkins/client"
)

// Worker claims and runs jobs for one server.
type Worker struct {
	opts   *Options
	client *client.Client

	policy policy
	ssh    sshKeys
}

// jobContext pairs a claimed job with the repository it belongs to.
type jobContext struct {
	Job        *client.Job
	Repository *client.Repository
}

// New returns a Worker, authenticating against the server.
//
// The agent logs in like any other user. When Register is set and the
// credentials don't match an account, it creates one: on a fresh
// instance there is nobody to create it for them.
func New(ctx context.Context, opts *Options) (*Worker, error) {
	if opts == nil {
		opts = NewOptions()
	}
	applyDefaults(opts)

	c, err := authenticate(ctx, opts)
	if err != nil {
		return nil, err
	}

	return &Worker{opts: opts, client: c}, nil
}

// applyDefaults fills the zero values the configuration cannot supply.
//
// Everything else is defaulted by the config package on load; only the
// agent identity depends on the machine rather than on the document.
func applyDefaults(opts *Options) {
	if opts.AgentID == "" {
		opts.AgentID, _ = os.Hostname()
	}
	if opts.AgentID == "" {
		opts.AgentID = "atkins-agent"
	}

	defaults := NewOptions()
	if opts.DataDir == "" {
		opts.DataDir = defaults.DataDir
	}
	if opts.PollInterval <= 0 {
		opts.PollInterval = defaults.PollInterval
	}
	if opts.JobTimeout <= 0 {
		opts.JobTimeout = defaults.JobTimeout
	}
	if opts.HeartbeatInterval <= 0 {
		opts.HeartbeatInterval = defaults.HeartbeatInterval
	}
	if opts.Shell == "" {
		opts.Shell = defaults.Shell
	}
}

// authenticate returns a logged-in client for the agent.
//
// The normal path is enrolment: one shared token in the agent's
// environment, traded for the same rotating credentials a human holds.
// The email/password path exists for an agent running under an ordinary
// account that an admin has flagged by hand.
func authenticate(ctx context.Context, opts *Options) (*client.Client, error) {
	if opts.Token != "" {
		c := client.New(opts.Server)
		if _, err := c.Enrol(ctx, client.EnrolRequest{
			Token:   opts.Token,
			AgentID: opts.AgentID,
			Labels:  opts.Labels,
		}); err != nil {
			return nil, fmt.Errorf("agent enrolment failed: %w", err)
		}

		log.Printf("[agent] enrolled as %s", opts.AgentID)
		return c, nil
	}

	if opts.Email != "" && opts.Password != "" {
		hostname, _ := os.Hostname()

		c := client.New(opts.Server)
		if _, err := c.Login(ctx, client.LoginRequest{
			Email:    opts.Email,
			Password: opts.Password,
			Hostname: hostname,
		}); err != nil {
			return nil, fmt.Errorf("agent login failed: %w", err)
		}
		return c, nil
	}

	// Nothing configured: fall back to whatever `atkins --login`
	// stored on this machine.
	c, err := client.Open(opts.Server)
	if err != nil {
		return nil, fmt.Errorf("agent has no credentials: set %s, or ATKINS_EMAIL and ATKINS_PASSWORD, or run atkins --login: %w",
			EnvToken, err)
	}
	return c, nil
}

// Run claims and executes jobs until the context is cancelled.
func (w *Worker) Run(ctx context.Context) error {
	log.Printf("[agent] %s polling %s every %s (labels: %v, data: %s)",
		w.opts.AgentID, w.client.Server(), w.opts.PollInterval, w.opts.Labels, w.opts.DataDir)

	w.logPolicy(ctx)
	w.installSSHKeys(ctx)

	for {
		if err := ctx.Err(); err != nil {
			return nil
		}

		claimed, err := w.client.Claim(ctx, w.opts.AgentID, w.opts.Labels)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return nil
			}
			// A server restart or a network blip should not end the
			// agent; back off and try again.
			log.Printf("[agent] claim failed: %v", err)
			if !sleep(ctx, w.opts.PollInterval) {
				return nil
			}
			continue
		}

		if claimed == nil {
			if !sleep(ctx, w.opts.PollInterval) {
				return nil
			}
			continue
		}

		w.run(ctx, &jobContext{Job: claimed.Job, Repository: claimed.Repository})
	}
}

// run executes one claimed job and reports its outcome.
func (w *Worker) run(ctx context.Context, job *jobContext) {
	log.Printf("[agent] job %s: %s", job.Job.ID, job.Job.Command)

	// The lease outlives cancellation of the claim call, and the
	// outcome still has to be reported if the agent is shutting down.
	ctx, cancel := context.WithTimeout(ctx, w.opts.JobTimeout)
	defer cancel()

	stop := w.heartbeat(ctx, job.Job.ID)
	defer stop()

	if job.Repository == nil {
		w.fail(ctx, job, "job has no repository")
		return
	}

	// Re-check the allowlist here, not only at dispatch: this job may
	// have been queued while its repository was still permitted.
	allowed, err := w.allowed(ctx, job.Repository.Slug)
	if err != nil {
		w.fail(ctx, job, err.Error())
		return
	}
	if !allowed {
		w.fail(ctx, job, fmt.Sprintf(
			"refusing to run: %s is not on the repository allowlist", job.Repository.Slug))
		return
	}

	workspace, err := w.prepare(ctx, job)
	if err != nil {
		w.fail(ctx, job, err.Error())
		return
	}
	defer func() {
		if err := os.RemoveAll(workspace.Root); err != nil {
			log.Printf("[agent] job %s: cleanup failed: %v", job.Job.ID, err)
		}
	}()

	result := w.execute(ctx, job, workspace)

	status := client.StatusPassed
	switch {
	case result.TimedOut:
		status = client.StatusTimeout
	case result.ExitCode != 0:
		status = client.StatusFailed
	}

	w.report(ctx, job.Job.ID, client.JobStatusRequest{
		Status:   status,
		ExitCode: int64(result.ExitCode),
		Error:    result.Error,
	})

	log.Printf("[agent] job %s: %s (exit %d)", job.Job.ID, status, result.ExitCode)
}

// fail records an agent-side failure: something that went wrong around
// the command rather than inside it.
func (w *Worker) fail(ctx context.Context, job *jobContext, message string) {
	log.Printf("[agent] job %s failed: %s", job.Job.ID, message)

	w.appendLog(ctx, job.Job.ID, client.StreamError, message+"\n")
	w.report(ctx, job.Job.ID, client.JobStatusRequest{
		Status:   client.StatusFailed,
		ExitCode: 1,
		Error:    message,
	})
}

// report settles the job, using a context that survives job timeout so
// a timed-out job still lands as `timeout` rather than as a dead lease.
func (w *Worker) report(ctx context.Context, jobID string, req client.JobStatusRequest) {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), reportTimeout)
	defer cancel()

	if err := w.client.ReportStatus(ctx, jobID, req); err != nil {
		log.Printf("[agent] reporting job %s failed: %v", jobID, err)
	}
}

// appendLog pushes a chunk of output, tolerating failure: losing a log
// line should not lose the job.
func (w *Worker) appendLog(ctx context.Context, jobID, stream, content string) {
	if content == "" {
		return
	}

	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), reportTimeout)
	defer cancel()

	if err := w.client.AppendLog(ctx, jobID, stream, content); err != nil {
		log.Printf("[agent] appending log for job %s failed: %v", jobID, err)
	}
}

// heartbeat renews the lease until the returned function is called.
func (w *Worker) heartbeat(ctx context.Context, jobID string) func() {
	ctx, cancel := context.WithCancel(ctx)

	done := make(chan struct{})
	go func() {
		defer close(done)

		ticker := time.NewTicker(w.opts.HeartbeatInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := w.client.Heartbeat(ctx, jobID, w.opts.AgentID); err != nil {
					log.Printf("[agent] heartbeat for job %s failed: %v", jobID, err)
				}
			}
		}
	}()

	return func() {
		cancel()
		<-done
	}
}

// reportTimeout bounds the calls that record an outcome.
const reportTimeout = 15 * time.Second

// sleep waits for d, reporting false if the context ended first.
func sleep(ctx context.Context, d time.Duration) bool {
	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

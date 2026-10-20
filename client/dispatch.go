package client

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
)

// Environment variables a dispatched run exports to its steps. They are
// the CI/CD contract: a command running under an agent can tell that it
// belongs to a job, address that job, and dispatch children under it.
const (
	// EnvJobID is the job the command is running as. A nested atkins
	// invocation reads it and dispatches its own job as a child.
	EnvJobID = "ATKINS_JOB_ID"

	// EnvParentJobID is the job that dispatched this one, if any.
	EnvParentJobID = "ATKINS_PARENT_JOB_ID"

	// EnvRootJobID is the top of this job's tree, stable across nesting.
	EnvRootJobID = "ATKINS_ROOT_JOB_ID"

	// EnvJobParams is the JSON object the job carries from its dispatch.
	EnvJobParams = "ATKINS_JOB_PARAMS"

	// EnvArtefacts is a directory the job copies files into to have
	// them kept. When the command exits, the agent uploads whatever the
	// directory holds and attaches it to the job.
	//
	// The directory is the declaration that needs no schema: a pipeline says what
	// it wants kept by putting it somewhere, which works for any
	// command in any language without atkins having to parse its
	// output or its configuration.
	EnvArtefacts = "ATKINS_ARTEFACTS"

	// EnvServer selects which logged-in server to dispatch to when a
	// machine has credentials for more than one.
	EnvServer = "ATKINS_SERVER"

	// EnvLabels constrains which agents may run the dispatched job,
	// comma separated.
	EnvLabels = "ATKINS_LABELS"

	// EnvNoDispatch makes atkins run locally and skip dispatch, without
	// logging out. An agent sets it for the command it runs, and that
	// stops a job from dispatching itself forever.
	EnvNoDispatch = "ATKINS_NO_DISPATCH"

	// EnvCI marks every run an agent performs, matching what
	// other CI systems export.
	EnvCI = "CI"
)

// Dispatched is a job that a server accepted.
//
// A nil *Dispatched means the server holds no job for the run and the
// caller executes it locally.
type Dispatched struct {
	// JobID is the server's ID for the run.
	JobID string

	// URL is where a browser shows the job. atkins prints this one
	// line when it delegates a run.
	URL string

	// RepositorySlug is the normalized repository the job belongs to.
	RepositorySlug string
}

// DispatchOptions describes the run that atkins dispatches.
type DispatchOptions struct {
	// Directory is the directory atkins runs in. Empty means the process
	// working directory.
	Directory string

	// Command is the atkins invocation. Empty means os.Args.
	Command string

	// Server overrides which logged-in server to dispatch to.
	Server string
}

// ErrDispatchDisabled reports that the environment forbids dispatching
// the run: an agent sets the variable for the command it runs, and that
// stops a job from dispatching itself forever.
var ErrDispatchDisabled = errors.New(EnvNoDispatch + " is set")

// Dispatch sends the current run to the CI/CD server this machine has
// logged in to, and returns where to watch it.
//
// Dispatch runs only on request (--dispatch, or `client.dispatch` in the
// configuration), so every reason it can't happen is an error and never
// a quiet local run. A repository nobody can clone gets its refusal
// before the job exists: an unpushed commit dispatches a job
// that dies in `git checkout` on a machine nobody is watching.
func Dispatch(ctx context.Context, opts DispatchOptions) (*Dispatched, error) {
	if DispatchDisabled() {
		return nil, ErrDispatchDisabled
	}

	c, err := Open(configuredServer(opts.Server))
	if err != nil {
		return nil, fmt.Errorf("not logged in to an atkins server: %w", err)
	}

	directory := opts.Directory
	if directory == "" {
		directory, _ = os.Getwd()
	}

	checkout, err := DetectCheckout(directory)
	if err != nil {
		return nil, err
	}
	switch err := checkout.Publishable(); {
	case errors.Is(err, ErrDirtyCheckout):
		return nil, fmt.Errorf("%w: an agent would build %s, which does not have them", err, shortRevision(checkout.Revision))
	case errors.Is(err, ErrUnpushedCheckout):
		return nil, fmt.Errorf("%w: an agent clones the repository, and %s is only on this machine", err, shortRevision(checkout.Revision))
	case err != nil:
		return nil, err
	}

	command := opts.Command
	if command == "" {
		command = Command(os.Args)
	}

	ctx, cancel := context.WithTimeout(ctx, configuredTimeout())
	defer cancel()

	response, err := c.Dispatch(ctx, DispatchRequest{
		Repository:       checkout.Payload(),
		WorkingDirectory: checkout.WorkingDirectory,
		Command:          command,
		ParentID:         os.Getenv(EnvJobID),
		Labels:           Labels(),
	})
	if err != nil {
		return nil, fmt.Errorf("dispatch to %s failed: %w", c.Server(), err)
	}

	return &Dispatched{
		JobID:          response.JobID,
		URL:            c.JobURL(response.JobID, response.ViewToken),
		RepositorySlug: response.RepositorySlug,
	}, nil
}

// shortRevision abbreviates a commit for a message, leaving anything
// that isn't a full sha alone.
func shortRevision(revision string) string {
	if len(revision) > 12 {
		return revision[:12]
	}
	if revision == "" {
		return "HEAD"
	}
	return revision
}

// DispatchDisabled reports whether ATKINS_NO_DISPATCH holds a
// true-ish value.
func DispatchDisabled() bool {
	value := strings.TrimSpace(os.Getenv(EnvNoDispatch))
	switch strings.ToLower(value) {
	case "", "0", "false", "no":
		return false
	}
	return true
}

// Command renders an argv as the command to record.
//
// Command replaces argv[0] with `atkins` and keeps no part of the local
// name: the recorded command runs again on another machine, where the
// binary has the name `atkins`, and the local binary's name is a
// fact about this laptop. A build under test as `./bin/atkins-linux-amd64`,
// the `atkins.old` the install job leaves behind, or a distro package
// named `atkins-cli` would otherwise queue a command no agent can run,
// and the job comes back as exit 127 with an empty log.
//
// A run of something other than atkins is what the explicit `command`
// override on the trigger endpoint is for.
func Command(argv []string) string {
	if len(argv) == 0 {
		return commandName
	}

	command := make([]string, len(argv))
	copy(command, argv)
	command[0] = commandName

	return strings.Join(command, " ")
}

// commandName is what an agent invokes, whatever the local binary's
// name.
const commandName = "atkins"

// Labels returns the labels constraining which agents may run this
// machine's jobs.
func Labels() []string {
	if labels := Settings().Labels; len(labels) > 0 {
		return labels
	}
	return SplitLabels(os.Getenv(EnvLabels))
}

// SplitLabels parses a comma separated label list.
func SplitLabels(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}

	parts := strings.Split(value, ",")
	labels := make([]string, 0, len(parts))
	for _, part := range parts {
		if part = strings.TrimSpace(part); part != "" {
			labels = append(labels, part)
		}
	}
	return labels
}

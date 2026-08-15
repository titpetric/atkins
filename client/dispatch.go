package client

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Environment variables a dispatched run exports to its steps. They are
// the CI/CD contract: a command running under an agent can tell it is
// part of a job, address that job, and dispatch children under it.
const (
	// EnvJobID is the job the command is running as. A nested atkins
	// invocation reads it and dispatches its own job as a child.
	EnvJobID = "ATKINS_JOB_ID"

	// EnvParentJobID is the job that dispatched this one, if any.
	EnvParentJobID = "ATKINS_PARENT_JOB_ID"

	// EnvRootJobID is the top of this job's tree, stable across nesting.
	EnvRootJobID = "ATKINS_ROOT_JOB_ID"

	// EnvJobParams is the JSON object the job was dispatched with.
	EnvJobParams = "ATKINS_JOB_PARAMS"

	// EnvArtefacts is a directory the job copies files into to have
	// them kept. Whatever is in it when the command exits is uploaded
	// and attached to the job.
	//
	// It is the declaration that needs no schema: a pipeline says what
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

	// EnvNoDispatch runs locally instead of dispatching, without
	// logging out. An agent sets it for the command it runs, which is
	// what stops a job from dispatching itself forever.
	EnvNoDispatch = "ATKINS_NO_DISPATCH"

	// EnvCI is set for every run an agent performs, matching what
	// other CI systems export.
	EnvCI = "CI"
)

// Dispatched is a job handed to a server.
//
// A nil *Dispatched means the run was not delegated and the caller
// should execute it locally.
type Dispatched struct {
	// JobID is the server's ID for the run.
	JobID string

	// URL is where the job is watched in a browser. It is the only
	// thing atkins prints when it delegates a run.
	URL string

	// RepositorySlug is the normalized repository the job belongs to.
	RepositorySlug string
}

// DispatchOptions describes the run being handed over.
type DispatchOptions struct {
	// Directory is where atkins was invoked. Empty means the process
	// working directory.
	Directory string

	// Command is the atkins invocation. Empty means os.Args.
	Command string

	// Server overrides which logged-in server to dispatch to.
	Server string

	// Local forces the run to happen here, as if the machine were not
	// logged in.
	Local bool
}

// Dispatch hands the current run to the CI/CD server this machine is
// logged in to, and returns where to watch it.
//
// It returns nil whenever the run belongs here instead: no credential,
// not inside a git repository, dispatch disabled, or the server could
// not be reached. Delegation is a convenience, and losing it should
// degrade to the behaviour of an unattached machine rather than to an
// error.
func Dispatch(ctx context.Context, opts DispatchOptions) *Dispatched {
	if opts.Local || !Settings().Dispatch || DispatchDisabled() {
		return nil
	}

	c, err := Open(configuredServer(opts.Server))
	if err != nil {
		// Not logged in is the common case for a laptop; say nothing.
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
		fmt.Fprintf(os.Stderr, "atkins: dispatch to %s failed, running locally: %v\n", c.Server(), err)
		return nil
	}

	return &Dispatched{
		JobID:          response.JobID,
		URL:            c.JobURL(response.JobID, response.ViewToken),
		RepositorySlug: response.RepositorySlug,
	}
}

// DispatchDisabled reports whether ATKINS_NO_DISPATCH is set to a
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
// argv[0] is reduced to its base name: the recorded command is re-run
// on another machine, and `/home/tit/go/bin/atkins` says more about
// this laptop than about the job.
func Command(argv []string) string {
	if len(argv) == 0 {
		return "atkins"
	}

	command := make([]string, len(argv))
	copy(command, argv)
	command[0] = filepath.Base(command[0])

	return strings.Join(command, " ")
}

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

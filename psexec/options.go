package psexec

import "time"

// Options configures the Executor.
type Options struct {
	// DefaultTimeout is the default timeout for commands.
	DefaultTimeout time.Duration
	// DefaultDir is the default working directory for commands.
	DefaultDir string
	// DefaultEnv is the default environment variables for commands.
	DefaultEnv []string
	// DefaultShell is the shell to use for shell commands.
	DefaultShell string
}

// DefaultOptions returns the default options.
func DefaultOptions() *Options {
	return &Options{
		DefaultTimeout: 0, // No timeout
		DefaultShell:   "bash",
	}
}

// NewWithOptions creates a new Executor with the given options.
func NewWithOptions(opts *Options) *Executor {
	if opts == nil {
		opts = DefaultOptions()
	}
	shell := opts.DefaultShell
	if shell == "" {
		shell = "bash"
	}
	return &Executor{
		DefaultDir:     opts.DefaultDir,
		DefaultEnv:     opts.DefaultEnv,
		DefaultTimeout: opts.DefaultTimeout,
		DefaultShell:   shell,
	}
}

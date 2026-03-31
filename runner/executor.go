package runner

// Executor runs pipeline jobs and steps.
type Executor struct {
	opts *Options
}

// NewExecutor creates a new executor with default options.
func NewExecutor() *Executor {
	return &Executor{
		opts: DefaultOptions(),
	}
}

// NewExecutorWithOptions creates a new executor with custom options.
func NewExecutorWithOptions(opts *Options) *Executor {
	if opts == nil {
		opts = DefaultOptions()
	}
	return &Executor{
		opts: opts,
	}
}

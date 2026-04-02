package agent

// Options configures agent behavior.
type Options struct {
	Debug   bool
	Verbose bool
	Jail    bool
}

// State represents the current REPL state.
type State int

// State constants for the REPL.
const (
	StateIdle State = iota
	StateExecuting
	StateAutofix
	StateRetrying
)

// GitStats holds +/- line counts from git diff.
type GitStats struct {
	Added   int
	Removed int
}

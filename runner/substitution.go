package runner

import (
	"sync"
)

// substitutionResult is what one `$(...)` produced: everything the
// caller reads off the process, so a memo of it behaves as the run did.
type substitutionResult struct {
	output   string
	success  bool
	exitCode int
	err      string
}

// substitutionCache memoises `$(...)` results for one step execution.
//
// A step is interpolated more than once before it runs: the tree renders
// its label, the condition may read it, and the command itself is
// interpolated last. Every one of those passes over the same text, so
// without a memo the shell command inside it runs several times. A
// substitution that appends to a file, bumps a counter or takes a lock
// would do it once per pass, and a slow one would cost its time over
// again.
//
// The scope is one step execution, and one iteration of a loop is one
// execution. A later step asking the same question asks it again,
// because between two steps the answer may have changed - that is the
// whole reason a pipeline runs them in order.
//
// A nil cache memoises nothing and every method tolerates it, so a
// context assembled without one still runs.
type substitutionCache struct {
	mu      sync.Mutex
	results map[string]substitutionResult
}

// newSubstitutionCache returns an empty cache for one step execution.
func newSubstitutionCache() *substitutionCache {
	return &substitutionCache{results: map[string]substitutionResult{}}
}

// lookup returns what a command produced earlier in this step.
func (c *substitutionCache) lookup(key string) (substitutionResult, bool) {
	if c == nil {
		return substitutionResult{}, false
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	result, ok := c.results[key]
	return result, ok
}

// store keeps what a command produced, for the rest of this step.
func (c *substitutionCache) store(key string, result substitutionResult) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	c.results[key] = result
}

// substitutionKey identifies a substitution within a step: the command
// as it will be run, and the directory it runs in. The directory is part
// of it because a loop may change directory between iterations, and the
// same command there is a different question.
func substitutionKey(dir, command string) string {
	return dir + "\x00" + command
}

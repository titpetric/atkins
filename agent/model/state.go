package model

// State represents the current REPL state.
type State int

// State constants for the REPL.
const (
	StateIdle State = iota
	StateExecuting
	StateAutofix
	StateRetrying
)

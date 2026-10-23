package storage

import (
	"time"
)

// SampleRequest selects the commands one by one, each its own row. A
// zero value returns every command the database holds.
type SampleRequest struct {
	// Since and Until bound when a command started, Tool keeps one
	// tool and Dir one directory and below, as for StatsRequest.
	Since time.Time
	Until time.Time
	Tool  string
	Dir   string

	// Limit keeps the newest commands of the window. Zero keeps them
	// all: a reader that buckets the samples needs the whole window,
	// and the retention window bounds it.
	Limit int
}

// Sample is one recorded command: when it started, its key and how it
// ended.
//
// It carries the command's key and never the command line: a command
// line holds its values, and a value can be a token.
type Sample struct {
	// At is when the command started.
	At time.Time

	// Command is the key, as Key names it, and Tool its first word.
	Command string
	Tool    string

	// Kind is where the command came from: a step, a $() substitution
	// or a variable.
	Kind string

	// DurationMs is how long it took and ExitCode how it ended.
	DurationMs int64
	ExitCode   int64

	// Pipeline and Job are what ran it, and InvocationID the atkins
	// process, so the commands of one run group together.
	Pipeline     string
	Job          string
	InvocationID string
}

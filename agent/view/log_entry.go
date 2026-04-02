package view

import "time"

// LogEntry represents a single entry in the message log.
type LogEntry struct {
	Kind     string // "info", "error", "run", "prompt", "output"
	Text     string
	Task     string
	Running  bool
	Duration time.Duration
	Failed   bool
}

package model

import "time"

// ShellDoneMsg signals a shell command completed.
type ShellDoneMsg struct {
	Command  string
	Output   string
	Err      error
	ExitCode int
	Duration time.Duration
}

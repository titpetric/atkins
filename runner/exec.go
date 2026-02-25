package runner

import (
	"github.com/titpetric/atkins/psexec"
)

// ExecError represents an error from command execution.
type ExecError struct {
	Message      string
	Output       string
	LastExitCode int
}

// NewExecError creates an ExecError from a psexec.Result.
func NewExecError(result psexec.Result) ExecError {
	msg := "command failed"
	if result.Err() != nil {
		msg = result.Err().Error()
	}
	out := result.ErrorOutput()
	if out == "" {
		out = result.Output()
	}
	return ExecError{
		Message:      msg,
		Output:       out,
		LastExitCode: result.ExitCode(),
	}
}

// Error implements the error interface.
func (e ExecError) Error() string {
	return e.Message
}

// Len returns the length of the error output.
func (e ExecError) Len() int {
	return len(e.Output)
}

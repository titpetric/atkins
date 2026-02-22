// Package psexec provides process execution capabilities with support for
// interactive terminals, PTY allocation, and streaming output over various
// transports including websockets.
package psexec

import "bytes"

// Result provides access to the outcome of a process execution.
type Result interface {
	// Output returns the combined stdout content.
	Output() string
	// ErrorOutput returns the stderr content.
	ErrorOutput() string
	// ExitCode returns the process exit code.
	ExitCode() int
	// Err returns any error that occurred during execution.
	Err() error
	// Success returns true if the process completed with exit code 0.
	Success() bool
}

// processResult implements the Result interface.
type processResult struct {
	stdout   *bytes.Buffer
	stderr   *bytes.Buffer
	exitCode int
	err      error
}

// Output returns the captured stdout.
func (r *processResult) Output() string {
	if r.stdout == nil {
		return ""
	}
	return r.stdout.String()
}

// ErrorOutput returns the captured stderr.
func (r *processResult) ErrorOutput() string {
	if r.stderr == nil {
		return ""
	}
	return r.stderr.String()
}

// ExitCode returns the process exit code.
func (r *processResult) ExitCode() int {
	return r.exitCode
}

// Err returns any error that occurred.
func (r *processResult) Err() error {
	return r.err
}

// Success returns true if exit code is 0 and no error occurred.
func (r *processResult) Success() bool {
	return r.exitCode == 0 && r.err == nil
}

// newResult creates a new processResult.
func newResult() *processResult {
	return &processResult{
		stdout: new(bytes.Buffer),
		stderr: new(bytes.Buffer),
	}
}

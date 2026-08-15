//go:build !unix

package worker

import "os/exec"

// setProcessGroup is a no-op where process groups are not available.
// The context still kills the command itself; anything it spawned may
// outlive it, and a job timeout is correspondingly weaker.
func setProcessGroup(_ *exec.Cmd) {}

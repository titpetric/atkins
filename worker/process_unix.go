//go:build unix

package worker

import (
	"os/exec"
	"syscall"
)

// setProcessGroup puts a job's command in its own process group and
// kills the whole group when the context ends.
//
// Without it, exec.CommandContext signals only the shell it started. A
// job's real work is that shell's children, and they inherit the pipes
// the agent is reading, so killing the shell alone leaves the agent
// blocked on output from processes nothing is going to stop — and a
// job timeout stops meaning anything.
func setProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	cmd.Cancel = func() error {
		if cmd.Process == nil {
			return nil
		}
		return killProcessGroup(cmd.Process.Pid)
	}
}

// killProcessGroup stops the group a process leads.
//
// It is separate from setProcessGroup because an interactive job does
// not use that: pty.Start puts the command in a session of its own, so
// the group already exists and only the killing is wanted.
func killProcessGroup(pid int) error {
	// Negative pid: the process group, not just the leader.
	return syscall.Kill(-pid, syscall.SIGKILL)
}

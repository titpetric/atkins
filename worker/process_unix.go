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
		// Negative pid: the process group, not just the leader.
		return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
}

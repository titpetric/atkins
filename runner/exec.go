package runner

import (
	"os"
	"os/exec"
	"strings"
)

// ExecError represents an error from command execution.
type ExecError struct {
	Message      string
	Output       string
	LastExitCode int
}

// Error returns the error message.
func (r ExecError) Error() string {
	return r.Message
}

// Len returns the length of the error message.
func (r ExecError) Len() int {
	return len(r.Message)
}

// Exec runs shell commands.
type Exec struct{}

// NewExec creates a new Exec instance.
func NewExec() *Exec {
	return &Exec{}
}

// ExecuteCommand will run the command quietly.
func (e *Exec) ExecuteCommand(cmdStr string) (string, error) {
	return e.ExecuteCommandWithQuiet(cmdStr, false)
}

// ExecuteCommandWithQuiet executes a shell command with quiet mode.
func (e *Exec) ExecuteCommandWithQuiet(cmdStr string, verbose bool) (string, error) {
	return e.ExecuteCommandWithQuietAndCapture(cmdStr, verbose)
}

// ExecuteCommandWithQuietAndCapture executes a shell command with quiet mode and captures stderr.
// Returns (stdout, error). If error occurs, stderr is logged to the global buffer.
func (e *Exec) ExecuteCommandWithQuietAndCapture(cmdStr string, verbose bool) (string, error) {
	if cmdStr == "" {
		return "", nil
	}

	cmd := exec.Command("bash", "-c", cmdStr)

	// Inherit current process environment
	cmd.Env = os.Environ()

	output, err := cmd.CombinedOutput()

	if err != nil {
		// Extract exit code
		exitCode := 1
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		}

		resErr := ExecError{
			Message:      "failed to run command: " + err.Error(),
			LastExitCode: exitCode,
			Output:       string(output),
		}

		return "", resErr
	}

	return string(output), nil
}

// removeEnvKey removes a key from environment variable list
func removeEnvKey(env []string, key string) []string {
	prefix := key + "="
	result := make([]string, 0, len(env))
	for _, e := range env {
		if !strings.HasPrefix(e, prefix) {
			result = append(result, e)
		}
	}
	return result
}

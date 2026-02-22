package psexec_test

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/titpetric/atkins/psexec"
)

func TestNewCommand(t *testing.T) {
	cmd := psexec.NewCommand("echo", "hello", "world")

	assert.Equal(t, "echo", cmd.Name)
	assert.Equal(t, []string{"hello", "world"}, cmd.Args)
}

func TestNewShellCommand(t *testing.T) {
	cmd := psexec.NewShellCommand("echo $HOME && ls")

	assert.Equal(t, "bash", cmd.Name)
	assert.Equal(t, []string{"-c", "echo $HOME && ls"}, cmd.Args)
}

func TestCommand_WithDir(t *testing.T) {
	cmd := psexec.NewCommand("pwd").WithDir("/tmp")

	assert.Equal(t, "/tmp", cmd.Dir)
}

func TestCommand_WithEnv(t *testing.T) {
	env := []string{"FOO=bar", "BAZ=qux"}
	cmd := psexec.NewCommand("env").WithEnv(env)

	assert.Equal(t, env, cmd.Env)
}

func TestCommand_WithStdin(t *testing.T) {
	reader := strings.NewReader("input")
	cmd := psexec.NewCommand("cat").WithStdin(reader)

	assert.Equal(t, reader, cmd.Stdin)
}

func TestCommand_WithStdout(t *testing.T) {
	var buf bytes.Buffer
	cmd := psexec.NewCommand("echo").WithStdout(&buf)

	assert.Equal(t, &buf, cmd.Stdout)
}

func TestCommand_WithStderr(t *testing.T) {
	var buf bytes.Buffer
	cmd := psexec.NewCommand("test").WithStderr(&buf)

	assert.Equal(t, &buf, cmd.Stderr)
}

func TestCommand_WithTimeout(t *testing.T) {
	cmd := psexec.NewCommand("sleep").WithTimeout(5 * time.Second)

	assert.Equal(t, 5*time.Second, cmd.Timeout)
}

func TestCommand_WithPTY(t *testing.T) {
	cmd := psexec.NewCommand("vim").WithPTY()

	assert.True(t, cmd.UsePTY)
	assert.False(t, cmd.Interactive)
}

func TestCommand_AsInteractive(t *testing.T) {
	cmd := psexec.NewCommand("bash").AsInteractive()

	assert.True(t, cmd.UsePTY)
	assert.True(t, cmd.Interactive)
}

func TestCommand_Chaining(t *testing.T) {
	var buf bytes.Buffer
	input := strings.NewReader("data")

	cmd := psexec.NewCommand("test", "arg").
		WithDir("/tmp").
		WithEnv([]string{"KEY=value"}).
		WithStdin(input).
		WithStdout(&buf).
		WithTimeout(10 * time.Second).
		WithPTY()

	assert.Equal(t, "test", cmd.Name)
	assert.Equal(t, []string{"arg"}, cmd.Args)
	assert.Equal(t, "/tmp", cmd.Dir)
	assert.Equal(t, []string{"KEY=value"}, cmd.Env)
	assert.Equal(t, input, cmd.Stdin)
	assert.Equal(t, &buf, cmd.Stdout)
	assert.Equal(t, 10*time.Second, cmd.Timeout)
	assert.True(t, cmd.UsePTY)
}

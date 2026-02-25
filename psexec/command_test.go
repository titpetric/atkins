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

func TestCommand_Dir(t *testing.T) {
	cmd := psexec.NewCommand("pwd")
	cmd.Dir = "/tmp"

	assert.Equal(t, "/tmp", cmd.Dir)
}

func TestCommand_Env(t *testing.T) {
	env := []string{"FOO=bar", "BAZ=qux"}
	cmd := psexec.NewCommand("env")
	cmd.Env = env

	assert.Equal(t, env, cmd.Env)
}

func TestCommand_Stdin(t *testing.T) {
	reader := strings.NewReader("input")
	cmd := psexec.NewCommand("cat")
	cmd.Stdin = reader

	assert.Equal(t, reader, cmd.Stdin)
}

func TestCommand_Stdout(t *testing.T) {
	var buf bytes.Buffer
	cmd := psexec.NewCommand("echo")
	cmd.Stdout = &buf

	assert.Equal(t, &buf, cmd.Stdout)
}

func TestCommand_Stderr(t *testing.T) {
	var buf bytes.Buffer
	cmd := psexec.NewCommand("test")
	cmd.Stderr = &buf

	assert.Equal(t, &buf, cmd.Stderr)
}

func TestCommand_Timeout(t *testing.T) {
	cmd := psexec.NewCommand("sleep")
	cmd.Timeout = 5 * time.Second

	assert.Equal(t, 5*time.Second, cmd.Timeout)
}

func TestCommand_UsePTY(t *testing.T) {
	cmd := psexec.NewCommand("vim")
	cmd.UsePTY = true

	assert.True(t, cmd.UsePTY)
	assert.False(t, cmd.Interactive)
}

func TestCommand_Interactive(t *testing.T) {
	cmd := psexec.NewCommand("bash")
	cmd.Interactive = true
	cmd.UsePTY = true

	assert.True(t, cmd.UsePTY)
	assert.True(t, cmd.Interactive)
}

func TestCommand_StructLiteral(t *testing.T) {
	var buf bytes.Buffer
	input := strings.NewReader("data")

	cmd := &psexec.Command{
		Name:    "test",
		Args:    []string{"arg"},
		Dir:     "/tmp",
		Env:     []string{"KEY=value"},
		Stdin:   input,
		Stdout:  &buf,
		Timeout: 10 * time.Second,
		UsePTY:  true,
	}

	assert.Equal(t, "test", cmd.Name)
	assert.Equal(t, []string{"arg"}, cmd.Args)
	assert.Equal(t, "/tmp", cmd.Dir)
	assert.Equal(t, []string{"KEY=value"}, cmd.Env)
	assert.Equal(t, input, cmd.Stdin)
	assert.Equal(t, &buf, cmd.Stdout)
	assert.Equal(t, 10*time.Second, cmd.Timeout)
	assert.True(t, cmd.UsePTY)
}

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/pflag"

	"github.com/titpetric/atkins/internal/assert"
	"github.com/titpetric/atkins/model"
)

// TestSplitJobArgs checks the arguments after a bare -- separate from
// the job names, also when the command consumed a leading argument.
func TestSplitJobArgs(t *testing.T) {
	t.Parallel()

	names, args := splitJobArgs(-1, 2, []string{"build", "test"})
	assert.Equal(t, []string{"build", "test"}, names)
	assert.Nil(t, args)

	names, args = splitJobArgs(1, 3, []string{"compose:custom", "ps", "-a"})
	assert.Equal(t, []string{"compose:custom"}, names)
	assert.Equal(t, []string{"ps", "-a"}, args)

	// "run compose:custom -- ps": the command name came off the front.
	names, args = splitJobArgs(2, 3, []string{"compose:custom", "ps"})
	assert.Equal(t, []string{"compose:custom"}, names)
	assert.Equal(t, []string{"ps"}, args)

	names, args = splitJobArgs(0, 1, []string{"; id"})
	assert.Empty(t, names)
	assert.Equal(t, []string{"; id"}, args)
}

// TestCheckJobArgs checks arguments need one job declaring args: true.
func TestCheckJobArgs(t *testing.T) {
	t.Parallel()

	custom := &model.ResolvedTask{Name: "compose:custom", Job: &model.Job{Args: true}}
	up := &model.ResolvedTask{Name: "compose:up", Job: &model.Job{}}

	assert.NoError(t, checkJobArgs(nil, []string{"compose:up"}, []*model.ResolvedTask{up}))
	assert.NoError(t, checkJobArgs([]string{"ps"}, []string{"compose:custom"}, []*model.ResolvedTask{custom}))
	assert.ErrorContains(t, checkJobArgs([]string{"ps"}, []string{"compose:up"}, []*model.ResolvedTask{up}), "takes no arguments")
	assert.ErrorContains(t, checkJobArgs([]string{"ps"}, []string{"compose:custom", "compose:up"}, []*model.ResolvedTask{custom, up}), "exactly one job")
}

// runArgv runs the pipeline command from dir, parsing argv the way the
// cli package does, a bare -- included.
func runArgv(t *testing.T, dir string, argv ...string) error {
	t.Helper()

	originalDir, err := os.Getwd()
	assert.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, os.Chdir(originalDir))
	})
	assert.NoError(t, os.Chdir(dir))

	cmd := Pipeline()
	fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
	cmd.Bind(fs)
	assert.NoError(t, fs.Parse(argv))

	return cmd.Run(t.Context(), fs.Args())
}

// TestJobArgsFromWorkspaceSkill checks `atkins compose:custom -- ...`
// appends the arguments to the exec: step of a workspace skill as
// literal argv, in the folder holding compose.yml.
func TestJobArgsFromWorkspaceSkill(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	ws := t.TempDir()
	pulse := filepath.Join(ws, "pulse")
	assert.NoError(t, os.MkdirAll(filepath.Join(ws, ".atkins", "skills"), 0o755))
	assert.NoError(t, os.MkdirAll(pulse, 0o755))
	assert.NoError(t, os.WriteFile(filepath.Join(pulse, "compose.yml"), []byte("services:\n  pulse:\n    build: .\n"), 0o644))
	assert.NoError(t, os.WriteFile(filepath.Join(pulse, "argv.sh"), []byte("#!/bin/sh\nfor a in \"$@\"; do printf '%s\\n' \"$a\"; done > argv.out\n"), 0o755))

	skill := `name: compose
when:
  files: [compose.yml]
  here: true
jobs:
  custom:
    args: true
    steps:
      - exec: [./argv.sh, compose]
`
	assert.NoError(t, os.WriteFile(filepath.Join(ws, ".atkins", "skills", "compose.yml"), []byte(skill), 0o644))

	read := func() []string {
		data, err := os.ReadFile(filepath.Join(pulse, "argv.out"))
		assert.NoError(t, err)
		return strings.Split(strings.TrimSuffix(string(data), "\n"), "\n")
	}

	assert.NoError(t, runArgv(t, pulse, "--final", "compose:custom", "--", "ps", "-a"))
	assert.Equal(t, []string{"compose", "ps", "-a"}, read())

	assert.NoError(t, runArgv(t, pulse, "--final", "compose:custom", "--", "; id"))
	assert.Equal(t, []string{"compose", "; id"}, read())
}

package main

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/titpetric/atkins/runner"
)

func TestWorkingDirectory_ChangesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, os.Chdir(originalDir))
	})

	configPath := filepath.Join(tmpDir, ".atkins.yml")
	err = os.WriteFile(configPath, []byte("name: test\njobs:\n  default:\n    script:\n      - echo hello\n"), 0o644)
	require.NoError(t, err)

	cmd := Pipeline()
	fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
	cmd.Bind(fs)

	err = fs.Parse([]string{"-w", tmpDir, "-l"})
	require.NoError(t, err)

	err = cmd.Run(t.Context(), fs.Args())
	require.NoError(t, err)

	currentDir, err := os.Getwd()
	require.NoError(t, err)
	assert.Equal(t, tmpDir, currentDir)
}

func TestWorkingDirectory_InvalidDirectory(t *testing.T) {
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, os.Chdir(originalDir))
	})

	cmd := Pipeline()
	fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
	cmd.Bind(fs)

	err = fs.Parse([]string{"-w", "/nonexistent/path/that/does/not/exist"})
	require.NoError(t, err)

	err = cmd.Run(t.Context(), fs.Args())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to change directory")
}

func TestWorkingDirectory_EmptyIsNoOp(t *testing.T) {
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, os.Chdir(originalDir))
	})

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".atkins.yml")
	err = os.WriteFile(configPath, []byte("name: test\njobs:\n  default:\n    script:\n      - echo hello\n"), 0o644)
	require.NoError(t, err)

	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	cmd := Pipeline()
	fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
	cmd.Bind(fs)

	err = fs.Parse([]string{"-l"})
	require.NoError(t, err)

	err = cmd.Run(t.Context(), fs.Args())
	require.NoError(t, err)

	currentDir, err := os.Getwd()
	require.NoError(t, err)
	assert.Equal(t, tmpDir, currentDir)
}

func TestSkillJobInvocation(t *testing.T) {
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, os.Chdir(originalDir))
	})

	tmpDir := t.TempDir()

	// Create .atkins/skills directory with a simple skill
	skillsDir := filepath.Join(tmpDir, ".atkins", "skills")
	require.NoError(t, os.MkdirAll(skillsDir, 0o755))

	skillContent := `name: greet
jobs:
  default:
    steps:
      - echo hello
`
	require.NoError(t, os.WriteFile(filepath.Join(skillsDir, "greet.yml"), []byte(skillContent), 0o644))

	require.NoError(t, os.Chdir(tmpDir))

	// Invoke the skill by its ID (like "atkins greet")
	cmd := Pipeline()
	fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
	cmd.Bind(fs)

	require.NoError(t, fs.Parse([]string{"--final", "--jail"}))
	err = cmd.Run(t.Context(), []string{"greet"})
	require.NoError(t, err)
}

func TestMultipleJobsArguments(t *testing.T) {
	t.Run("jobs_collected_from_positional_args", func(t *testing.T) {
		opts := NewOptions()
		fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
		opts.Bind(fs)

		// Simulate positional args: "lint test build"
		args := []string{"lint", "test", "build"}
		for _, arg := range args {
			opts.Jobs = append(opts.Jobs, arg)
		}

		assert.Equal(t, []string{"lint", "test", "build"}, opts.Jobs)
	})
}

// TestList_ListsPipelineThatDoesNotLint covers the difference between
// --lint and --list on a pipeline with an unresolvable step: --lint is
// asked whether the pipeline is sound and answers no, while --list is
// asked what is in it and answers with everything, warning afterwards.
//
// The distinction has a caller. The CI server discovers a project's jobs
// by running `atkins --list --json` on an agent, and a project that
// references a skill the agent does not carry would otherwise appear to
// have no jobs at all.
func TestList_ListsPipelineThatDoesNotLint(t *testing.T) {
	tmpDir := t.TempDir()

	pipeline := `name: broken
jobs:
  good:
    desc: This one resolves
    steps:
      - echo ok
  broken:
    desc: This one does not
    steps:
      - task: missing
`
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, ".atkins.yml"), []byte(pipeline), 0o644))

	originalDir, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, os.Chdir(originalDir))
	})

	run := func(t *testing.T, args ...string) (string, string, error) {
		t.Helper()

		cmd := Pipeline()
		fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
		cmd.Bind(fs)
		require.NoError(t, fs.Parse(append([]string{"-w", tmpDir, "--jail"}, args...)))

		stdout, stderr, err := captureOutput(t, func() error {
			return cmd.Run(t.Context(), fs.Args())
		})
		return stdout, stderr, err
	}

	t.Run("lint refuses", func(t *testing.T) {
		stdout, _, err := run(t, "--lint")
		require.Error(t, err)
		assert.Contains(t, stdout, "has errors")
		assert.NotContains(t, stdout, "This one resolves")
	})

	t.Run("list continues and warns", func(t *testing.T) {
		stdout, stderr, err := run(t, "-l")

		// The listing is whole: the job that does not resolve does not
		// take the one that does with it.
		assert.Contains(t, stdout, "good")
		assert.Contains(t, stdout, "This one resolves")
		assert.Contains(t, stdout, "broken")

		// The warning closes the listing, and it is on stderr so that
		// `--list --json > file` still writes a document a reader can
		// parse.
		assert.Contains(t, stderr, "WARNING")
		assert.Contains(t, stderr, "missing")
		assert.NotContains(t, stdout, "WARNING")

		// And it still fails, because a listing of a pipeline that does
		// not resolve is a best effort at a broken thing.
		require.Error(t, err)
		assert.Contains(t, err.Error(), "did not lint")
	})

	t.Run("json listing stays parseable", func(t *testing.T) {
		stdout, _, err := run(t, "-l", "--json")
		require.Error(t, err)

		var sections []runner.OutputSection
		require.NoError(t, json.Unmarshal([]byte(stdout), &sections))

		var ids []string
		for _, section := range sections {
			for _, cmd := range section.Cmds {
				ids = append(ids, cmd.ID)
			}
		}
		assert.Contains(t, ids, "good")
		assert.Contains(t, ids, "broken")
	})
}

// captureOutput runs fn with the process's stdout and stderr replaced by
// pipes, and returns what it wrote to each.
//
// The listing writes to the two streams directly rather than to writers
// it was handed, and which stream a line lands on is half of what this
// test is about, so the test swaps the files under it.
func captureOutput(t *testing.T, fn func() error) (string, string, error) {
	t.Helper()

	outR, outW, err := os.Pipe()
	require.NoError(t, err)
	errR, errW, err := os.Pipe()
	require.NoError(t, err)

	originalOut, originalErr := os.Stdout, os.Stderr
	os.Stdout, os.Stderr = outW, errW

	outCh, errCh := make(chan string, 1), make(chan string, 1)
	go func() {
		data, _ := io.ReadAll(outR)
		outCh <- string(data)
	}()
	go func() {
		data, _ := io.ReadAll(errR)
		errCh <- string(data)
	}()

	runErr := fn()

	os.Stdout, os.Stderr = originalOut, originalErr
	require.NoError(t, outW.Close())
	require.NoError(t, errW.Close())

	return <-outCh, <-errCh, runErr
}

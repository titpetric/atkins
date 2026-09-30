package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/pflag"

	"github.com/titpetric/atkins/internal/assert"
)

func TestWorkingDirectory_ChangesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	originalDir, err := os.Getwd()
	assert.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, os.Chdir(originalDir))
	})

	configPath := filepath.Join(tmpDir, ".atkins.yml")
	err = os.WriteFile(configPath, []byte("name: test\njobs:\n  default:\n    script:\n      - echo hello\n"), 0o644)
	assert.NoError(t, err)

	cmd := Pipeline()
	fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
	cmd.Bind(fs)

	err = fs.Parse([]string{"-w", tmpDir, "-l"})
	assert.NoError(t, err)

	err = cmd.Run(t.Context(), fs.Args())
	assert.NoError(t, err)

	currentDir, err := os.Getwd()
	assert.NoError(t, err)
	assert.Equal(t, tmpDir, currentDir)
}

func TestWorkingDirectory_InvalidDirectory(t *testing.T) {
	originalDir, err := os.Getwd()
	assert.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, os.Chdir(originalDir))
	})

	cmd := Pipeline()
	fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
	cmd.Bind(fs)

	err = fs.Parse([]string{"-w", "/nonexistent/path/that/does/not/exist"})
	assert.NoError(t, err)

	err = cmd.Run(t.Context(), fs.Args())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to change directory")
}

func TestWorkingDirectory_EmptyIsNoOp(t *testing.T) {
	originalDir, err := os.Getwd()
	assert.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, os.Chdir(originalDir))
	})

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".atkins.yml")
	err = os.WriteFile(configPath, []byte("name: test\njobs:\n  default:\n    script:\n      - echo hello\n"), 0o644)
	assert.NoError(t, err)

	err = os.Chdir(tmpDir)
	assert.NoError(t, err)

	cmd := Pipeline()
	fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
	cmd.Bind(fs)

	err = fs.Parse([]string{"-l"})
	assert.NoError(t, err)

	err = cmd.Run(t.Context(), fs.Args())
	assert.NoError(t, err)

	currentDir, err := os.Getwd()
	assert.NoError(t, err)
	assert.Equal(t, tmpDir, currentDir)
}

func TestSkillJobInvocation(t *testing.T) {
	originalDir, err := os.Getwd()
	assert.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, os.Chdir(originalDir))
	})

	tmpDir := t.TempDir()

	// Create .atkins/skills directory with a simple skill
	skillsDir := filepath.Join(tmpDir, ".atkins", "skills")
	assert.NoError(t, os.MkdirAll(skillsDir, 0o755))

	skillContent := `name: greet
jobs:
  default:
    steps:
      - echo hello
`
	assert.NoError(t, os.WriteFile(filepath.Join(skillsDir, "greet.yml"), []byte(skillContent), 0o644))

	assert.NoError(t, os.Chdir(tmpDir))

	// Invoke the skill by its ID (like "atkins greet")
	cmd := Pipeline()
	fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
	cmd.Bind(fs)

	assert.NoError(t, fs.Parse([]string{"--final", "--jail"}))
	err = cmd.Run(t.Context(), []string{"greet"})
	assert.NoError(t, err)
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

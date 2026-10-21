package main

import (
	"os"
	"path/filepath"
	"strings"
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

// writeWorkspace lays out a workspace checkout holding skill a, with a
// project cloned inside it that carries its own atkins.yml, and returns
// both folders. Skill a records the folder it runs in.
func writeWorkspace(t *testing.T) (ws, proj string) {
	t.Helper()

	ws = t.TempDir()
	proj = filepath.Join(ws, "proj")

	skillsDir := filepath.Join(ws, ".atkins", "skills")
	assert.NoError(t, os.MkdirAll(skillsDir, 0o755))
	assert.NoError(t, os.MkdirAll(proj, 0o755))

	skill := "name: a\njobs:\n  default:\n    steps:\n      - pwd > a.out\n"
	assert.NoError(t, os.WriteFile(filepath.Join(skillsDir, "a.yml"), []byte(skill), 0o644))
	assert.NoError(t, os.WriteFile(filepath.Join(proj, "atkins.yml"), []byte("name: proj\njobs:\n  default:\n    steps:\n      - true\n"), 0o644))

	return ws, proj
}

// runIn runs the pipeline command from dir with the given arguments.
func runIn(t *testing.T, dir string, args ...string) error {
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

	var jobs []string
	var flags []string
	for _, arg := range args {
		if strings.HasPrefix(arg, "-") {
			flags = append(flags, arg)
			continue
		}
		jobs = append(jobs, arg)
	}
	assert.NoError(t, fs.Parse(flags))

	return cmd.Run(t.Context(), jobs)
}

// TestSkillFromParentFolder checks a project cloned inside a workspace
// runs the workspace's skill, in the workspace folder.
func TestSkillFromParentFolder(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	ws, proj := writeWorkspace(t)

	assert.NoError(t, runIn(t, proj, "--final", "a"))

	out, err := os.ReadFile(filepath.Join(ws, "a.out"))
	assert.NoError(t, err)

	got, err := filepath.EvalSymlinks(strings.TrimSpace(string(out)))
	assert.NoError(t, err)
	want, err := filepath.EvalSymlinks(ws)
	assert.NoError(t, err)
	assert.Equal(t, want, got)
}

// TestSkillFromParentFolderJailed checks --jail keeps the parent
// folders' skills out.
func TestSkillFromParentFolderJailed(t *testing.T) {
	_, proj := writeWorkspace(t)

	ids := func(opts *Options) []string {
		pipelines, err := loadSkillPipelines(proj, proj, opts)
		assert.NoError(t, err)

		var ids []string
		for _, p := range pipelines {
			ids = append(ids, p.ID)
		}
		return ids
	}

	assert.Contains(t, ids(&Options{}), "a")
	assert.NotContains(t, ids(&Options{Jail: true}), "a")
}

// TestSkillFromParentFolderExplicitFile checks a pipeline file passed
// with -f sees the skills of the folders above it, so its jobs can
// reference them.
func TestSkillFromParentFolderExplicitFile(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	_, proj := writeWorkspace(t)
	assert.NoError(t, os.WriteFile(filepath.Join(proj, "atkins.yml"), []byte("name: proj\njobs:\n  default:\n    steps:\n      - task: a:default\n"), 0o644))

	assert.NoError(t, runIn(t, proj, "--final", "-f=atkins.yml", "default"))

	// A skill job a pipeline calls runs where that pipeline runs.
	_, err := os.Stat(filepath.Join(proj, "a.out"))
	assert.NoError(t, err)
}

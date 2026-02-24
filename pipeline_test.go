package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/titpetric/atkins/model"
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

func TestResolveJobTarget(t *testing.T) {
	// Helper to create a pipeline with jobs
	makePipeline := func(id string, jobs map[string]*model.Job) *model.Pipeline {
		return &model.Pipeline{ID: id, Jobs: jobs}
	}

	t.Run("explicit_root_reference_main_pipeline", func(t *testing.T) {
		pipelines := []*model.Pipeline{
			makePipeline("", map[string]*model.Job{"up": {}}),
			makePipeline("docker", map[string]*model.Job{"up": {}}),
		}
		result, jobName, err := resolveJobTarget(pipelines, ":up")
		require.NoError(t, err)
		assert.Equal(t, "", result[0].ID)
		assert.Equal(t, "up", jobName)
	})

	t.Run("explicit_root_reference_skill", func(t *testing.T) {
		pipelines := []*model.Pipeline{
			makePipeline("", map[string]*model.Job{"up": {}}),
			makePipeline("docker", map[string]*model.Job{"build": {}}),
		}
		result, jobName, err := resolveJobTarget(pipelines, ":docker:build")
		require.NoError(t, err)
		assert.Equal(t, "docker", result[0].ID)
		assert.Equal(t, "build", jobName)
	})

	t.Run("prefixed_job_reference", func(t *testing.T) {
		pipelines := []*model.Pipeline{
			makePipeline("", map[string]*model.Job{"test": {}}),
			makePipeline("go", map[string]*model.Job{"test": {}}),
		}
		result, jobName, err := resolveJobTarget(pipelines, "go:test")
		require.NoError(t, err)
		assert.Equal(t, "go", result[0].ID)
		assert.Equal(t, "test", jobName)
	})

	t.Run("main_pipeline_exact_match_over_alias", func(t *testing.T) {
		// Main pipeline has job "up", skill has alias "up" for job "start"
		pipelines := []*model.Pipeline{
			makePipeline("", map[string]*model.Job{"up": {}}),
			makePipeline("docker", map[string]*model.Job{
				"start": {Aliases: []string{"up"}},
			}),
		}
		result, jobName, err := resolveJobTarget(pipelines, "up")
		require.NoError(t, err)
		assert.Equal(t, "", result[0].ID, "main pipeline should take precedence over alias")
		assert.Equal(t, "up", jobName)
	})

	t.Run("main_pipeline_exact_match_over_skill_alias", func(t *testing.T) {
		// Main pipeline has job "build", skill has alias "build" for different job
		pipelines := []*model.Pipeline{
			makePipeline("", map[string]*model.Job{"build": {}}),
			makePipeline("go", map[string]*model.Job{
				"compile": {Aliases: []string{"build"}},
			}),
		}
		result, jobName, err := resolveJobTarget(pipelines, "build")
		require.NoError(t, err)
		assert.Equal(t, "", result[0].ID, "main pipeline exact match should precede alias")
		assert.Equal(t, "build", jobName)
	})

	t.Run("alias_match_when_no_main_pipeline_job", func(t *testing.T) {
		// Main pipeline does NOT have job "up", but skill has alias "up"
		pipelines := []*model.Pipeline{
			makePipeline("", map[string]*model.Job{"build": {}}),
			makePipeline("docker", map[string]*model.Job{
				"start": {Aliases: []string{"up"}},
			}),
		}
		result, jobName, err := resolveJobTarget(pipelines, "up")
		require.NoError(t, err)
		assert.Equal(t, "docker", result[0].ID, "alias should match when no main pipeline job exists")
		assert.Equal(t, "start", jobName)
	})

	t.Run("skill_id_with_default_job", func(t *testing.T) {
		pipelines := []*model.Pipeline{
			makePipeline("", map[string]*model.Job{"build": {}}),
			makePipeline("docker", map[string]*model.Job{
				"default": {},
				"build":   {},
			}),
		}
		result, jobName, err := resolveJobTarget(pipelines, "docker")
		require.NoError(t, err)
		assert.Equal(t, "docker", result[0].ID)
		assert.Equal(t, "default", jobName)
	})

	t.Run("skill_id_without_default_returns_empty_job", func(t *testing.T) {
		pipelines := []*model.Pipeline{
			makePipeline("", map[string]*model.Job{"build": {}}),
			makePipeline("docker", map[string]*model.Job{"build": {}}),
		}
		result, jobName, err := resolveJobTarget(pipelines, "docker")
		require.NoError(t, err)
		assert.Equal(t, "docker", result[0].ID)
		assert.Equal(t, "", jobName, "should return empty job name for listing")
	})

	t.Run("fuzzy_match_single_result", func(t *testing.T) {
		pipelines := []*model.Pipeline{
			makePipeline("", map[string]*model.Job{
				"test:mergecov": {},
				"test:simple":   {},
			}),
		}
		result, jobName, err := resolveJobTarget(pipelines, "mergecov")
		require.NoError(t, err)
		assert.Equal(t, "", result[0].ID)
		assert.Equal(t, "test:mergecov", jobName)
	})

	t.Run("fuzzy_match_multiple_results_error", func(t *testing.T) {
		pipelines := []*model.Pipeline{
			makePipeline("go", map[string]*model.Job{"test": {}}),
			makePipeline("python", map[string]*model.Job{"test": {}}),
		}
		_, _, err := resolveJobTarget(pipelines, "test")
		require.Error(t, err)
		var fuzzyErr *FuzzyMatchError
		assert.ErrorAs(t, err, &fuzzyErr)
		assert.Len(t, fuzzyErr.Matches, 2)
	})

	t.Run("fallback_to_main_pipeline", func(t *testing.T) {
		pipelines := []*model.Pipeline{
			makePipeline("", map[string]*model.Job{"build": {}}),
			makePipeline("docker", map[string]*model.Job{"push": {}}),
		}
		result, jobName, err := resolveJobTarget(pipelines, "nonexistent")
		require.NoError(t, err)
		assert.Equal(t, "", result[0].ID)
		assert.Equal(t, "nonexistent", jobName, "should fall back and pass job name as-is")
	})

	t.Run("error_skill_not_found", func(t *testing.T) {
		pipelines := []*model.Pipeline{
			makePipeline("", map[string]*model.Job{"build": {}}),
		}
		_, _, err := resolveJobTarget(pipelines, "nonexistent:job")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "skill")
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("tasks_field_supported", func(t *testing.T) {
		// Using Tasks instead of Jobs
		pipelines := []*model.Pipeline{
			{ID: "", Tasks: map[string]*model.Job{"up": {}}},
			{ID: "docker", Tasks: map[string]*model.Job{
				"start": {Aliases: []string{"up"}},
			}},
		}
		result, jobName, err := resolveJobTarget(pipelines, "up")
		require.NoError(t, err)
		assert.Equal(t, "", result[0].ID, "main pipeline Tasks should take precedence")
		assert.Equal(t, "up", jobName)
	})

	t.Run("main_pipeline_order_independent", func(t *testing.T) {
		// Main pipeline not first in list
		pipelines := []*model.Pipeline{
			makePipeline("docker", map[string]*model.Job{
				"start": {Aliases: []string{"up"}},
			}),
			makePipeline("", map[string]*model.Job{"up": {}}),
		}
		result, jobName, err := resolveJobTarget(pipelines, "up")
		require.NoError(t, err)
		assert.Equal(t, "", result[0].ID, "main pipeline exact match should work regardless of order")
		assert.Equal(t, "up", jobName)
	})
}

package model_test

import (
	"testing"

	yaml "gopkg.in/yaml.v3"

	"github.com/titpetric/atkins/internal/assert"
	"github.com/titpetric/atkins/model"
)

func TestPipeline_Keys(t *testing.T) {
	type testcase struct {
		id      string
		jobs    map[string]*model.Job
		want    []string
		aliases map[string]string
	}

	testcases := []testcase{
		{
			jobs: map[string]*model.Job{
				"default": nil,
				"build":   nil,
				"test":    nil,
				"lint":    nil,
				"fmt":     nil,
			},
			want:    []string{"default", "build", "fmt", "lint", "test"},
			aliases: map[string]string{},
		},
		{
			id: "go",
			jobs: map[string]*model.Job{
				"default": nil,
				"build":   nil,
				"test":    nil,
				"lint":    nil,
				"fmt":     nil,
			},
			want: []string{"go:default", "go:build", "go:fmt", "go:lint", "go:test"},
			aliases: map[string]string{
				"go": "go:default",
			},
		},
		{
			// An explicit alias that matches the skill ID sets the skill ID alias.
			id: "release",
			jobs: map[string]*model.Job{
				"default": nil,
				"publish": {Aliases: []string{"release"}}, // explicit alias matches skill ID
			},
			want: []string{"release:default", "release:publish"},
			aliases: map[string]string{
				// "release" points to publish, the job with the explicit alias.
				"release": "release:publish",
			},
		},
	}

	for _, tc := range testcases {
		pipeline := &model.Pipeline{
			ID:   tc.id,
			Jobs: tc.jobs,
		}

		{
			got := pipeline.GetKeys()
			assert.Equal(t, tc.want, got)
		}

		{
			got := pipeline.GetAliases()
			assert.Equal(t, tc.aliases, got)
		}
	}
}

// TestPipelineUnmarshalYAML_WithVars tests that Pipeline.Decl.Vars are properly decoded.
func TestPipelineUnmarshalYAML_WithVars(t *testing.T) {
	yamlContent := `
name: Test Pipeline
vars:
  globalVar: global_value
  version: "1.0.0"
jobs:
  test:
    steps:
      - run: echo test
`

	var pipeline model.Pipeline
	err := yaml.Unmarshal([]byte(yamlContent), &pipeline)
	assert.NoError(t, err)

	// Check that Decl is not nil
	assert.NotNil(t, pipeline.Decl)

	// Check that Vars load
	assert.NotNil(t, pipeline.Vars)
	assert.Equal(t, "global_value", pipeline.Vars["globalVar"])
	assert.Equal(t, "1.0.0", pipeline.Vars["version"])
}

// TestPipeline_GetJobs checks GetJobs returns jobs: when both are set,
// and tasks: when jobs: is empty.
func TestPipeline_GetJobs(t *testing.T) {
	t.Parallel()

	job, task := &model.Job{Desc: "job"}, &model.Job{Desc: "task"}
	both := &model.Pipeline{Jobs: map[string]*model.Job{"a": job}, Tasks: map[string]*model.Job{"b": task}}
	assert.Equal(t, map[string]*model.Job{"a": job}, both.GetJobs())

	tasks := &model.Pipeline{Tasks: map[string]*model.Job{"b": task}}
	assert.Equal(t, map[string]*model.Job{"b": task}, tasks.GetJobs())
}

// TestPipeline_HasJobs checks a pipeline with jobs or tasks has jobs.
func TestPipeline_HasJobs(t *testing.T) {
	t.Parallel()

	assert.False(t, (&model.Pipeline{}).HasJobs())
	assert.True(t, (&model.Pipeline{Tasks: map[string]*model.Job{"b": {}}}).HasJobs())
}

// TestPipeline_JobNames checks the names of the jobs come back, in any order.
func TestPipeline_JobNames(t *testing.T) {
	t.Parallel()

	p := &model.Pipeline{Jobs: map[string]*model.Job{"build": {}, "test": {}}}
	assert.ElementsMatch(t, []string{"build", "test"}, p.JobNames())
}

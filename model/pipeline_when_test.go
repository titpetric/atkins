package model_test

import (
	"testing"

	yaml "gopkg.in/yaml.v3"

	"github.com/titpetric/atkins/internal/assert"
	"github.com/titpetric/atkins/model"
)

// TestPipelineWhen_IsEmpty checks a block without files is empty, a nil
// block included.
func TestPipelineWhen_IsEmpty(t *testing.T) {
	t.Parallel()

	var none *model.PipelineWhen
	assert.True(t, none.IsEmpty())
	assert.True(t, (&model.PipelineWhen{Here: true}).IsEmpty())
	assert.False(t, (&model.PipelineWhen{Files: []string{"compose.yml"}}).IsEmpty())
}

// TestPipelineWhen_Decode checks here:, contains: and a job's own when:
// decode alongside args: and an exec: step.
func TestPipelineWhen_Decode(t *testing.T) {
	t.Parallel()

	var pipeline model.Pipeline
	assert.NoError(t, yaml.Unmarshal([]byte(`
when:
  files: [compose.yml]
  here: true
jobs:
  build:
    when:
      files: [compose.yml]
      here: true
      contains: '(?m)^\s+build:'
    cmd: docker compose build
  custom:
    args: true
    steps:
      - exec: [docker, compose]
`), &pipeline))

	assert.Equal(t, &model.PipelineWhen{Files: []string{"compose.yml"}, Here: true}, pipeline.When)

	build := pipeline.Jobs["build"]
	assert.Equal(t, `(?m)^\s+build:`, build.When.Contains)
	assert.True(t, build.When.Here)

	custom := pipeline.Jobs["custom"]
	assert.True(t, custom.Args)
	assert.Equal(t, []string{"docker", "compose"}, custom.Steps[0].Exec)
}

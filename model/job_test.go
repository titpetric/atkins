package model_test

import (
	"os"
	"testing"

	yaml "gopkg.in/yaml.v3"

	"github.com/titpetric/atkins/internal/assert"
	"github.com/titpetric/atkins/model"
)

// TestJobUnmarshalYAML_WithVars tests that Job.Decl.Vars are properly decoded.
func TestJobUnmarshalYAML_WithVars(t *testing.T) {
	yamlContent := `
name: test:run
vars:
  testBinaries: "file1.test\nfile2.test"
  count: 42
steps:
  - run: echo test
`

	var job model.Job
	err := yaml.Unmarshal([]byte(yamlContent), &job)
	assert.NoError(t, err)

	// Check that Decl is not nil
	assert.NotNil(t, job.Decl)

	// Check that Vars load
	assert.NotNil(t, job.Vars)
	assert.Equal(t, "file1.test\nfile2.test", job.Vars["testBinaries"])
	assert.Equal(t, 42, job.Vars["count"])
}

// TestJobUnmarshalYAML_FullDepthDecoding tests full decoding of vars and include in Decl.
func TestJobUnmarshalYAML_FullDepthDecoding(t *testing.T) {
	// Create a temporary include file
	tmpFile, err := os.CreateTemp("", "test-vars-*.yml")
	assert.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, os.Remove(tmpFile.Name()))
	})

	// Write vars to include file
	_, err = tmpFile.WriteString(`
includedVar: included_value
nestedObject:
  key: nested_key_value
`)
	assert.NoError(t, err)
	assert.NoError(t, tmpFile.Close())

	yamlContent := `
name: test:job
vars:
  localVar: local_value
  localCount: 123
include:
  - ` + tmpFile.Name() + `
steps:
  - run: echo test
`

	var job model.Job
	err = yaml.Unmarshal([]byte(yamlContent), &job)
	assert.NoError(t, err)

	// Check vars (at Decl level)
	assert.NotNil(t, job.Vars)
	assert.Equal(t, "local_value", job.Vars["localVar"])
	assert.Equal(t, 123, job.Vars["localCount"])

	// Check include (at Decl level)
	assert.NotNil(t, job.Include)
	assert.Len(t, job.Include.Files, 1)
	assert.Equal(t, tmpFile.Name(), job.Include.Files[0])
}

// TestNestedJobInheritance tests that nested jobs (with ':' in name) properly decode Decl.
func TestNestedJobInheritance(t *testing.T) {
	yamlContent := `
vars:
  nestedVar: nested_value
desc: "A nested job"
steps:
   - run: echo nested
`

	var job model.Job
	err := yaml.Unmarshal([]byte(yamlContent), &job)
	assert.NoError(t, err)

	// Set the name to simulate a nested job
	job.Name = "test:nested:job"

	// Check Decl initialization
	assert.NotNil(t, job.Decl)
	assert.NotNil(t, job.Vars)
	assert.Equal(t, "nested_value", job.Vars["nestedVar"])

	// Check that IsRootLevel correctly identifies nested jobs
	assert.False(t, job.IsRootLevel())

	// Check the root level job detection
	job.Name = "rootjob"
	assert.True(t, job.IsRootLevel())
}

// TestJobUnmarshalYAML_WithEnv tests that Job.Decl.Env is properly decoded.
func TestJobUnmarshalYAML_WithEnv(t *testing.T) {
	yamlContent := `
name: test:job
env:
  vars:
    JOB_ENV_VAR: job_env_value
    GOOS: linux
    GOARCH: amd64
steps:
  - run: echo test
`

	var job model.Job
	err := yaml.Unmarshal([]byte(yamlContent), &job)
	assert.NoError(t, err)

	// Check that Decl is not nil
	assert.NotNil(t, job.Decl)

	// Check that Env loads
	assert.NotNil(t, job.Env)
	assert.NotNil(t, job.Env.Vars)
	assert.Equal(t, "job_env_value", job.Env.Vars["JOB_ENV_VAR"])
	assert.Equal(t, "linux", job.Env.Vars["GOOS"])
	assert.Equal(t, "amd64", job.Env.Vars["GOARCH"])
}

// TestJobUnmarshalYAML_CmdToSyntheticStep tests that job-level cmd creates a synthetic step
func TestJobUnmarshalYAML_CmdToSyntheticStep(t *testing.T) {
	yamlContent := `
desc: Down all containers
cmd: docker compose down
`

	var job model.Job
	err := yaml.Unmarshal([]byte(yamlContent), &job)
	assert.NoError(t, err)

	// Check for the synthetic step
	assert.NotNil(t, job.Steps)
	assert.Len(t, job.Steps, 1)
	assert.Equal(t, "docker compose down", job.Steps[0].Run)
	assert.Equal(t, "docker compose down", job.Steps[0].Name)
	assert.True(t, job.Steps[0].HidePrefix)
	assert.False(t, job.Passthru)
}

// TestJobUnmarshalYAML_RunToSyntheticStep tests that job-level run creates a synthetic step
func TestJobUnmarshalYAML_RunToSyntheticStep(t *testing.T) {
	yamlContent := `
desc: Run tests
run: go test ./...
`

	var job model.Job
	err := yaml.Unmarshal([]byte(yamlContent), &job)
	assert.NoError(t, err)

	// Check for the synthetic step
	assert.NotNil(t, job.Steps)
	assert.Len(t, job.Steps, 1)
	assert.Equal(t, "go test ./...", job.Steps[0].Run)
	assert.False(t, job.Passthru)
}

// TestJobUnmarshalYAML_CmdTakesPrecedenceOverRun tests that cmd takes precedence over run
func TestJobUnmarshalYAML_CmdTakesPrecedenceOverRun(t *testing.T) {
	yamlContent := `
desc: Test precedence
cmd: docker compose up
run: should be ignored
`

	var job model.Job
	err := yaml.Unmarshal([]byte(yamlContent), &job)
	assert.NoError(t, err)

	// Check that the synthetic step uses cmd
	assert.NotNil(t, job.Steps)
	assert.Len(t, job.Steps, 1)
	assert.Equal(t, "docker compose up", job.Steps[0].Run)
}

// TestJobUnmarshalYAML_NoSyntheticStepWithExistingSteps tests no synthetic step when steps exist
func TestJobUnmarshalYAML_NoSyntheticStepWithExistingSteps(t *testing.T) {
	yamlContent := `
desc: Job with steps
cmd: should be ignored
steps:
  - run: echo "explicit step"
`

	var job model.Job
	err := yaml.Unmarshal([]byte(yamlContent), &job)
	assert.NoError(t, err)

	// Check that the job keeps the explicit steps and adds no synthetic one
	assert.Len(t, job.Steps, 1)
	assert.Equal(t, "echo \"explicit step\"", job.Steps[0].Run)
	assert.False(t, job.Passthru) // Stays false with explicit steps
}

// TestJobUnmarshalYAML_NoSyntheticStepWithExistingCmds tests no synthetic step when cmds exist
func TestJobUnmarshalYAML_NoSyntheticStepWithExistingCmds(t *testing.T) {
	yamlContent := `
desc: Job with cmds
cmd: should be ignored
cmds:
  - run: echo "explicit cmd"
`

	var job model.Job
	err := yaml.Unmarshal([]byte(yamlContent), &job)
	assert.NoError(t, err)

	// Check that the job keeps the explicit cmds and adds no synthetic step
	assert.Len(t, job.Cmds, 1)
	assert.Nil(t, job.Steps) // No steps get created
}

// TestJobUnmarshalYAML_StringShorthand tests string shorthand creates synthetic step
func TestJobUnmarshalYAML_StringShorthand(t *testing.T) {
	yamlContent := `docker compose up -d`

	var job model.Job
	err := yaml.Unmarshal([]byte(yamlContent), &job)
	assert.NoError(t, err)

	// Check for the synthetic step
	assert.NotNil(t, job.Steps)
	assert.Len(t, job.Steps, 1)
	assert.Equal(t, "docker compose up -d", job.Steps[0].Run)
	assert.Equal(t, "docker compose up -d", job.Desc)
	assert.False(t, job.Passthru)
}

// TestJob_GetAliases checks a job returns its aliases, and a nil job none.
func TestJob_GetAliases(t *testing.T) {
	t.Parallel()

	var none *model.Job
	assert.Nil(t, none.GetAliases())
	assert.Equal(t, []string{"b"}, (&model.Job{Aliases: []string{"b"}}).GetAliases())
}

// TestJob_ShouldShow checks a root job shows, a nested job hides, and show:
// overrides both.
func TestJob_ShouldShow(t *testing.T) {
	t.Parallel()

	show, hide := true, false
	assert.True(t, (&model.Job{Name: "build"}).ShouldShow())
	assert.False(t, (&model.Job{Name: "go:build"}).ShouldShow())
	assert.True(t, (&model.Job{Name: "go:build", Show: &show}).ShouldShow())
	assert.False(t, (&model.Job{Name: "build", Show: &hide}).ShouldShow())
}

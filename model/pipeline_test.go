package model_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	yaml "gopkg.in/yaml.v3"

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

	// Check that Vars are loaded
	assert.NotNil(t, job.Vars)
	assert.Equal(t, "file1.test\nfile2.test", job.Vars["testBinaries"])
	assert.Equal(t, 42, job.Vars["count"])
}

// TestStepUnmarshalYAML_WithVars tests that Step.Decl.Vars are properly decoded.
func TestStepUnmarshalYAML_WithVars(t *testing.T) {
	yamlContent := `
name: test step
run: echo test
vars:
  myVar: value123
  count: 42
`

	var step model.Step
	err := yaml.Unmarshal([]byte(yamlContent), &step)
	assert.NoError(t, err)

	// Check that Decl is not nil
	assert.NotNil(t, step.Decl)

	// Check that Vars are loaded
	assert.NotNil(t, step.Vars)
	assert.Equal(t, "value123", step.Vars["myVar"])
	assert.Equal(t, 42, step.Vars["count"])
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

	// Check that Vars are loaded
	assert.NotNil(t, pipeline.Vars)
	assert.Equal(t, "global_value", pipeline.Vars["globalVar"])
	assert.Equal(t, "1.0.0", pipeline.Vars["version"])
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

// TestStepUnmarshalYAML_WithInclude tests that Step.Decl.Include is properly decoded.
func TestStepUnmarshalYAML_WithInclude(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test-step-vars-*.yml")
	assert.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, os.Remove(tmpFile.Name()))
	})

	_, err = tmpFile.WriteString(`stepVar: step_var_value`)
	assert.NoError(t, err)
	assert.NoError(t, tmpFile.Close())

	yamlContent := `
name: test step
run: echo test
include:
  - ` + tmpFile.Name()

	var step model.Step
	err = yaml.Unmarshal([]byte(yamlContent), &step)
	assert.NoError(t, err)

	// Check that Decl is not nil
	assert.NotNil(t, step.Decl)

	// Check that Include is loaded
	assert.NotNil(t, step.Include)
	assert.Len(t, step.Include.Files, 1)
	assert.Equal(t, tmpFile.Name(), step.Include.Files[0])
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

	// Check that root level jobs are identified correctly
	job.Name = "rootjob"
	assert.True(t, job.IsRootLevel())
}

// TestStepUnmarshalYAML_WithEnv tests that Step.Decl.Env is properly decoded.
func TestStepUnmarshalYAML_WithEnv(t *testing.T) {
	yamlContent := `
name: test step
run: echo test
env:
  vars:
    MY_VAR: myvalue
    ANOTHER_VAR: another_value
`

	var step model.Step
	err := yaml.Unmarshal([]byte(yamlContent), &step)
	assert.NoError(t, err)

	// Check that Decl is not nil
	assert.NotNil(t, step.Decl)

	// Check that Env is loaded
	assert.NotNil(t, step.Env)
	assert.NotNil(t, step.Env.Vars)
	assert.Equal(t, "myvalue", step.Env.Vars["MY_VAR"])
	assert.Equal(t, "another_value", step.Env.Vars["ANOTHER_VAR"])
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

	// Check that Env is loaded
	assert.NotNil(t, job.Env)
	assert.NotNil(t, job.Env.Vars)
	assert.Equal(t, "job_env_value", job.Env.Vars["JOB_ENV_VAR"])
	assert.Equal(t, "linux", job.Env.Vars["GOOS"])
	assert.Equal(t, "amd64", job.Env.Vars["GOARCH"])
}

// TestStepString_WithRun tests Step.String() for run field
func TestStepString_WithRun(t *testing.T) {
	step := &model.Step{
		Run: "echo hello",
	}
	assert.Equal(t, "run: echo hello", step.String())
}

// TestStepString_WithCmd tests Step.String() for cmd field
func TestStepString_WithCmd(t *testing.T) {
	step := &model.Step{
		Cmd: "make build",
	}
	assert.Equal(t, "cmd: make build", step.String())
}

// TestStepString_WithCmds tests Step.String() for cmds field
func TestStepString_WithCmds(t *testing.T) {
	tests := []struct {
		name     string
		cmds     []string
		expected string
	}{
		{
			name:     "single command in cmds",
			cmds:     []string{"echo test"},
			expected: "cmds: <1 commands>",
		},
		{
			name:     "multiple commands in cmds",
			cmds:     []string{"echo hello", "echo world"},
			expected: "cmds: <2 commands>",
		},
		{
			name:     "three commands in cmds",
			cmds:     []string{"cmd1", "cmd2", "cmd3"},
			expected: "cmds: <3 commands>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			step := &model.Step{
				Cmds: tt.cmds,
			}
			assert.Equal(t, tt.expected, step.String())
		})
	}
}

// TestStepString_WithTask tests Step.String() for task field
func TestStepString_WithTask(t *testing.T) {
	step := &model.Step{
		Task: "build",
	}
	assert.Equal(t, "task: build", step.String())
}

// TestStepString_Priority tests Step.String() field priority
func TestStepString_Priority(t *testing.T) {
	// Task has highest priority
	step := &model.Step{
		Task: "build",
		Run:  "echo hello",
		Cmd:  "make",
		Cmds: []string{"cmd1", "cmd2"},
	}
	assert.Equal(t, "task: build", step.String())

	// Then Run
	step = &model.Step{
		Run:  "echo hello",
		Cmd:  "make",
		Cmds: []string{"cmd1", "cmd2"},
	}
	assert.Equal(t, "run: echo hello", step.String())

	// Then Cmd
	step = &model.Step{
		Cmd:  "make",
		Cmds: []string{"cmd1", "cmd2"},
	}
	assert.Equal(t, "cmd: make", step.String())

	// Finally Cmds
	step = &model.Step{
		Cmds: []string{"cmd1", "cmd2"},
	}
	assert.Equal(t, "cmds: <2 commands>", step.String())
}

// TestStepUnmarshalYAML_WithCmds tests that Step.Cmds are properly decoded
func TestStepUnmarshalYAML_WithCmds(t *testing.T) {
	yamlContent := `
name: multi-command step
cmds:
  - echo "step 1"
  - echo "step 2"
  - echo "step 3"
`

	var step model.Step
	err := yaml.Unmarshal([]byte(yamlContent), &step)
	assert.NoError(t, err)

	// Check that Cmds are loaded
	assert.Equal(t, 3, len(step.Cmds))
	assert.Equal(t, "echo \"step 1\"", step.Cmds[0])
	assert.Equal(t, "echo \"step 2\"", step.Cmds[1])
	assert.Equal(t, "echo \"step 3\"", step.Cmds[2])

	// Check String() representation
	assert.Equal(t, "cmds: <3 commands>", step.String())
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

	// Check that a synthetic step was created
	assert.NotNil(t, job.Steps)
	assert.Len(t, job.Steps, 1)
	assert.Equal(t, "docker compose down", job.Steps[0].Run)
	assert.Equal(t, "docker compose down", job.Steps[0].Name)
	assert.True(t, job.Steps[0].HidePrefix)
	assert.True(t, job.Passthru)
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

	// Check that a synthetic step was created
	assert.NotNil(t, job.Steps)
	assert.Len(t, job.Steps, 1)
	assert.Equal(t, "go test ./...", job.Steps[0].Run)
	assert.True(t, job.Passthru)
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

	// Check that cmd was used for the synthetic step
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

	// Check that the explicit steps are used, not synthetic
	assert.Len(t, job.Steps, 1)
	assert.Equal(t, "echo \"explicit step\"", job.Steps[0].Run)
	assert.False(t, job.Passthru) // Should not be set when using explicit steps
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

	// Check that the explicit cmds are used, not synthetic
	assert.Len(t, job.Cmds, 1)
	assert.Nil(t, job.Steps) // Steps should not be created
}

// TestJobUnmarshalYAML_StringShorthand tests string shorthand creates synthetic step
func TestJobUnmarshalYAML_StringShorthand(t *testing.T) {
	yamlContent := `docker compose up -d`

	var job model.Job
	err := yaml.Unmarshal([]byte(yamlContent), &job)
	assert.NoError(t, err)

	// Check that a synthetic step was created
	assert.NotNil(t, job.Steps)
	assert.Len(t, job.Steps, 1)
	assert.Equal(t, "docker compose up -d", job.Steps[0].Run)
	assert.Equal(t, "docker compose up -d", job.Desc)
	assert.True(t, job.Passthru)
}

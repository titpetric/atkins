package model_test

import (
	"fmt"
	"os"
	"testing"

	yaml "gopkg.in/yaml.v3"

	"github.com/titpetric/atkins/internal/assert"
	"github.com/titpetric/atkins/model"
)

// TestStep_ExecString checks an exec: step renders its argv for display,
// quoting what a reader could take for shell syntax, and labels itself.
func TestStep_ExecString(t *testing.T) {
	t.Parallel()

	step := &model.Step{Exec: []string{"docker", "compose", "; id", ""}}
	assert.Equal(t, `docker compose "; id" ""`, step.ExecString())
	assert.Equal(t, []string{`docker compose "; id" ""`}, step.Commands())
	assert.Equal(t, `exec: docker compose "; id" ""`, step.String())
	assert.Equal(t, `exec: docker compose "; id" ""`, step.DisplayLabel())
	assert.Equal(t, "exec", step.Label(true).Type)
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

	// Check that Vars load
	assert.NotNil(t, step.Vars)
	assert.Equal(t, "value123", step.Vars["myVar"])
	assert.Equal(t, 42, step.Vars["count"])
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

	// Check that Include loads
	assert.NotNil(t, step.Include)
	assert.Len(t, step.Include.Files, 1)
	assert.Equal(t, tmpFile.Name(), step.Include.Files[0])
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

	// Check that Env loads
	assert.NotNil(t, step.Env)
	assert.NotNil(t, step.Env.Vars)
	assert.Equal(t, "myvalue", step.Env.Vars["MY_VAR"])
	assert.Equal(t, "another_value", step.Env.Vars["ANOTHER_VAR"])
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

// TestStepTasks tests splitting a task: reference into individual job names
func TestStepTasks(t *testing.T) {
	tests := []struct {
		name     string
		task     string
		expected []string
	}{
		{"empty", "", []string{}},
		{"single", "build", []string{"build"}},
		{"multiple", "step1 step2 step3", []string{"step1", "step2", "step3"}},
		{"repeated", "build build", []string{"build", "build"}},
		{"extra whitespace", "  lint\ttest  ", []string{"lint", "test"}},
		{"namespaced", "demo:hello :demo:bye", []string{"demo:hello", ":demo:bye"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			step := &model.Step{Task: tt.task}
			assert.Equal(t, tt.expected, step.Tasks())
		})
	}
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

	// Check that Cmds load
	assert.Equal(t, 3, len(step.Cmds))
	assert.Equal(t, "echo \"step 1\"", step.Cmds[0])
	assert.Equal(t, "echo \"step 2\"", step.Cmds[1])
	assert.Equal(t, "echo \"step 3\"", step.Cmds[2])

	// Check String() representation
	assert.Equal(t, "cmds: <3 commands>", step.String())
}

// mockDispatcher records the tasks and commands a step dispatches.
type mockDispatcher struct {
	tasks    []string
	commands []string
}

func TestStep_Dispatch_Task(t *testing.T) {
	step := &model.Step{Task: "build"}
	d := &mockDispatcher{}

	var dispatcher model.StepDispatcher = d
	err := step.Dispatch(dispatcher)

	assert.NoError(t, err)
	assert.Equal(t, []string{"build"}, d.tasks)
	assert.Empty(t, d.commands)
}

func TestStep_Dispatch_SingleCommand(t *testing.T) {
	step := &model.Step{Run: "echo hello"}
	d := &mockDispatcher{}

	err := step.Dispatch(d)

	assert.NoError(t, err)
	assert.Empty(t, d.tasks)
	assert.Equal(t, []string{"echo hello"}, d.commands)
}

func TestStep_Dispatch_MultipleCommands(t *testing.T) {
	step := &model.Step{Cmds: []string{"echo a", "echo b", "echo c"}}
	d := &mockDispatcher{}

	err := step.Dispatch(d)

	assert.NoError(t, err)
	assert.Empty(t, d.tasks)
	assert.Equal(t, []string{"echo a", "echo b", "echo c"}, d.commands)
}

func TestStep_Dispatch_ErrorPropagation(t *testing.T) {
	step := &model.Step{Cmds: []string{"echo a", "echo b"}}
	expectedErr := fmt.Errorf("fail")

	d := &mockDispatcher{}
	errDispatcher := &errorDispatcher{err: expectedErr}

	// Normal dispatch works
	assert.NoError(t, step.Dispatch(d))

	// Error dispatch propagates
	err := step.Dispatch(errDispatcher)
	assert.ErrorIs(t, err, expectedErr)
}

type errorDispatcher struct {
	err error
}

func (m *mockDispatcher) Task(step *model.Step) error {
	m.tasks = append(m.tasks, step.Task)
	return nil
}

func (m *mockDispatcher) Command(step *model.Step, _ int, cmd string) error {
	m.commands = append(m.commands, cmd)
	return nil
}

func (e *errorDispatcher) Task(_ *model.Step) error { return e.err }

func (e *errorDispatcher) Command(*model.Step, int, string) error { return e.err }

// TestStep_IsDeferred checks the step reports its deferred: flag.
func TestStep_IsDeferred(t *testing.T) {
	t.Parallel()

	assert.True(t, (&model.Step{Deferred: true}).IsDeferred())
	assert.False(t, (&model.Step{Run: "echo"}).IsDeferred())
}

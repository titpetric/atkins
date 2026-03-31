package model

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPipeline_Walk_Order(t *testing.T) {
	p := &Pipeline{
		Jobs: map[string]*Job{
			"build":   {Desc: "Build"},
			"default": {Desc: "Default"},
			"test":    {Desc: "Test"},
			"deploy":  {Desc: "Deploy"},
		},
	}

	var visited []string
	err := p.Walk(func(name string, job *Job) error {
		visited = append(visited, name)
		return nil
	})

	require.NoError(t, err)
	assert.Equal(t, []string{"default", "build", "deploy", "test"}, visited)
}

func TestPipeline_Walk_NoDefault(t *testing.T) {
	p := &Pipeline{
		Jobs: map[string]*Job{
			"build": {Desc: "Build"},
			"test":  {Desc: "Test"},
		},
	}

	var visited []string
	err := p.Walk(func(name string, job *Job) error {
		visited = append(visited, name)
		return nil
	})

	require.NoError(t, err)
	assert.Equal(t, []string{"build", "test"}, visited)
}

func TestPipeline_Walk_ErrorPropagation(t *testing.T) {
	p := &Pipeline{
		Jobs: map[string]*Job{
			"a": {},
			"b": {},
		},
	}

	expectedErr := fmt.Errorf("stop")
	err := p.Walk(func(name string, _ *Job) error {
		if name == "a" {
			return expectedErr
		}
		return nil
	})

	assert.ErrorIs(t, err, expectedErr)
}

func TestJob_Walk(t *testing.T) {
	j := &Job{
		Steps: []*Step{
			{Run: "echo hello"},
			{Task: "build"},
			{Cmd: "ls"},
		},
	}

	var indices []int
	var cmds []string
	err := j.Walk(func(i int, step *Step) error {
		indices = append(indices, i)
		cmds = append(cmds, step.String())
		return nil
	})

	require.NoError(t, err)
	assert.Equal(t, []int{0, 1, 2}, indices)
	assert.Len(t, cmds, 3)
}

func TestJob_Walk_NilChildren(t *testing.T) {
	j := &Job{}

	count := 0
	err := j.Walk(func(_ int, _ *Step) error {
		count++
		return nil
	})

	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

type mockDispatcher struct {
	tasks    []string
	commands []string
}

func (m *mockDispatcher) Task(step *Step) error {
	m.tasks = append(m.tasks, step.Task)
	return nil
}

func (m *mockDispatcher) Command(step *Step, _ int, cmd string) error {
	m.commands = append(m.commands, cmd)
	return nil
}

func TestStep_Dispatch_Task(t *testing.T) {
	step := &Step{Task: "build"}
	d := &mockDispatcher{}

	err := step.Dispatch(d)

	require.NoError(t, err)
	assert.Equal(t, []string{"build"}, d.tasks)
	assert.Empty(t, d.commands)
}

func TestStep_Dispatch_SingleCommand(t *testing.T) {
	step := &Step{Run: "echo hello"}
	d := &mockDispatcher{}

	err := step.Dispatch(d)

	require.NoError(t, err)
	assert.Empty(t, d.tasks)
	assert.Equal(t, []string{"echo hello"}, d.commands)
}

func TestStep_Dispatch_MultipleCommands(t *testing.T) {
	step := &Step{Cmds: []string{"echo a", "echo b", "echo c"}}
	d := &mockDispatcher{}

	err := step.Dispatch(d)

	require.NoError(t, err)
	assert.Empty(t, d.tasks)
	assert.Equal(t, []string{"echo a", "echo b", "echo c"}, d.commands)
}

func TestStep_Dispatch_ErrorPropagation(t *testing.T) {
	step := &Step{Cmds: []string{"echo a", "echo b"}}
	expectedErr := fmt.Errorf("fail")

	d := &mockDispatcher{}
	errDispatcher := &errorDispatcher{err: expectedErr}

	// Normal dispatch works
	require.NoError(t, step.Dispatch(d))

	// Error dispatch propagates
	err := step.Dispatch(errDispatcher)
	assert.ErrorIs(t, err, expectedErr)
}

type errorDispatcher struct {
	err error
}

func (e *errorDispatcher) Task(_ *Step) error { return e.err }

func (e *errorDispatcher) Command(*Step, int, string) error { return e.err }

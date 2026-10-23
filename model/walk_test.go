package model_test

import (
	"fmt"
	"testing"

	"github.com/titpetric/atkins/internal/assert"
	"github.com/titpetric/atkins/model"
)

func TestPipeline_Walk_Order(t *testing.T) {
	p := &model.Pipeline{
		Jobs: map[string]*model.Job{
			"build":   {Desc: "Build"},
			"default": {Desc: "Default"},
			"test":    {Desc: "Test"},
			"deploy":  {Desc: "Deploy"},
		},
	}

	var visited []string
	var visit model.PipelineWalkFunc = func(name string, job *model.Job) error {
		visited = append(visited, name)
		return nil
	}

	assert.NoError(t, p.Walk(visit))
	assert.Equal(t, []string{"default", "build", "deploy", "test"}, visited)
}

func TestPipeline_Walk_NoDefault(t *testing.T) {
	p := &model.Pipeline{
		Jobs: map[string]*model.Job{
			"build": {Desc: "Build"},
			"test":  {Desc: "Test"},
		},
	}

	var visited []string
	err := p.Walk(func(name string, job *model.Job) error {
		visited = append(visited, name)
		return nil
	})

	assert.NoError(t, err)
	assert.Equal(t, []string{"build", "test"}, visited)
}

func TestPipeline_Walk_ErrorPropagation(t *testing.T) {
	p := &model.Pipeline{
		Jobs: map[string]*model.Job{
			"a": {},
			"b": {},
		},
	}

	expectedErr := fmt.Errorf("stop")
	err := p.Walk(func(name string, _ *model.Job) error {
		if name == "a" {
			return expectedErr
		}
		return nil
	})

	assert.ErrorIs(t, err, expectedErr)
}

func TestJob_Walk(t *testing.T) {
	j := &model.Job{
		Steps: []*model.Step{
			{Run: "echo hello"},
			{Task: "build"},
			{Cmd: "ls"},
		},
	}

	var indices []int
	var cmds []string
	var visit model.JobWalkFunc = func(i int, step *model.Step) error {
		indices = append(indices, i)
		cmds = append(cmds, step.String())
		return nil
	}

	assert.NoError(t, j.Walk(visit))
	assert.Equal(t, []int{0, 1, 2}, indices)
	assert.Len(t, cmds, 3)
}

func TestJob_Walk_NilChildren(t *testing.T) {
	j := &model.Job{}

	count := 0
	err := j.Walk(func(_ int, _ *model.Step) error {
		count++
		return nil
	})

	assert.NoError(t, err)
	assert.Equal(t, 0, count)
}

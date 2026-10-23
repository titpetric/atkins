package model_test

import (
	"testing"

	"github.com/titpetric/atkins/internal/assert"
	"github.com/titpetric/atkins/model"
)

// TestNewResolvedTask checks the constructor keeps the name, job and pipeline.
func TestNewResolvedTask(t *testing.T) {
	t.Parallel()

	pipeline, job := &model.Pipeline{ID: "go"}, &model.Job{Desc: "build"}
	task := model.NewResolvedTask(pipeline, job, "go:build")
	assert.Equal(t, &model.ResolvedTask{Name: "go:build", Job: job, Pipeline: pipeline}, task)
}

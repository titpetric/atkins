package main

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"testing"

	"github.com/titpetric/atkins/internal/assert"
	"github.com/titpetric/atkins/model"
	"github.com/titpetric/atkins/runner"
)

// TestLintPipelines_JSON checks -l --json prints a lint failure as JSON
// naming each finding and ends the invocation with an error, while an
// inactive skill with a broken reference does not fail the listing.
func TestLintPipelines_JSON(t *testing.T) {
	broken := &model.Pipeline{
		Name: "Broken",
		Path: "/ws/atkins.yml",
		Jobs: map[string]*model.Job{
			"default": {Steps: []*model.Step{{Task: "missing"}}},
		},
	}
	inactive := &model.Pipeline{
		ID:       "composer",
		Name:     "Composer",
		Inactive: true,
		Jobs: map[string]*model.Job{
			"update": {Steps: []*model.Step{{Task: "missing"}}},
		},
	}

	capture := func(pipelines ...*model.Pipeline) (string, bool, error) {
		old := os.Stdout
		r, w, err := os.Pipe()
		assert.NoError(t, err)
		os.Stdout = w

		done, lintErr := lintPipelines(&Options{List: true, JSON: true}, pipelines)

		assert.NoError(t, w.Close())
		os.Stdout = old

		var buf bytes.Buffer
		_, err = io.Copy(&buf, r)
		assert.NoError(t, err)
		return buf.String(), done, lintErr
	}

	out, done, err := capture(broken)
	assert.True(t, done)
	assert.ErrorIs(t, err, io.EOF)

	var result runner.OutputError
	assert.NoError(t, json.Unmarshal([]byte(out), &result), "output: %s", out)
	assert.Equal(t, "pipeline 'Broken' has errors", result.Error)
	assert.Len(t, result.Lint, 1)
	assert.Equal(t, "default", result.Lint[0].Job)
	assert.Equal(t, "/ws/atkins.yml", result.Lint[0].Path)

	out, done, err = capture(inactive)
	assert.False(t, done)
	assert.NoError(t, err)
	assert.Equal(t, "", out)
}

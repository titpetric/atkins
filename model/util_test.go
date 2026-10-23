package model_test

import (
	"testing"

	yaml "gopkg.in/yaml.v3"

	"github.com/titpetric/atkins/internal/assert"
	"github.com/titpetric/atkins/model"
)

// TestDecl_Initialized checks a type embedding *Decl decodes vars:, include:
// and env: into it, and holds an empty Decl when the file sets none.
func TestDecl_Initialized(t *testing.T) {
	t.Parallel()

	var bare model.Step
	assert.NoError(t, yaml.Unmarshal([]byte(`run: echo`), &bare))
	assert.NotNil(t, bare.Decl)

	var step model.Step
	src := "run: echo\nvars: {a: 1}\ninclude: vars.yml\nenv: {vars: {B: two}}\n"
	assert.NoError(t, yaml.Unmarshal([]byte(src), &step))
	assert.Equal(t, map[string]any{"a": 1}, step.Vars)
	assert.Equal(t, []string{"vars.yml"}, step.Include.Files)
	assert.Equal(t, map[string]any{"B": "two"}, step.Env.Vars)

	var bad model.Job
	assert.Error(t, yaml.Unmarshal([]byte("steps: []\nvars: [1]\n"), &bad))
}

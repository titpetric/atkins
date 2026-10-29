package runner_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/titpetric/atkins/internal/assert"
	"github.com/titpetric/atkins/model"
	"github.com/titpetric/atkins/runner"
)

func TestMergeSkillVariables_PreservesExistingKeys(t *testing.T) {
	ctx := &runner.ExecutionContext{
		Variables: runner.NewContextVariables(map[string]any{
			"name":   "from-caller",
			"semver": "v1.0.0",
		}),
		Env:   make(map[string]string),
		Depth: 2, // simulate nested (cross-pipeline) context
	}

	decl := &model.Decl{
		Vars: map[string]any{
			"name":  "from-skill",
			"image": "skillimage/${{ name }}",
			"extra": "new-value",
		},
	}

	err := runner.MergeSkillVariables(ctx, decl)
	assert.NoError(t, err)

	// Existing keys keep their values
	assert.Equal(t, "from-caller", ctx.Variables.Get("name"))
	assert.Equal(t, "v1.0.0", ctx.Variables.Get("semver"))
	// The merge adds new keys from the skill
	assert.Equal(t, "new-value", ctx.Variables.Get("extra"))
	// Interpolated new key uses the skill's own value (not caller's) during resolution,
	// but since "name" existed on stack, the skill's "name" resolved to "from-skill" internally;
	// the final "image" is a new key so it gets set.
	assert.NotNil(t, ctx.Variables.Get("image"))
}

func TestMergeSkillVariables_NilDecl(t *testing.T) {
	ctx := &runner.ExecutionContext{
		Variables: runner.NewContextVariables(map[string]any{"existing": "value"}),
		Env:       make(map[string]string),
	}

	err := runner.MergeSkillVariables(ctx, nil)
	assert.NoError(t, err)
	assert.Equal(t, "value", ctx.Variables.Get("existing"))
}

func TestProcessDecl(t *testing.T) {
	t.Parallel()

	include := filepath.Join(t.TempDir(), "vars.yml")
	assert.NoError(t, os.WriteFile(include, []byte("name: included\nregion: eu\n"), 0o644))

	ctx := &runner.ExecutionContext{
		Variables: runner.NewContextVariables(map[string]any{"env": "prod"}),
		Env:       runner.Env{},
	}
	vars, err := runner.ProcessDecl(ctx, &model.Decl{
		Include: &model.IncludeDecl{Files: []string{include}},
		Vars:    map[string]any{"name": "${{ env }}-app"},
	})
	assert.NoError(t, err)
	assert.Equal(t, "prod-app", vars["name"], "vars set the value of an included name")
	assert.Equal(t, "eu", vars["region"])

	_, err = runner.ProcessDecl(ctx, &model.Decl{Include: &model.IncludeDecl{Files: []string{include + ".missing"}}})
	assert.ErrorContains(t, err, "failed to load vars file")

	vars, err = runner.ProcessDecl(ctx, nil)
	assert.NoError(t, err)
	assert.Empty(t, vars)
}

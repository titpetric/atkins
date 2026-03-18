package runner_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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
	require.NoError(t, err)

	// Existing keys must be preserved
	assert.Equal(t, "from-caller", ctx.Variables.Get("name"))
	assert.Equal(t, "v1.0.0", ctx.Variables.Get("semver"))
	// New keys from skill should be added
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
	require.NoError(t, err)
	assert.Equal(t, "value", ctx.Variables.Get("existing"))
}

package runner_test

import (
	"testing"

	"github.com/titpetric/atkins/internal/assert"
	"github.com/titpetric/atkins/model"
	"github.com/titpetric/atkins/runner"
)

// TestResolver covers the unified graph MergeVariables resolves vars and
// env.vars through when a block declares both: a var reads an env var that
// reads another var.
func TestResolver(t *testing.T) {
	t.Parallel()

	ctx := &runner.ExecutionContext{
		Variables: runner.NewContextVariables(nil),
		Env:       runner.Env{},
	}
	decl := &model.Decl{
		Vars: map[string]any{
			"name":     "atkins",
			"greeting": "$(echo hello $TARGET)",
		},
		Env: &model.EnvDecl{Vars: map[string]any{
			"TARGET": "${{ name }}",
		}},
	}

	assert.NoError(t, runner.MergeVariables(ctx, decl))
	assert.Equal(t, "atkins", ctx.Env["TARGET"])
	assert.Equal(t, "hello atkins", ctx.Variables.Get("greeting"))
}

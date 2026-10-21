package model_test

import (
	"testing"

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

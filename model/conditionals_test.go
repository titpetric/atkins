package model_test

import (
	"testing"

	"github.com/titpetric/atkins/internal/assert"
	"github.com/titpetric/atkins/model"
)

// TestConditionals_String checks one condition prints as written and more
// than one prints each in parentheses, joined with &&.
func TestConditionals_String(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "", model.Conditionals{}.String())
	assert.Equal(t, "a == 1", model.Conditionals{" a == 1 "}.String())
	assert.Equal(t, "(a == 1) && (b)", model.Conditionals{"a == 1", "b"}.String())
}

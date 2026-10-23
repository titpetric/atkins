package main

import (
	"testing"

	"github.com/titpetric/atkins/internal/assert"
)

// The configuration resolves once and every later call shares it.
func TestLoadRuntimeConfig(t *testing.T) {
	t.Parallel()

	first, err := loadRuntimeConfig(t.TempDir())
	assert.NoError(t, err)
	assert.True(t, first != nil)

	second, err := loadRuntimeConfig(t.TempDir())
	assert.NoError(t, err)
	assert.True(t, first == second)
}

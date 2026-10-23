package config_test

import (
	"strings"
	"testing"

	"github.com/titpetric/atkins/config"
	"github.com/titpetric/atkins/internal/assert"
)

func TestRuntimeDefaultConfig(t *testing.T) {
	t.Parallel()

	// The document explains itself to the user who opens it first.
	assert.True(t, strings.HasPrefix(string(config.RuntimeDefaultConfig), "# atkins configuration."))
	assert.Contains(t, string(config.RuntimeDefaultConfig), "version: 1\n")
}

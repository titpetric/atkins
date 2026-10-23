package main

import (
	"bytes"
	"testing"

	"github.com/titpetric/atkins/internal/assert"
)

// --help anywhere on the command line writes the help document and runs nothing.
func TestStartHelp(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	assert.NoError(t, start(&out, []string{"go:fmt", "--help"}))
	assert.Contains(t, out.String(), "Usage: atkins [flags] [job...]")
}

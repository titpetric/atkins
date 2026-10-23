package main

import (
	"bytes"
	"testing"

	"github.com/titpetric/atkins/internal/assert"
)

// The build variables reach the help document's version line.
func TestVersionInHelp(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	assert.NoError(t, WriteHelp(&out))
	assert.Contains(t, out.String(), "Version "+Version+", commit "+Commit+".")
}

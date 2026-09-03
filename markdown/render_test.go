package markdown

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/titpetric/atkins/colors"
)

const renderDoc = "# Title\n\n" + sampleTable + "\n\nA trailing paragraph.\n"

// TestRenderNoColor checks the document is returned byte for byte when
// color is false, which is what a file or a pipe gets.
func TestRenderNoColor(t *testing.T) {
	assert.Equal(t, renderDoc, Render(renderDoc, false))
}

// TestRenderColor checks the table is redrawn with borders and the rest
// of the document is untouched.
func TestRenderColor(t *testing.T) {
	out := Render(renderDoc, true)

	assert.Contains(t, out, "# Title")
	assert.Contains(t, out, "A trailing paragraph.")
	assert.Contains(t, out, "┌")
	assert.Contains(t, out, "┐")
	assert.Contains(t, out, "└")
	assert.Contains(t, out, "┘")
	assert.NotContains(t, out, "| Flag")
	assert.NotContains(t, out, "`--list`")

	// The plain text survives under the colour codes.
	assert.Contains(t, colors.StripANSI(out), "--list")
	assert.Contains(t, colors.StripANSI(out), "List jobs")
}

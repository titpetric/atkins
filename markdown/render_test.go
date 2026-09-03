package markdown

import (
	"strings"
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

// TestRenderColor checks the table is redrawn with borders, the heading
// is coloured, and the rest of the document is untouched.
func TestRenderColor(t *testing.T) {
	out := Render(renderDoc, true)

	assert.Contains(t, out, colors.BrightCyan("# Title"))
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

// TestRenderColorizesSubheading checks a heading below level 1 is
// coloured bright white rather than bright cyan, matching helpdoc's own
// heading() scheme.
func TestRenderColorizesSubheading(t *testing.T) {
	out := Render("## A job\n\nProse.\n", true)

	assert.Contains(t, out, colors.BrightWhite("## A job"))
}

// TestRenderDoesNotDoubleColorHeading checks a heading the caller already
// coloured, as helpdoc's own headings are, is left alone: Sections
// isolates it same as a plain one, but it no longer starts with `#` so
// the heading regexp does not match it.
func TestRenderDoesNotDoubleColorHeading(t *testing.T) {
	coloured := colors.BrightCyan("# Title")
	out := Render(coloured+"\n\nProse.\n", true)

	assert.Equal(t, coloured, out[:len(coloured)])
}

// TestRenderPreservesTrailingNewline checks a document that ends with a
// newline still ends with exactly one after Render redraws its
// sections and rejoins them: helpdoc.Write's buffer always ends with
// one, and `atkins --help` in a terminal should not lose it.
func TestRenderPreservesTrailingNewline(t *testing.T) {
	out := Render(renderDoc, true)
	assert.True(t, strings.HasSuffix(out, "\n"))

	noTrailing := strings.TrimSuffix(renderDoc, "\n")
	out = Render(noTrailing, true)
	assert.False(t, strings.HasSuffix(out, "\n"))
}

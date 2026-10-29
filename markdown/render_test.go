package markdown

import (
	"strings"
	"testing"

	"github.com/titpetric/atkins/colors"
	"github.com/titpetric/atkins/internal/assert"
)

const renderDoc = "# Title\n\n" + sampleTable + "\n\nA trailing paragraph.\n"

// TestRenderNoColor checks Render returns the document byte for byte when
// color is false, as for a file or a pipe.
func TestRenderNoColor(t *testing.T) {
	assert.Equal(t, renderDoc, Render(renderDoc, false))
}

// TestRenderColor checks Render redraws the table with borders, colours
// the heading, and leaves the rest of the document alone.
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

// TestRenderColorizesSubheading checks Render colours a heading below
// level 1 bright white, matching helpdoc's own heading() scheme where
// level 1 gets bright cyan.
func TestRenderColorizesSubheading(t *testing.T) {
	out := Render("## A job\n\nProse.\n", true)

	assert.Contains(t, out, colors.BrightWhite("## A job"))
}

// TestRenderDoesNotDoubleColorHeading checks Render leaves alone a heading
// the caller already coloured, as helpdoc does with its own headings:
// Sections isolates it same as a plain one, but it starts with an escape
// code. The heading regexp needs a leading `#` and skips it.
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

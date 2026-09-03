package markdown

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSections(t *testing.T) {
	doc := "# Heading\n\nParagraph one.\n\nParagraph two,\nsecond line.\n"

	sections := Sections(doc)

	assert.Equal(t, []string{
		"# Heading",
		"Paragraph one.",
		"Paragraph two,\nsecond line.",
	}, sections)
}

// TestSectionsKeepsFence checks a blank line inside a fenced code block
// does not split the fence into two sections.
func TestSectionsKeepsFence(t *testing.T) {
	doc := "before\n\n```yaml\njobs:\n\n  default: {}\n```\n\nafter\n"

	sections := Sections(doc)

	assert.Equal(t, []string{
		"before",
		"```yaml\njobs:\n\n  default: {}\n```",
		"after",
	}, sections)
}

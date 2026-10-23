package markdown

import (
	"strings"
)

// Sections splits a document into the blocks a blank line separates, keeping fences whole.
func Sections(doc string) []string {
	lines := strings.Split(doc, "\n")

	var sections []string
	var current []string
	inFence := false

	flush := func() {
		if len(current) == 0 {
			return
		}
		sections = append(sections, strings.Join(current, "\n"))
		current = nil
	}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "```") {
			inFence = !inFence
			current = append(current, line)
			continue
		}

		if !inFence && trimmed == "" {
			flush()
			continue
		}

		current = append(current, line)
	}
	flush()

	return sections
}

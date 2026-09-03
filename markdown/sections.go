// Package markdown finds structure in a rendered markdown document without
// a full parser: the blank-line-separated blocks a document is already
// written in, and the GFM pipe tables among them.
//
// It exists for one caller: `atkins --help` builds one long markdown
// document, and a table in it reads better in a terminal as a bordered,
// coloured grid than as pipe-delimited text. Sections splits the document
// so a table block can be found and replaced without disturbing the
// prose around it; Render does the replacing.
package markdown

import "strings"

// Sections splits a document into the blocks a blank line separates.
//
// A fenced code block (opened and closed by a line starting with ```) is
// kept as one section even when it contains a blank line, so a YAML
// example inside `atkins --help` is never split mid-fence.
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

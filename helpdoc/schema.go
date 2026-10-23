package helpdoc

import (
	"strings"

	_ "embed"
)

// schema is schema.md, the hand-written atkins.yml reference.
//
//go:embed schema.md
var schema string

// Schema returns the atkins.yml reference as markdown, without a heading.
func Schema() string {
	return strings.TrimSpace(schema)
}

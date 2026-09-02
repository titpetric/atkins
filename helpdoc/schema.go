package helpdoc

import (
	"strings"

	_ "embed"
)

// schema.md is written by hand, not generated. It is a reduction of the
// JSON Schema `splint -schema -i model` prints: the keys an author
// types, in the spellings the model's UnmarshalYAML methods accept,
// without the Go field names and the display-only fields a pipeline file
// cannot set. Regenerating it from the schema would lose all of that.
//
// TestSchemaDocumentsModelKeys keeps it honest: it reads the yaml tags
// off model.Pipeline, model.Job and model.Step and fails when this file
// documents a key none of them declares.
//
//go:embed schema.md
var schema string

// Schema returns the atkins.yml reference as markdown, without a heading
// of its own.
func Schema() string {
	return strings.TrimSpace(schema)
}

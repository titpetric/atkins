// Package helpdoc renders `atkins --help`.
//
// The help reads as a document: the usage line, the commands, every flag
// with its default, the skills installed on this machine with the jobs
// each contributes, and a reference for the atkins.yml schema. A person
// or an agent reading it can run atkins and write a pipeline without
// opening a second page. The section order lets a reader stop early: what
// to type, then the flags, then the jobs the installed skills add, then
// the schema for writing a pipeline of your own.
//
// Write builds the document as plain markdown throughout, with padded
// table cells so it reads as columns in a file too. Write adds colour -
// on the headings, and on every table, which the markdown package redraws
// with borders - only when the caller says the writer is a terminal, so
// `atkins --help > help.md` is a document and not a screenshot of one.
//
// A person writes schema.md by hand; no generator touches it. It reduces
// the JSON Schema `splint -schema -i model` prints: the keys an author
// types, in the spellings the model's UnmarshalYAML methods accept,
// without the Go field names and the display-only fields a pipeline file
// cannot set. Regenerating it from the schema would lose that.
// TestSchemaDocumentsModelKeys keeps it honest: it reads the yaml tags off
// model.Pipeline, model.Job and model.Step and fails when schema.md
// documents a key none of them declares.
package helpdoc
